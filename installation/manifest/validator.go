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

	agentMbusUrl, agentMbusFound := v.extractAgentMbusProperty(manifest)
	mbusUrl, mbusFound := v.extractMbusUrl(manifest)

	if mbusFound && agentMbusFound {
		err := v.validateMbusUrls(mbusUrl, agentMbusUrl)
		if err != nil {
			errs = append(errs, bosherr.WrapError(err, "Validating mbus cloud provider properties"))
		}
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	return nil
}

func (v *validator) isBlank(str string) bool {
	return str == "" || strings.TrimSpace(str) == ""
}

func (v *validator) extractMbusUrl(manifest Manifest) (string, bool) {
	mbusUrl := manifest.Mbus

	if v.isBlank(mbusUrl) {
		return "", false
	}

	return mbusUrl, true
}

func (v *validator) extractAgentMbusProperty(manifest Manifest) (string, bool) {
	agentProperties, found := manifest.Properties["agent"]
	if !found {
		return "", false
	}

	agentPropertiesMap, ok := agentProperties.(biproperty.Map)
	if !ok {
		return "", false
	}

	agentMbusUrlProperty, found := agentPropertiesMap["mbus"]
	if !found {
		return "", false
	}

	agentMbusUrl, ok := agentMbusUrlProperty.(string)
	if !ok {
		return "", false
	}

	return agentMbusUrl, true
}

func (v *validator) validateMbusUrls(mbusUrlString string, agentMbusUrlString string) error {
	errs := []error{}

	mbusUrl, err := url.ParseRequestURI(mbusUrlString)
	if err != nil {
		errs = append(errs, bosherr.Error("cloud_provider.mbus should be a valid URL"))
	}

	agentMbusUrl, err := url.ParseRequestURI(agentMbusUrlString)
	if err != nil {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus should be a valid URL"))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	if mbusUrl.Scheme != "https" {
		errs = append(errs, bosherr.Error("cloud_provider.mbus must use https protocol"))
	}

	if agentMbusUrl.Scheme != "https" {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus must use https protocol"))
	}

	if (mbusUrl.User != nil) && (agentMbusUrl.User != nil) {
		mbusCredsAreEqual := (agentMbusUrl.User.String() == mbusUrl.User.String())
		if !mbusCredsAreEqual {
			errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus and cloud_provider.mbus should have the same password and username"))
		}
	}

	_, mbusPort, _ := net.SplitHostPort(mbusUrl.Host)
	_, agentMbusPort, _ := net.SplitHostPort(agentMbusUrl.Host)
	if mbusPort != agentMbusPort {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus and cloud_provider.mbus should have the same ports"))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	return nil

}
