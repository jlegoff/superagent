package otelcol

import (
	"context"
	"fmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/oklog/ulid/v2"
	"github.com/open-telemetry/opamp-go/client/types"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"superagent/opamp"
	"superagent/supervisor"
	"sync/atomic"
	"time"
)

type OtelCol struct {
	DataDir  string
	LogDir   string
	BinPath  string
	Name     string
	OpampUrl string
	ApiKey   string
}

type Supervisor struct {
	Config      OtelCol
	Commander   *Commander
	OpampClient *opamp.Client
	Logger      types.Logger
	InstanceId  ulid.ULID
	// Final effective config of the Collector.
	EffectiveConfig atomic.Value

	// A channel to indicate there is a new config to apply.
	hasNewConfig chan struct{}
}

func NewOtelCol(name string, dataDir string, logDir string, binPath string, opampUrl string, apiKey string) *OtelCol {
	return &OtelCol{Name: name, DataDir: dataDir, LogDir: logDir, BinPath: binPath, OpampUrl: opampUrl, ApiKey: apiKey}
}

func (otelcol *OtelCol) GetType() string {
	return "otelcol"
}

func (otelcol *OtelCol) GetName() string {
	return otelcol.Name
}

func (otelcol *OtelCol) GetSupervisor() supervisor.Supervisor {
	logger := &supervisor.Logger{Logger: log.Default()}
	return &Supervisor{Config: *otelcol, Logger: logger, hasNewConfig: make(chan struct{}, 1)}
}

func (s *Supervisor) Start() error {
	var err error
	s.InstanceId, err = supervisor.GetOrCreateInstanceId(s.Config.DataDir)
	if err != nil {
		return err
	}

	commander, err := NewCommander(s.Logger, s.Config.BinPath, s.getConfigPaths()...)
	if err != nil {
		return err
	}
	s.Commander = commander

	opampClient := opamp.NewOpampClient(
		opamp.Config{
			OpampUrl: s.Config.OpampUrl,
			ApiKey:   s.Config.ApiKey,
		},
		s,
		s.Logger)

	s.OpampClient = &opampClient
	err = s.OpampClient.StartOpAMP()
	if err != nil {
		return fmt.Errorf("cannot start the opamp client %s", err)
	}

	go s.runAgentProcess()
	return nil
}

func (s *Supervisor) getEffectiveConfigFilePath() string {
	s.Logger.Debugf("config path %s", filepath.Join(s.Config.DataDir, "effective.yaml"))
	return filepath.Join(s.Config.DataDir, "effective.yaml")
}

func (s *Supervisor) runAgentProcess() {
	if _, err := os.Stat(s.getEffectiveConfigFilePath()); err == nil {
		// We have an effective config file saved previously. Use it to start the agent.
		s.startAgent()
	}

	restartTimer := time.NewTimer(0)
	restartTimer.Stop()

	for {
		select {
		case <-s.hasNewConfig:
			restartTimer.Stop()
			s.applyConfigWithAgentRestart()

		case <-s.Commander.Done():
			errMsg := fmt.Sprintf(
				"Agent process PID=%d exited unexpectedly, exit code=%d. Will restart in a bit...",
				s.Commander.Pid(), s.Commander.ExitCode(),
			)
			s.Logger.Debugf(errMsg)
			s.OpampClient.SetUnhealthy(errMsg)

			// Wait 5 seconds before starting again.
			restartTimer.Stop()
			restartTimer.Reset(5 * time.Second)

		case <-restartTimer.C:
			s.startAgent()
		}
	}
}

func (s *Supervisor) applyConfigWithAgentRestart() {
	s.Logger.Debugf("Restarting the agent with the new config.")
	cfg := s.EffectiveConfig.Load().(string)
	err := s.Commander.Stop(context.Background())
	if err != nil {
		s.Logger.Errorf("cannot stop agent %v", err)
	}
	s.writeEffectiveConfigToFile(cfg)
	s.startAgent()
}

func (s *Supervisor) startAgent() {
	err := s.Commander.Start(context.Background())
	if err != nil {
		errMsg := fmt.Sprintf("Cannot start the agent: %v", err)
		s.Logger.Errorf(errMsg)
		s.OpampClient.SetUnhealthy(errMsg)
		return
	}
	s.OpampClient.SetHealthy(time.Now())
}

func (s *Supervisor) writeEffectiveConfigToFile(cfg string) {
	f, err := os.Create(s.getEffectiveConfigFilePath())
	if err != nil {
		s.Logger.Errorf("Cannot write effective config file: %v", err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			s.Logger.Errorf("Error closing file %v", err)
		}
	}()

	_, err = f.WriteString(cfg)
	if err != nil {
		s.Logger.Errorf("Error writing effective config %v", err)
	}
}

func (s *Supervisor) getConfigPaths() []string {
	return []string{filepath.Join(s.Config.DataDir, "configuration", "otelcol.yaml")}
}

func (s *Supervisor) Stop() error {
	return s.Commander.Stop(context.Background())
}

func (s *Supervisor) Setup() error {
	err := supervisor.EnsureDirExists(s.Config.DataDir)
	if err != nil {
		return err
	}
	err = supervisor.EnsureDirExists(s.Config.LogDir)
	if err != nil {
		return err
	}
	return nil
}

func (s *Supervisor) GetAgentDescription() opamp.Agent {
	hostName, err := os.Hostname()
	if err != nil {
		s.Logger.Errorf("Could not get hostname: %s", err)
	}
	host := opamp.Host{
		Name: hostName,
		Id:   hostName,
	}

	operatingSystem := opamp.Os{
		Type: runtime.GOOS,
	}

	service := opamp.Service{
		Name:    "io.opentelemetry.collector",
		Version: "0.0.1",
	}
	return opamp.Agent{
		InstanceId: s.InstanceId,
		Host:       host,
		Os:         operatingSystem,
		Service:    service,
	}
}

func (s *Supervisor) GetEffectiveConfigMap() map[string]opamp.ConfigFile {
	return make(map[string]opamp.ConfigFile)
}

func (s *Supervisor) ApplyRemoteConfig(ctx context.Context, config opamp.RemoteConfig) {
	configChanged, err := s.composeEffectiveConfig(config)
	if err != nil {
		s.OpampClient.SetRemoteConfigError(config.Hash, err.Error())
	} else {
		s.OpampClient.SetRemoteConfigApplied(config.Hash)
	}

	if configChanged {
		s.OpampClient.SetRemoteConfig(ctx)
		s.Logger.Debugf("Config is changed. Signal to restart the agent.")
		// Signal that there is a new config.
		select {
		case s.hasNewConfig <- struct{}{}:
		default:
		}
	}

}

func (s *Supervisor) composeEffectiveConfig(config opamp.RemoteConfig) (configChanged bool, err error) {
	var k = koanf.New(".")

	// Begin with empty config. We will merge received configs on top of it.
	if err := k.Load(rawbytes.Provider([]byte{}), yaml.Parser()); err != nil {
		return false, err
	}

	// Sort to make sure the order of merging is stable.
	var names []string
	for name := range config.Configs {
		if name == "" {
			continue
		}
		names = append(names, name)
	}

	sort.Strings(names)

	// Append instance config as the last item.
	names = append(names, "")

	// Merge received configs.
	for _, name := range names {
		if name != "" {
			s.Logger.Debugf("Applying remote configuration %s", name)
		}
		item := config.Configs[name]
		var k2 = koanf.New(".")
		err := k2.Load(rawbytes.Provider([]byte(item.Content)), yaml.Parser())
		if err != nil {
			return false, fmt.Errorf("cannot parse config named %s: %v", name, err)
		}
		err = k.Merge(k2)
		if err != nil {
			return false, fmt.Errorf("cannot merge config named %s: %v", name, err)
		}
	}

	// The merged final result is our effective config.
	effectiveConfigBytes, err := k.Marshal(yaml.Parser())
	if err != nil {
		return false, err
	}

	// Check if effective config is changed.
	newEffectiveConfig := string(effectiveConfigBytes)
	configChanged = false
	if (s.EffectiveConfig.Load() == nil) || (s.EffectiveConfig.Load().(string) != newEffectiveConfig) {
		s.Logger.Debugf("Effective config changed.")
		s.EffectiveConfig.Store(newEffectiveConfig)
		configChanged = true
	}

	return configChanged, nil
}
