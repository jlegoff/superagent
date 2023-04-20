package otelcol

type OtelCol struct {
	DataDir string
	LogDir  string
	BinPath string
	Name    string
}

func NewOtelCol(name string) *OtelCol {
	return &OtelCol{Name: name}
}

func NewOtelColSupervisor(dataDir string, logDir string) *OtelCol {
	return &OtelCol{
		DataDir: dataDir,
		LogDir:  logDir,
	}
}

func (otelcol *OtelCol) Start() error {
	return nil
}

func (otelcol *OtelCol) Stop() error {
	return nil
}

func (otelcol *OtelCol) GetType() string {
	return "otelcol"
}

func (otelcol *OtelCol) GetName() string {
	return otelcol.Name
}
