package manifest

import (
	"net"
	"net/url"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-init/internal/github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-init/internal/github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-init/internal/github.com/cloudfoundry/bosh-utils/property"
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
)

type Validator interface {
	Validate(Manifest, birelsetmanifest.Manifest) error
}

type validator struct {
	logger boshlog.Logger
}

func NewValidator(logger boshlog.Logger) Validator {
	return &validator{
		logger: logger,
	}
}

func (v *validator) Validate(manifest Manifest, releaseSetManifest birelsetmanifest.Manifest) error {
	errs := []error{}

	cpiJobName := manifest.Template.Name
	if v.isBlank(cpiJobName) {
		errs = append(errs, bosherr.Error("cloud_provider.template.name must be provided"))
	}

	cpiReleaseName := manifest.Template.Release
	if v.isBlank(cpiReleaseName) {
		errs = append(errs, bosherr.Error("cloud_provider.template.release must be provided"))
	}

	_, found := releaseSetManifest.FindByName(cpiReleaseName)
	if !found {
		errs = append(errs, bosherr.Errorf("cloud_provider.template.release '%s' must refer to a release in releases", cpiReleaseName))
	}

	err := v.validateMbusConfig(manifest)
	if err != nil {
		errs = append(errs, bosherr.WrapError(err, "Validating mbus cloud provider properties"))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	return nil
}

func (v *validator) isBlank(str string) bool {
	return str == "" || strings.TrimSpace(str) == ""
}

func (v *validator) validateMbusConfig(manifest Manifest) error {
	errs := []error{}

	mbusURLString := manifest.Mbus
	if v.isBlank(mbusURLString) {
		errs = append(errs, bosherr.Error("cloud_provider.mbus must be provided"))
	}

	agentProperties, found := manifest.Properties["agent"]
	if !found {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent must be specified"))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	agentPropertiesMap, ok := agentProperties.(biproperty.Map)
	if !ok {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent must be a hash"))
		return bosherr.NewMultiError(errs...)
	}

	agentMbusURLProperty, found := agentPropertiesMap["mbus"]
	if !found {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus must be specified"))
		return bosherr.NewMultiError(errs...)
	}

	agentMbusURLString, ok := agentMbusURLProperty.(string)
	if !ok {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus should be string"))
		return bosherr.NewMultiError(errs...)
	}

	mbusURL, err := url.ParseRequestURI(mbusURLString)
	if err != nil {
		errs = append(errs, bosherr.Error("cloud_provider.mbus should be a valid URL"))
	}

	agentMbusURL, err := url.ParseRequestURI(agentMbusURLString)
	if err != nil {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus should be a valid URL"))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	if mbusURL.Scheme != "https" {
		errs = append(errs, bosherr.Error("cloud_provider.mbus must use https protocol"))
	}

	if agentMbusURL.Scheme != "https" {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus must use https protocol"))
	}

	if (mbusURL.User != nil) && (agentMbusURL.User != nil) {
		mbusCredsAreEqual := (agentMbusURL.User.String() == mbusURL.User.String())
		if !mbusCredsAreEqual {
			errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus and cloud_provider.mbus should have the same password and username"))
		}
	}

	_, mbusPort, _ := net.SplitHostPort(mbusURL.Host)
	_, agentMbusPort, _ := net.SplitHostPort(agentMbusURL.Host)
	if mbusPort != agentMbusPort {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus and cloud_provider.mbus should have the same ports"))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	return nil

}
