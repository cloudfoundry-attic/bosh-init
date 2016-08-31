package cmd

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	cmdconf "github.com/cloudfoundry/bosh-init/cmd/config"
	boshui "github.com/cloudfoundry/bosh-init/ui"
)

type BasicLoginStrategy struct {
	sessionFactory func(cmdconf.Config) Session

	opts   BoshOpts
	config cmdconf.Config
	ui     boshui.UI
}

func NewBasicLoginStrategy(
	sessionFactory func(cmdconf.Config) Session,
	opts BoshOpts,
	config cmdconf.Config,
	ui boshui.UI,
) BasicLoginStrategy {
	return BasicLoginStrategy{
		sessionFactory: sessionFactory,
		opts:           opts,
		config:         config,
		ui:             ui,
	}
}

func (s BasicLoginStrategy) Try() error {
	sess := s.sessionFactory(s.config)

	initialCreds := sess.Credentials()

	for {
		authed, err := s.tryOnce(sess.Environment(), initialCreds)
		if err != nil {
			return err
		}

		if authed {
			return nil
		}

		if initialCreds.IsBasicComplete() {
			return bosherr.Error("Invalid credentials")
		}
	}
}

func (s BasicLoginStrategy) tryOnce(environment string, creds cmdconf.Creds) (bool, error) {
	creds, err := s.askForCreds(creds)
	if err != nil {
		return false, err
	}

	updatedConfig := s.config.SetCredentials(environment, creds)
	updatedConfig = updatedConfig.SetSkipSslValidation(environment, s.opts.SkipSslValidationOpt)

	sess := s.sessionFactory(updatedConfig)

	director, err := sess.Director()
	if err != nil {
		return false, err
	}

	authed, err := director.IsAuthenticated()
	if err != nil {
		return false, err
	}

	if !authed {
		s.ui.ErrorLinef("Failed to login to '%s'", environment)
		return false, nil
	}

	err = updatedConfig.Save()
	if err != nil {
		return false, err
	}

	s.ui.PrintLinef("Logged in to '%s'", environment)

	return true, nil
}

func (s BasicLoginStrategy) askForCreds(creds cmdconf.Creds) (cmdconf.Creds, error) {
	if len(creds.Username) == 0 {
		username, err := s.ui.AskForText("Username")
		if err != nil {
			return cmdconf.Creds{}, err
		}

		creds.Username = username
	}

	if len(creds.Password) == 0 {
		password, err := s.ui.AskForPassword("Password")
		if err != nil {
			return cmdconf.Creds{}, err
		}

		creds.Password = password
	}

	return creds, nil
}
