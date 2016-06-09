package manifest

import (
	"net"
	"net/url"
	"strings"

	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
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

	mbusUrl, mbusUrlIsValid, err := v.extractMbusUrl(manifest)
	if err != nil {
		errs = append(errs, err)
	}

	agentMbusUrl, agentMbusUrlFound, err := v.extractAgentMbusUrl(manifest)
	if err != nil {
		errs = append(errs, err)
	}

	if mbusUrlIsValid && agentMbusUrlFound {
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

func (v *validator) extractMbusUrl(manifest Manifest) (url.URL, bool, error) {
	mbusUrlString := manifest.Mbus

	if v.isBlank(mbusUrlString) {
		return url.URL{}, false, bosherr.Errorf("cloud_provider.mbus must be provided")
	}

	mbusUrl, err := url.ParseRequestURI(mbusUrlString)
	if err != nil {
		return url.URL{}, false, bosherr.Error("cloud_provider.mbus must be a valid URL")
	}

	return *mbusUrl, true, nil
}

func (v *validator) extractAgentMbusUrl(manifest Manifest) (url.URL, bool, error) {
	agentProperties, found := manifest.Properties["agent"]
	if !found {
		return url.URL{}, false, nil
	}

	agentPropertiesMap, ok := agentProperties.(biproperty.Map)
	if !ok {
		return url.URL{}, false, nil
	}

	agentMbusUrlProperty, found := agentPropertiesMap["mbus"]
	if !found {
		return url.URL{}, false, nil
	}

	agentMbusUrlString, ok := agentMbusUrlProperty.(string)
	if !ok {
		return url.URL{}, false, bosherr.Error("cloud_provider.properties.agent.mbus must be a string")
	}

	agentMbusUrl, err := url.ParseRequestURI(agentMbusUrlString)
	if err != nil {
		return url.URL{}, false, bosherr.Error("cloud_provider.properties.agent.mbus must be a valid URL")
	}

	return *agentMbusUrl, true, nil
}

func (v *validator) validateMbusUrls(mbusUrl url.URL, agentMbusUrl url.URL) error {
	errs := []error{}

	if mbusUrl.Scheme != "https" {
		errs = append(errs, bosherr.Error("cloud_provider.mbus must use https protocol"))
	}

	if agentMbusUrl.Scheme != "https" {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus must use https protocol"))
	}

	if (mbusUrl.User != nil) && (agentMbusUrl.User != nil) {
		mbusCredsAreEqual := (agentMbusUrl.User.String() == mbusUrl.User.String())
		if !mbusCredsAreEqual {
			errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus and cloud_provider.mbus must have the same password and username"))
		}
	}

	_, mbusPort, _ := net.SplitHostPort(mbusUrl.Host)
	_, agentMbusPort, _ := net.SplitHostPort(agentMbusUrl.Host)
	if mbusPort != agentMbusPort {
		errs = append(errs, bosherr.Error("cloud_provider.properties.agent.mbus and cloud_provider.mbus must have the same ports"))
	}

	if len(errs) > 0 {
		return bosherr.NewMultiError(errs...)
	}

	return nil

}
