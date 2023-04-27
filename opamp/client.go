package opamp

import (
	"context"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"net/http"
	"time"
)

type Config struct {
	OpampUrl string
	ApiKey   string
}

type Client struct {
	Config      Config
	OpampClient client.OpAMPClient
	Supervisor  *Supervisor
	Logger      types.Logger
}

type Supervisor interface {
	GetAgentDescription() Agent
	GetEffectiveConfigMap() map[string]ConfigFile
	ApplyRemoteConfig(context.Context, RemoteConfig)
}

func NewOpampClient(config Config, sup Supervisor, logger types.Logger) Client {
	return Client{
		Config:     config,
		Supervisor: &sup,
		Logger:     logger,
	}
}

func (c *Client) StartOpAMP() error {
	c.OpampClient = client.NewHTTP(c.Logger)

	settings := types.StartSettings{
		OpAMPServerURL: c.Config.OpampUrl,
		InstanceUid:    (*c.Supervisor).GetAgentDescription().InstanceId.String(),
		Header:         http.Header{"api-key": {c.Config.ApiKey}},
		Callbacks: types.CallbacksStruct{
			OnConnectFunc: func() {
				c.Logger.Debugf("Connected to the server.")
			},
			OnConnectFailedFunc: func(err error) {
				c.Logger.Errorf("Failed to connect to the server: %v", err)
			},
			OnErrorFunc: func(err *protobufs.ServerErrorResponse) {
				c.Logger.Errorf("Server returned an error response: %v", err.ErrorMessage)
			},
			GetEffectiveConfigFunc: func(ctx context.Context) (*protobufs.EffectiveConfig, error) {
				return c.createEffectiveConfigMsg(), nil
			},
			OnMessageFunc: c.onMessage,
		},
		Capabilities: protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsRemoteConfig |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsOwnMetrics |
			protobufs.AgentCapabilities_AgentCapabilities_ReportsHealth,
	}
	err := c.OpampClient.SetAgentDescription(c.createAgentDescription())
	if err != nil {
		return err
	}

	err = c.OpampClient.SetHealth(&protobufs.AgentHealth{Healthy: false})
	if err != nil {
		return err
	}

	c.Logger.Debugf("Starting OpAMP client...")

	err = c.OpampClient.Start(context.Background(), settings)
	if err != nil {
		return err
	}

	c.Logger.Debugf("OpAMP Client started.")

	return nil
}

func (c *Client) createAgentDescription() *protobufs.AgentDescription {
	agent := (*c.Supervisor).GetAgentDescription()

	return &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			keyVal("service.name", agent.Service.Name),
			keyVal("service.version", agent.Service.Version),
			keyVal("service.instance.id", agent.InstanceId.String()),
		},
		NonIdentifyingAttributes: []*protobufs.KeyValue{
			keyVal("os.type", agent.Os.Type),
			keyVal("os.version", agent.Os.Version),
			keyVal("host.id", agent.Host.Id),
			keyVal("host.name", agent.Host.Name),
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

func (c *Client) createEffectiveConfigMsg() *protobufs.EffectiveConfig {
	configFile := make(map[string]*protobufs.AgentConfigFile)
	for name, config := range (*c.Supervisor).GetEffectiveConfigMap() {
		configFile[name] = &protobufs.AgentConfigFile{
			Body:        []byte(config.Content),
			ContentType: config.ContentType}
	}
	cfg := &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: configFile,
		},
	}
	return cfg
}

func (c *Client) onMessage(ctx context.Context, msg *types.MessageData) {
	if msg.RemoteConfig != nil {
		c.Logger.Debugf("Received remote config from server, hash=%x.", msg.RemoteConfig.ConfigHash)
		configMap := make(map[string]ConfigFile)
		for name, config := range msg.RemoteConfig.Config.ConfigMap {
			configMap[name] = ConfigFile{
				Content:     string(config.Body),
				ContentType: config.ContentType,
			}
		}
		remoteConfig := RemoteConfig{
			Configs: configMap,
			Hash:    string(msg.RemoteConfig.ConfigHash),
		}
		(*c.Supervisor).ApplyRemoteConfig(ctx, remoteConfig)
	}
}

func (c *Client) SetUnhealthy(lastError string) {
	err := c.OpampClient.SetHealth(&protobufs.AgentHealth{Healthy: false, LastError: lastError})
	if err != nil {
		c.Logger.Errorf("cannot set health %v", err)
	}
}

func (c *Client) SetHealthy(startTime time.Time) {
	err := c.OpampClient.SetHealth(&protobufs.AgentHealth{Healthy: true, StartTimeUnixNano: uint64(startTime.UnixNano())})
	if err != nil {
		c.Logger.Errorf("cannot set health %v", err)
	}
}

func (c *Client) SetRemoteConfigError(lastHash string, errorMessage string) {
	err := c.OpampClient.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
		LastRemoteConfigHash: []byte(lastHash),
		Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED,
		ErrorMessage:         errorMessage,
	})
	if err != nil {
		c.Logger.Errorf("cannot set remote config error %v", err)
	}
}

func (c *Client) SetRemoteConfigApplied(lastHash string) {
	err := c.OpampClient.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
		LastRemoteConfigHash: []byte(lastHash),
		Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
	})
	if err != nil {
		c.Logger.Errorf("cannot set remote config status %v", err)
	}
}

func (c *Client) SetRemoteConfig(ctx context.Context) {
	err := c.OpampClient.UpdateEffectiveConfig(ctx)
	if err != nil {
		c.Logger.Errorf("cannot set remote config %v", err)
	}
}
