package config

//go:generate counterfeiter . Config

type Config interface {
	Environment() string
	Environments() []Environment
	ResolveEnvironment(urlOrAlias string) string
	SetEnvironment(urlOrAlias, alias, caCert string) Config

	CACert(url string) string
	SkipSslValidation(url string) bool
	SetSkipSslValidation(url string, flag bool) Config

	Credentials(url string) Creds
	SetCredentials(url string, creds Creds) Config
	UnsetCredentials(url string) Config

	Deployment(url string) string
	SetDeployment(url, nameOrPath string) Config

	Save() error
}

type Environment struct {
	URL   string
	Alias string
}
