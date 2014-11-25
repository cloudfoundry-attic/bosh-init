package cmd

import (
	"errors"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshsys "github.com/cloudfoundry/bosh-agent/system"

	bmcloud "github.com/cloudfoundry/bosh-micro-cli/cloud"
	bmconfig "github.com/cloudfoundry/bosh-micro-cli/config"
	bmcpi "github.com/cloudfoundry/bosh-micro-cli/cpi"
	bmdeployer "github.com/cloudfoundry/bosh-micro-cli/deployer"
	bmhttpclient "github.com/cloudfoundry/bosh-micro-cli/deployer/httpclient"
	bmstemcell "github.com/cloudfoundry/bosh-micro-cli/deployer/stemcell"
	bmdepl "github.com/cloudfoundry/bosh-micro-cli/deployment"
	bmdeplval "github.com/cloudfoundry/bosh-micro-cli/deployment/validator"
	bmeventlog "github.com/cloudfoundry/bosh-micro-cli/eventlogger"
	bmui "github.com/cloudfoundry/bosh-micro-cli/ui"
)

type deployCmd struct {
	ui                      bmui.UI
	userConfig              bmconfig.UserConfig
	fs                      boshsys.FileSystem
	cpiManifestParser       bmdepl.ManifestParser
	boshManifestParser      bmdepl.ManifestParser
	boshDeploymentValidator bmdeplval.DeploymentValidator
	cpiInstaller            bmcpi.Installer
	stemcellExtractor       bmstemcell.Extractor
	deploymentRecord        bmdeployer.DeploymentRecord
	deployer                bmdeployer.Deployer
	deploymentUUID          string
	eventLogger             bmeventlog.EventLogger
	logger                  boshlog.Logger
	logTag                  string
}

func NewDeployCmd(
	ui bmui.UI,
	userConfig bmconfig.UserConfig,
	fs boshsys.FileSystem,
	cpiManifestParser bmdepl.ManifestParser,
	boshManifestParser bmdepl.ManifestParser,
	boshDeploymentValidator bmdeplval.DeploymentValidator,
	cpiInstaller bmcpi.Installer,
	stemcellExtractor bmstemcell.Extractor,
	deploymentRecord bmdeployer.DeploymentRecord,
	deployer bmdeployer.Deployer,
	deploymentUUID string,
	eventLogger bmeventlog.EventLogger,
	logger boshlog.Logger,
) *deployCmd {
	return &deployCmd{
		ui:                      ui,
		userConfig:              userConfig,
		fs:                      fs,
		cpiManifestParser:       cpiManifestParser,
		boshManifestParser:      boshManifestParser,
		boshDeploymentValidator: boshDeploymentValidator,
		cpiInstaller:            cpiInstaller,
		stemcellExtractor:       stemcellExtractor,
		deploymentRecord:        deploymentRecord,
		deployer:                deployer,
		deploymentUUID:          deploymentUUID,
		eventLogger:             eventLogger,
		logger:                  logger,
		logTag:                  "deployCmd",
	}
}

func (c *deployCmd) Name() string {
	return "deploy"
}

func (c *deployCmd) Run(args []string) error {
	cpiEndpoint, stemcellTarballPath, err := c.parseCmdInputs(args)
	if err != nil {
		return err
	}

	boshDeployment, cpiDeployment, extractedStemcell, err := c.validateInputFiles(stemcellTarballPath)
	if err != nil {
		return err
	}
	defer extractedStemcell.Delete()

	httpClient := bmhttpclient.NewHTTPClient(c.logger)
	cpiCmdRunner := bmcloud.NewHTTPCmdRunner(c.deploymentUUID, cpiEndpoint, httpClient, c.logger)
	cloud := bmcloud.NewCloud(cpiCmdRunner, c.deploymentUUID, c.logger)

	err = c.deployer.Deploy(
		cloud,
		boshDeployment,
		extractedStemcell,
		cpiDeployment.Registry,
		cpiDeployment.SSHTunnel,
		cpiDeployment.Mbus,
	)
	if err != nil {
		return bosherr.WrapError(err, "Deploying Microbosh")
	}

	return nil
}

type Deployment struct{}

func (c *deployCmd) validateInputFiles(stemcellTarballPath string) (
	boshDeployment bmdepl.Deployment,
	cpiDeployment bmdepl.Deployment,
	extractedStemcell bmstemcell.ExtractedStemcell,
	err error,
) {
	validationStage := c.eventLogger.NewStage("validating")
	validationStage.Start()

	manifestValidationStep := validationStage.NewStep("Validating deployment manifest")
	manifestValidationStep.Start()

	if c.userConfig.DeploymentFile == "" {
		err = bosherr.New("No deployment set")
		manifestValidationStep.Fail(err.Error())
		return boshDeployment, cpiDeployment, nil, err
	}

	deploymentFilePath := c.userConfig.DeploymentFile

	c.logger.Info(c.logTag, "Checking for deployment `%s'", deploymentFilePath)
	if !c.fs.FileExists(deploymentFilePath) {
		err = bosherr.New("Verifying that the deployment `%s' exists", deploymentFilePath)
		manifestValidationStep.Fail(err.Error())
		return boshDeployment, cpiDeployment, nil, err
	}

	cpiDeployment, err = c.cpiManifestParser.Parse(deploymentFilePath)
	if err != nil {
		err = bosherr.WrapError(err, "Parsing CPI deployment manifest `%s'", deploymentFilePath)
		manifestValidationStep.Fail(err.Error())
		return boshDeployment, cpiDeployment, nil, err
	}

	boshDeployment, err = c.boshManifestParser.Parse(deploymentFilePath)
	if err != nil {
		err = bosherr.WrapError(err, "Parsing deployment manifest `%s'", deploymentFilePath)
		manifestValidationStep.Fail(err.Error())
		return boshDeployment, cpiDeployment, nil, err
	}

	err = c.boshDeploymentValidator.Validate(boshDeployment)
	if err != nil {
		err = bosherr.WrapError(err, "Validating deployment manifest")
		manifestValidationStep.Fail(err.Error())
		return boshDeployment, cpiDeployment, nil, err
	}

	manifestValidationStep.Finish()

	stemcellValidationStep := validationStage.NewStep("Validating stemcell")
	stemcellValidationStep.Start()

	if !c.fs.FileExists(stemcellTarballPath) {
		err = bosherr.New("Verifying that the stemcell `%s' exists", stemcellTarballPath)
		stemcellValidationStep.Fail(err.Error())
		return boshDeployment, cpiDeployment, nil, err
	}

	extractedStemcell, err = c.stemcellExtractor.Extract(stemcellTarballPath)
	if err != nil {
		err = bosherr.WrapError(err, "Extracting stemcell from `%s'", stemcellTarballPath)
		stemcellValidationStep.Fail(err.Error())
		return boshDeployment, cpiDeployment, nil, err
	}

	stemcellValidationStep.Finish()

	validationStage.Finish()

	return boshDeployment, cpiDeployment, extractedStemcell, nil
}

func (c *deployCmd) parseCmdInputs(args []string) (string, string, error) {
	if len(args) != 2 {
		c.ui.Error("Invalid usage - deploy command requires exactly 2 arguments")
		c.ui.Sayln("Expected usage: bosh-micro deploy <cpi-endpoint> <stemcell-tarball>")
		c.logger.Error(c.logTag, "Invalid arguments: ")
		return "", "", errors.New("Invalid usage - deploy command requires exactly 2 arguments")
	}
	return args[0], args[1], nil
}
