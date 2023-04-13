package meta

import (
	"fmt"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

type Meta struct {
	ApiKey string
	Agents []Agent
}

type Agent struct {
	Type string
	Name string
}

func LoadConfig(path string) (*Meta, error) {
	k := koanf.New("::")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return nil, err
	}

	meta := &Meta{}
	if err := k.Unmarshal("", meta); err != nil {
		return nil, fmt.Errorf("cannot parse %v: %w", path, err)
	}

	return meta, nil
}
