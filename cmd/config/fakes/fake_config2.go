package fakes

import (
	"github.com/cloudfoundry/bosh-init/cmd/config"
)

type FakeConfig2 struct {
	Existing ConfigContents

	Saved   *ConfigContents
	SaveErr error
}

type ConfigContents struct {
	EnvironmentURL    string
	EnvironmentAlias  string
	EnvironmentCACert string

	Called bool
}

func (f *FakeConfig2) Environment() string {
	return f.Existing.EnvironmentURL
}

func (f *FakeConfig2) Environments() []config.Environment {
	panic("Not implemented")
}

func (f *FakeConfig2) ResolveEnvironment(environmentOrName string) string {
	return ""
}

func (f *FakeConfig2) SetEnvironment(environment, alias, caCert string) config.Config {
	f.Saved = &ConfigContents{}

	return &FakeConfig2{
		Existing: ConfigContents{
			EnvironmentURL:    environment,
			EnvironmentAlias:  alias,
			EnvironmentCACert: caCert,
		},

		Saved:   f.Saved,
		SaveErr: f.SaveErr,
	}
}

func (f *FakeConfig2) CACert(environment string) string {
	return f.Existing.EnvironmentCACert
}

func (f *FakeConfig2) SkipSslValidation(environment string) bool {
	panic("Not implemented")
}

func (f *FakeConfig2) SetSkipSslValidation(environment string, flag bool) config.Config {
	panic("Not implemented")
}

func (f *FakeConfig2) Credentials(environment string) config.Creds {
	panic("Not implemented")
}

func (f *FakeConfig2) SetCredentials(environment string, creds config.Creds) config.Config {
	panic("Not implemented")
}

func (f *FakeConfig2) UnsetCredentials(environment string) config.Config {
	panic("Not implemented")
}

func (f *FakeConfig2) Deployment(environment string) string {
	panic("Not implemented")
}

func (f *FakeConfig2) SetDeployment(environment string, nameOrPath string) config.Config {
	panic("Not implemented")
}

func (f *FakeConfig2) Save() error {
	f.Saved.EnvironmentURL = f.Existing.EnvironmentURL
	f.Saved.EnvironmentAlias = f.Existing.EnvironmentAlias
	f.Saved.EnvironmentCACert = f.Existing.EnvironmentCACert
	f.Saved.Called = true
	return f.SaveErr
}
