package otelcol

import (
	"context"
	"fmt"
	"github.com/oklog/ulid/v2"
	"github.com/open-telemetry/opamp-go/client/types"
	"go.uber.org/zap"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"superagent/opamp"
	"superagent/supervisor"
)

type OtelCol struct {
	DataDir  string
	LogDir   string
	BinPath  string
	Name     string
	OpampUrl string
	ApiKey   string
}

type OtelColSupervisor struct {
	Config      OtelCol
	Commander   *Commander
	OpampClient *opamp.Client
	Logger      types.Logger
	InstanceId  ulid.ULID
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

func (collector *OtelCol) GetSupervisor() supervisor.Supervisor {
	logger := &supervisor.Logger{Logger: log.Default()}
	return &OtelColSupervisor{Config: *collector, Logger: logger}
}

func (s *OtelColSupervisor) Start() error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}

	s.InstanceId, err = supervisor.GetOrCreateInstanceId(s.Config.DataDir)
	if err != nil {
		return err
	}

	commander, err := NewCommander(logger, s.Config.BinPath, s.getConfigPaths()...)
	if err != nil {
		return err
	}
	s.Commander = commander
	err = s.Commander.Start(context.Background())
	if err != nil {
		return fmt.Errorf("Cannot start the agent: %s", err)
	}

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
		return fmt.Errorf("Cannot start the opamp client %s", err)
	}
	return nil
}

func (s *OtelColSupervisor) getConfigPaths() []string {
	return []string{filepath.Join(s.Config.DataDir, "configuration", "otelcol.yaml")}
}

func (s *OtelColSupervisor) Stop() error {
	return s.Commander.Stop(context.Background())
}

func (s *OtelColSupervisor) Setup() error {
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

func (s *OtelColSupervisor) GetAgentDescription() opamp.Agent {
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

func (s *OtelColSupervisor) GetEffectiveConfigMap() map[string]opamp.ConfigFile {
	return make(map[string]opamp.ConfigFile)
}

func (s *OtelColSupervisor) ApplyRemoteConfig(context.Context, opamp.RemoteConfig) {
}
