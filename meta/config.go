package meta

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"superagent/otelcol"
	"superagent/supervisor"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/file"
)

type Meta struct {
	ApiKey  string
	DataDir string
	LogDir  string
	Agents  []Agent
}

type Agent interface {
	GetType() string
	GetName() string
	GetSupervisor() supervisor.Supervisor
}

func LoadConfig(path string) (*Meta, error) {
	k := koanf.New("::")
	if err := k.Load(file.Provider(path), Parser()); err != nil {
		return nil, err
	}

	meta := &Meta{}
	if err := k.Unmarshal("", meta); err != nil {
		return nil, fmt.Errorf("cannot parse %v: %w", path, err)
	}

	return meta, nil
}

// implements a koanf parser.
type MetaParser struct{}

func Parser() *MetaParser {
	return &MetaParser{}
}

func (p *MetaParser) Unmarshal(b []byte) (map[string]interface{}, error) {
	var firstPass map[string]interface{}
	if err := yaml.Unmarshal(b, &firstPass); err != nil {
		return nil, err
	}
	secondPass := make(map[string]interface{})
	agents := make([]Agent, 0)
	seenAgents := make(map[string]string)
	dataDir, found := firstPass["dataDir"]
	if !found {
		return nil, fmt.Errorf("No dataDir defined")
	}
	logDir, found := firstPass["logDir"]
	if !found {
		return nil, fmt.Errorf("No logDir defined")
	}
	for k, v := range firstPass {
		switch k {
		case "apiKey", "dataDir", "logDir":
			secondPass[k] = v
		case "agents":
			for _, a := range firstPass[k].([]interface{}) {
				parsedAgent, err := parseAgent(a, dataDir.(string), logDir.(string))
				if err != nil {
					return nil, err
				}
				_, found := seenAgents[parsedAgent.GetName()]
				if found {
					return nil, fmt.Errorf("Agent '%s' defined multiple times", parsedAgent.GetName())
				}
				seenAgents[parsedAgent.GetName()] = ""
				agents = append(agents, parsedAgent)
			}
			secondPass[k] = agents
		default:
			return nil, fmt.Errorf("Unknown parameter '%s'", k)
		}
	}
	return secondPass, nil
}

func parseAgent(in interface{}, baseDataDir string, baseLogDir string) (Agent, error) {
	config := in.(map[string]interface{})
	agentName, found := findString(config, "name")
	if !found {
		return nil, fmt.Errorf("No name defined for agent")
	}

	agentType, found := findString(config, "type")
	if !found {
		return nil, fmt.Errorf("Undefined type for agent '%s'", config["name"].(string))
	}
	dataDir := filepath.Join(baseDataDir, agentType, agentName)
	logDir := filepath.Join(baseLogDir, agentType, agentName)
	switch agentType {
	case "otelcol":
		exec, found := config["executable"]
		if !found {
			return nil, fmt.Errorf("No executable defined")
		}
		return otelcol.NewOtelCol(agentName, dataDir, logDir, exec.(string)), nil
	case "nrdot":
		exec, found := config["executable"]
		if !found {
			return nil, fmt.Errorf("No executable defined")
		}
		return otelcol.NewNrDot(agentName, dataDir, logDir, exec.(string)), nil
	default:
		return nil, fmt.Errorf("Unknown agent type '%s'", agentType)
	}
}

func findString(config map[string]interface{}, param string) (string, bool) {
	value, found := config[param]
	if !found {
		return "", found
	}
	return value.(string), true
}

func (p *MetaParser) Marshal(o map[string]interface{}) ([]byte, error) {
	return yaml.Marshal(o)
}
