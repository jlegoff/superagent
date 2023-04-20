package meta

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMetaConfig(t *testing.T) {
	configPath := "testdata/meta_config.yaml"
	meta, err := LoadConfig(configPath)
	assert.Nil(t, err)

	assert.Equal(t, meta.ApiKey, "key")
	assert.Equal(t, meta.DataDir, "/etc/newrelic/meta")
	assert.Equal(t, meta.LogDir, "/var/log/newrelic/meta")
	assert.Equal(t, len(meta.Agents), 2)

	nrdot := meta.Agents[0]
	assert.Equal(t, nrdot.GetType(), "nrdot")
	assert.Equal(t, nrdot.GetName(), "nrdot-name")

	otelcol := meta.Agents[1]
	assert.Equal(t, otelcol.GetType(), "otelcol")
	assert.Equal(t, otelcol.GetName(), "otelcol-name")
}

func TestWrongAgentType(t *testing.T) {
	configPath := "testdata/meta_config_wrong_agent_type.yaml"
	_, err := LoadConfig(configPath)
	assert.EqualErrorf(t, err, "Unknown agent type 'unknown-type'", "Wrong error message")
}

func TestNoAgentType(t *testing.T) {
	configPath := "testdata/meta_config_no_agent_type.yaml"
	_, err := LoadConfig(configPath)
	assert.EqualErrorf(t, err, "Undefined type for agent 'unknown-name'", "Wrong error message")
}

func TestRepeatedAgent(t *testing.T) {
	configPath := "testdata/meta_config_repeated_agent.yaml"
	_, err := LoadConfig(configPath)
	assert.EqualErrorf(t, err, "Agent 'nrdot-name' defined multiple times", "Wrong error message")
}

func TestUnknownParameter(t *testing.T) {
	configPath := "testdata/meta_config_unknown_param.yaml"
	_, err := LoadConfig(configPath)
	assert.EqualErrorf(t, err, "Unknown parameter 'unknownParam'", "Wrong error message")
}
