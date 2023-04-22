package otelcol

import (
	"superagent/supervisor"
)

type Nrdot struct {
	DataDir string
	LogDir  string
	BinPath string
	Name    string
}

type NrDotSupervisor struct {
	Config Nrdot
}

func NewNrDot(name string, dataDir string, logDir string, binPath string) *Nrdot {
	return &Nrdot{Name: name, DataDir: dataDir, LogDir: logDir, BinPath: binPath}
}

func (nrdot *Nrdot) GetType() string {
	return "nrdot"
}

func (nrdot *Nrdot) GetName() string {
	return nrdot.Name
}

func (nrdot *Nrdot) GetSupervisor() supervisor.Supervisor {
	return &NrDotSupervisor{Config: *nrdot}
}

func (supervisor *NrDotSupervisor) Start() error {
	return nil
}

func (supervisor *NrDotSupervisor) Stop() error {
	return nil
}

func (sup *NrDotSupervisor) Setup() error {
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
