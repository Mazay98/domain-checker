package environment

// Env is a custom type to determine environment we run in.
type Env string

// Available environments we know.
const (
	Local      Env = "local"
	Staging    Env = "staging"
	Production Env = "production"

	ServiceName = "go-domain-checker"
)

// IsLocal returns true if we are running in a local environment.
func (e Env) IsLocal() bool {
	return e == Local
}

// IsProduction returns true if we are running in a production environment.
func (e Env) IsProduction() bool {
	return e == Production || e == Staging
}

// String returns string representation of the Env.
func (e Env) String() string {
	return string(e)
}
