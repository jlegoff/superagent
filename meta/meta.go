package meta

import (
	"superagent/supervisor"
)

type MetaAgent struct {
	config      Meta
	supervisors map[string]supervisor.Supervisor
}

func NewMetaAgent(configPath string) (*MetaAgent, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return &MetaAgent{config: *config}, nil
}

func (m *MetaAgent) Start() error {
	m.supervisors = make(map[string]supervisor.Supervisor)
	for _, agentConfig := range m.config.Agents {
		sup := agentConfig.GetSupervisor()
		err := sup.Setup()
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

func (m *MetaAgent) Stop() error {
	for _, sup := range m.supervisors {
		err := sup.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}
