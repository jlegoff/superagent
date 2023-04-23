package otelcol

import (
	"context"
	"fmt"
	"github.com/oklog/ulid/v2"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	Config    OtelCol
	Commander *Commander
	// The OpAMP client to connect to the OpAMP Server.
	OpampClient client.OpAMPClient
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

func (supervisor *OtelColSupervisor) Start() error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	commander, err := NewCommander(logger, supervisor.Config.BinPath, supervisor.getConfigPaths()...)
	if err != nil {
		return err
	}
	supervisor.Commander = commander
	err = supervisor.Commander.Start(context.Background())
	if err != nil {
		return fmt.Errorf("Cannot start the agent: %s", err)
	}
	err = supervisor.startOpAMP()
	if err != nil {
		return fmt.Errorf("Cannot start the opamp server %s", err)
	}
	return nil
}

func (supervisor *OtelColSupervisor) getConfigPaths() []string {
	return []string{filepath.Join(supervisor.Config.DataDir, "configuration", "otelcol.yaml")}
}

func (supervisor *OtelColSupervisor) Stop() error {
	return supervisor.Commander.Stop(context.Background())
}

func (sup *OtelColSupervisor) Setup() error {
	err := supervisor.EnsureDirExists(sup.Config.DataDir)
	if err != nil {
		return err
	}
	err = supervisor.EnsureDirExists(sup.Config.LogDir)
	if err != nil {
		return err
	}
	return nil
}

func (s *OtelColSupervisor) startOpAMP() error {
	s.OpampClient = client.NewHTTP(s.Logger)

	settings := types.StartSettings{
		OpAMPServerURL: s.Config.OpampUrl,
		InstanceUid:    s.InstanceId.String(),
		Header:         http.Header{"api-key": {s.Config.ApiKey}},
		Callbacks: types.CallbacksStruct{
			OnConnectFunc: func() {
				s.Logger.Debugf("Connected to the server.")
			},
			OnConnectFailedFunc: func(err error) {
				s.Logger.Errorf("Failed to connect to the server: %v", err)
			},
			OnErrorFunc: func(err *protobufs.ServerErrorResponse) {
				s.Logger.Errorf("Server returned an error response: %v", err.ErrorMessage)
			},
			GetEffectiveConfigFunc: func(ctx context.Context) (*protobufs.EffectiveConfig, error) {
				return s.createEffectiveConfigMsg(), nil
			},
			OnMessageFunc: s.onMessage,
		},
		Capabilities: protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsRemoteConfig |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsOwnMetrics |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsHealth,
	}
	err := s.OpampClient.SetAgentDescription(s.createAgentDescription())
	if err != nil {
		return err
	}

	err = s.OpampClient.SetHealth(&protobufs.AgentHealth{Healthy: false})
	if err != nil {
		return err
	}

	s.Logger.Debugf("Starting OpAMP client...")

	err = s.OpampClient.Start(context.Background(), settings)
	if err != nil {
		return err
	}

	s.Logger.Debugf("OpAMP Client started.")

	return nil
}

func (s *OtelColSupervisor) createAgentDescription() *protobufs.AgentDescription {
	hostname, _ := os.Hostname()

	return &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			keyVal("service.name", "io.opentelemetry.collector"),
			keyVal("service.version", "0.0.1"),
		},
		NonIdentifyingAttributes: []*protobufs.KeyValue{
			keyVal("os.type", runtime.GOOS),
			keyVal("host.name", hostname),
		},
	}
}

func keyVal(key, val string) *protobufs.KeyValue {
	return &protobufs.KeyValue{
		Key: key,
		Value: &protobufs.AnyValue{
			Value: &protobufs.AnyValue_StringValue{StringValue: val},
		},
	}
}

func (s *OtelColSupervisor) createEffectiveConfigMsg() *protobufs.EffectiveConfig {
	cfg := &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{},
		},
	}
	return cfg
}

func (s *OtelColSupervisor) onMessage(ctx context.Context, msg *types.MessageData) {

}
