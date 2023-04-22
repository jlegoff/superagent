package supervisor

type Supervisor interface {
	Start() error
	Stop() error
	Setup() error
}
