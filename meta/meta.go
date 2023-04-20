package meta

import (
	"fmt"
	"path/filepath"
	"superagent/otelcol"
)

type MetaAgent struct {
	config      Meta
	supervisors map[string]Supervisor
}

type Supervisor interface {
	Start() error
	Stop() error
}

func NewMetaAgent(configPath string) (*MetaAgent, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return &MetaAgent{config: *config}, nil
}

func (m *MetaAgent) Start() error {
	m.supervisors = make(map[string]Supervisor)
	for _, agentConfig := range m.config.Agents {
		err := EnsureDirExists(m.getDataDir(agentConfig.GetName(), agentConfig.GetType()))
		if err != nil {
			return err
		}
		err = EnsureDirExists(m.getLogDir(agentConfig.GetName(), agentConfig.GetType()))
		if err != nil {
			return err
		}
		sup, err := m.getSupervisor(agentConfig.GetName(), agentConfig.GetType())
		if err != nil {
			return err
		}
		m.supervisors[agentConfig.GetName()] = sup
	}

	for _, sup := range m.supervisors {
		err := sup.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MetaAgent) getDataDir(agentName string, agentType string) string {
	return filepath.Join(m.config.DataDir, agentType, agentName)
}

func (m *MetaAgent) getLogDir(agentName string, agentType string) string {
	return filepath.Join(m.config.LogDir, agentType, agentName)
}

func (m *MetaAgent) getSupervisor(agentName string, agentType string) (Supervisor, error) {
	dataDir := m.getDataDir(agentName, agentType)
	logDir := m.getLogDir(agentName, agentType)
	switch agentType {
	case "otelcol":
		return otelcol.NewOtelColSupervisor(dataDir, logDir), nil
	default:
		return nil, fmt.Errorf("Unknown agent type %s", agentType)
	}
}
