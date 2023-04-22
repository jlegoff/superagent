package otelcol

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"path/filepath"
	"superagent/supervisor"
)

type OtelCol struct {
	DataDir string
	LogDir  string
	BinPath string
	Name    string
}

type OtelColSupervisor struct {
	Config    OtelCol
	Commander *Commander
}

func NewOtelCol(name string, dataDir string, logDir string, binPath string) *OtelCol {
	return &OtelCol{Name: name, DataDir: dataDir, LogDir: logDir, BinPath: binPath}
}

func (otelcol *OtelCol) GetType() string {
	return "otelcol"
}

func (otelcol *OtelCol) GetName() string {
	return otelcol.Name
}

func (collector *OtelCol) GetSupervisor() supervisor.Supervisor {
	return &OtelColSupervisor{Config: *collector}
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
