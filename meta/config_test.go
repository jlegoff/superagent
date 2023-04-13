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
	assert.Equal(t, len(meta.Agents), 2)

	nrdot := meta.Agents[0]
	assert.Equal(t, nrdot.Type, "nrdot")
	assert.Equal(t, nrdot.Name, "nrdot-name")

	otelcol := meta.Agents[1]
	assert.Equal(t, otelcol.Type, "otelcol")
	assert.Equal(t, otelcol.Name, "otelcol-name")
}
