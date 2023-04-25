package opamp

import "github.com/oklog/ulid/v2"

type Agent struct {
	InstanceId ulid.ULID
	Host       Host
	Os         Os
	Service    Service
}

type Host struct {
	Id   string
	Name string
}

type Os struct {
	Version string
	Type    string
}

type Service struct {
	Name    string
	Version string
}

type ConfigFile struct {
	Content     string
	ContentType string
}

type RemoteConfig struct {
	Configs map[string]ConfigFile
	Hash    string
}
