package otelcol

type Nrdot struct {
	DataDir string
	LogDir  string
	BinPath string
	Name    string
}

func NewNrDot(name string) *Nrdot {
	return &Nrdot{Name: name}
}

func (nrdot *Nrdot) GetType() string {
	return "nrdot"
}

func (nrdot *Nrdot) GetName() string {
	return nrdot.Name
}
