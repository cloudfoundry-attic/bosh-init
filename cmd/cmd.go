package cmd

import (
	"fmt"

	cmdconf "github.com/cloudfoundry/bosh-init/cmd/config"
	boshdir "github.com/cloudfoundry/bosh-init/director"
	boshtpl "github.com/cloudfoundry/bosh-init/director/template"
	boshrel "github.com/cloudfoundry/bosh-init/release"
	boshreldir "github.com/cloudfoundry/bosh-init/releasedir"
	boshssh "github.com/cloudfoundry/bosh-init/ssh"
	boshui "github.com/cloudfoundry/bosh-init/ui"
	boshuit "github.com/cloudfoundry/bosh-init/ui/task"
)

type Cmd struct {
	BoshOpts *BoshOpts
	Opts     interface{}

	deps BasicDeps
}

func NewCmd(boshOpts *BoshOpts, opts interface{}, deps BasicDeps) Cmd {
	return Cmd{boshOpts, opts, deps}
}

type cmdConveniencePanic struct {
	Err error
}

func (c Cmd) Execute() (cmdErr error) {
	// Catch convenience panics from panicIfErr
	defer func() {
		if r := recover(); r != nil {
			if cp, ok := r.(cmdConveniencePanic); ok {
				cmdErr = cp.Err
			} else {
				panic(r)
			}
		}
	}()

	c.configureUI()
	c.configureFS()

	deps := c.deps

	switch opts := c.Opts.(type) {
	case *CreateEnvOpts:
		envProvider := func(path string, vars boshtpl.Variables) DeploymentPreparer {
			return NewEnvFactory(deps, path, vars).Preparer()
		}

		stage := boshui.NewStage(deps.UI, deps.Time, deps.Logger)
		return NewCreateEnvCmd(deps.UI, envProvider).Run(stage, *opts)

	case *DeleteEnvOpts:
		envProvider := func(path string, vars boshtpl.Variables) DeploymentDeleter {
			return NewEnvFactory(deps, path, vars).Deleter()
		}

		stage := boshui.NewStage(deps.UI, deps.Time, deps.Logger)
		return NewDeleteCmd(deps.UI, envProvider).Run(stage, *opts)

	case *EnvironmentsOpts:
		return NewEnvironmentsCmd(c.config(), deps.UI).Run()

	case *EnvironmentOpts:
		sessionFactory := func(config cmdconf.Config) Session {
			return NewSessionFromOpts(*c.BoshOpts, config, deps.UI, false, false, deps.FS, deps.Logger)
		}

		return NewEnvironmentCmd(sessionFactory, c.config(), deps.UI).Run(*opts)

	case *LogInOpts:
		sessionFactory := func(config cmdconf.Config) Session {
			return NewSessionFromOpts(*c.BoshOpts, config, deps.UI, true, true, deps.FS, deps.Logger)
		}

		config := c.config()
		basicStrategy := NewBasicLoginStrategy(sessionFactory, *c.BoshOpts, config, deps.UI)
		uaaStrategy := NewUAALoginStrategy(sessionFactory, *c.BoshOpts, config, deps.UI, deps.Logger)

		sess := NewSessionFromOpts(*c.BoshOpts, c.config(), deps.UI, true, true, deps.FS, deps.Logger)

		anonDirector, err := sess.AnonymousDirector()
		if err != nil {
			return err
		}

		return NewLogInCmd(basicStrategy, uaaStrategy, anonDirector).Run()

	case *LogOutOpts:
		config := c.config()
		sess := NewSessionFromOpts(*c.BoshOpts, config, deps.UI, true, true, deps.FS, deps.Logger)
		return NewLogOutCmd(sess.Environment(), config, deps.UI).Run()

	case *TaskOpts:
		eventsTaskReporter := boshuit.NewReporter(deps.UI, true)
		plainTaskReporter := boshuit.NewReporter(deps.UI, false)
		return NewTaskCmd(eventsTaskReporter, plainTaskReporter, c.director()).Run(*opts)

	case *TasksOpts:
		return NewTasksCmd(deps.UI, c.director()).Run(*opts)

	case *CancelTaskOpts:
		return NewCancelTaskCmd(c.director()).Run(*opts)

	case *DeploymentOpts:
		sessionFactory := func(config cmdconf.Config) Session {
			return NewSessionFromOpts(*c.BoshOpts, config, deps.UI, true, false, deps.FS, deps.Logger)
		}

		return NewDeploymentCmd(sessionFactory, c.config(), deps.UI).Run(*opts)

	case *DeploymentsOpts:
		return NewDeploymentsCmd(deps.UI, c.director()).Run()

	case *DeleteDeploymentOpts:
		return NewDeleteDeploymentCmd(deps.UI, c.deployment()).Run(*opts)

	case *ReleasesOpts:
		return NewReleasesCmd(deps.UI, c.director()).Run()

	case *UploadReleaseOpts:
		relProv, relDirProv := c.releaseProviders()
		releaseReader := relDirProv.NewReleaseReader(opts.Directory.Path)
		releaseWriter := relProv.NewArchiveWriter()
		releaseDir := relDirProv.NewFSReleaseDir(opts.Directory.Path)

		releaseArchiveFactory := func(path string) boshdir.ReleaseArchive {
			return boshdir.NewFSReleaseArchive(path, deps.FS)
		}

		cmd := NewUploadReleaseCmd(
			releaseReader, releaseWriter, releaseDir, c.director(), releaseArchiveFactory, deps.UI)

		return cmd.Run(*opts)

	case *DeleteReleaseOpts:
		return NewDeleteReleaseCmd(deps.UI, c.director()).Run(*opts)

	case *StemcellsOpts:
		return NewStemcellsCmd(deps.UI, c.director()).Run()

	case *UploadStemcellOpts:
		stemcellArchiveFactory := func(path string) boshdir.StemcellArchive {
			return boshdir.NewFSStemcellArchive(path, deps.FS)
		}

		return NewUploadStemcellCmd(c.director(), stemcellArchiveFactory, deps.UI).Run(*opts)

	case *DeleteStemcellOpts:
		return NewDeleteStemcellCmd(deps.UI, c.director()).Run(*opts)

	case *LocksOpts:
		return NewLocksCmd(deps.UI, c.director()).Run()

	case *ErrandsOpts:
		return NewErrandsCmd(deps.UI, c.deployment()).Run()

	case *RunErrandOpts:
		director, deployment := c.directorAndDeployment()
		downloader := NewUIDownloader(director, deps.SHA1Calc, deps.Time, deps.FS, deps.UI)
		return NewRunErrandCmd(deployment, downloader, deps.UI).Run(*opts)

	case *DisksOpts:
		return NewDisksCmd(deps.UI, c.director()).Run(*opts)

	case *DeleteDiskOpts:
		return NewDeleteDiskCmd(deps.UI, c.director()).Run(*opts)

	case *SnapshotsOpts:
		return NewSnapshotsCmd(deps.UI, c.deployment()).Run(*opts)

	case *TakeSnapshotOpts:
		return NewTakeSnapshotCmd(c.deployment()).Run(*opts)

	case *DeleteSnapshotOpts:
		return NewDeleteSnapshotCmd(deps.UI, c.deployment()).Run(*opts)

	case *DeleteSnapshotsOpts:
		return NewDeleteSnapshotsCmd(deps.UI, c.deployment()).Run()

	case *BuildManifestOpts:
		return NewBuildManifestCmd(deps.UI).Run(*opts)

	case *CloudConfigOpts:
		return NewCloudConfigCmd(deps.UI, c.director()).Run()

	case *UpdateCloudConfigOpts:
		return NewUpdateCloudConfigCmd(deps.UI, c.director()).Run(*opts)

	case *RuntimeConfigOpts:
		return NewRuntimeConfigCmd(deps.UI, c.director()).Run()

	case *UpdateRuntimeConfigOpts:
		director := c.director()
		uploadReleaseCmd := NewUploadReleaseCmd(nil, nil, nil, director, nil, deps.UI)
		return NewUpdateRuntimeConfigCmd(deps.UI, director, uploadReleaseCmd).Run(*opts)

	case *ManifestOpts:
		return NewManifestCmd(deps.UI, c.deployment()).Run()

	case *EventsOpts:
		return NewEventsCmd(deps.UI, c.director()).Run(*opts)

	case *InspectReleaseOpts:
		return NewInspectReleaseCmd(deps.UI, c.director()).Run(*opts)

	case *VMsOpts:
		return NewVMsCmd(deps.UI, c.director()).Run(*opts)

	case *InstancesOpts:
		return NewInstancesCmd(deps.UI, c.deployment()).Run(*opts)

	case *VMResurrectionOpts:
		return NewVMResurrectionCmd(c.director()).Run(*opts)

	case *DeployOpts:
		director, deployment := c.directorAndDeployment()
		uploadReleaseCmd := NewUploadReleaseCmd(nil, nil, nil, director, nil, deps.UI)
		return NewDeployCmd(deps.UI, deployment, uploadReleaseCmd).Run(*opts)

	case *StartOpts:
		return NewStartCmd(deps.UI, c.deployment()).Run(*opts)

	case *StopOpts:
		return NewStopCmd(deps.UI, c.deployment()).Run(*opts)

	case *RestartOpts:
		return NewRestartCmd(deps.UI, c.deployment()).Run(*opts)

	case *RecreateOpts:
		return NewRecreateCmd(deps.UI, c.deployment()).Run(*opts)

	case *CloudCheckOpts:
		return NewCloudCheckCmd(c.deployment(), deps.UI).Run(*opts)

	case *CleanUpOpts:
		return NewCleanUpCmd(deps.UI, c.director()).Run(*opts)

	case *LogsOpts:
		director, deployment := c.directorAndDeployment()
		downloader := NewUIDownloader(director, deps.SHA1Calc, deps.Time, deps.FS, deps.UI)
		sshProvider := boshssh.NewProvider(deps.CmdRunner, deps.FS, deps.UI, deps.Logger)
		nonIntSSHRunner := sshProvider.NewSSHRunner(false)
		return NewLogsCmd(deployment, downloader, deps.UUIDGen, nonIntSSHRunner).Run(*opts)

	case *SSHOpts:
		sshProvider := boshssh.NewProvider(deps.CmdRunner, deps.FS, deps.UI, deps.Logger)
		intSSHRunner := sshProvider.NewSSHRunner(true)
		nonIntSSHRunner := sshProvider.NewSSHRunner(false)
		resultsSSHRunner := sshProvider.NewResultsSSHRunner(false)
		return NewSSHCmd(c.deployment(), deps.UUIDGen, intSSHRunner, nonIntSSHRunner, resultsSSHRunner, deps.UI).Run(*opts)

	case *SCPOpts:
		sshProvider := boshssh.NewProvider(deps.CmdRunner, deps.FS, deps.UI, deps.Logger)
		scpRunner := sshProvider.NewSCPRunner()
		return NewSCPCmd(c.deployment(), deps.UUIDGen, scpRunner, deps.UI).Run(*opts)

	case *ExportReleaseOpts:
		director, deployment := c.directorAndDeployment()
		downloader := NewUIDownloader(director, deps.SHA1Calc, deps.Time, deps.FS, deps.UI)
		return NewExportReleaseCmd(deployment, downloader).Run(*opts)

	case *InitReleaseOpts:
		return NewInitReleaseCmd(c.releaseDir(opts.Directory)).Run(*opts)

	case *ResetReleaseOpts:
		return NewResetReleaseCmd(c.releaseDir(opts.Directory)).Run(*opts)

	case *GenerateJobOpts:
		return NewGenerateJobCmd(c.releaseDir(opts.Directory)).Run(*opts)

	case *GeneratePackageOpts:
		return NewGeneratePackageCmd(c.releaseDir(opts.Directory)).Run(*opts)

	case *FinalizeReleaseOpts:
		_, relDirProv := c.releaseProviders()
		releaseReader := relDirProv.NewReleaseReader(opts.Directory.Path)
		releaseDir := relDirProv.NewFSReleaseDir(opts.Directory.Path)
		return NewFinalizeReleaseCmd(releaseReader, releaseDir, deps.UI).Run(*opts)

	case *CreateReleaseOpts:
		_, relDirProv := c.releaseProviders()
		releaseReader := relDirProv.NewReleaseReader(opts.Directory.Path)
		releaseDir := relDirProv.NewFSReleaseDir(opts.Directory.Path)
		return NewCreateReleaseCmd(releaseReader, releaseDir, deps.UI).Run(*opts)

	case *BlobsOpts:
		return NewBlobsCmd(c.blobsDir(opts.Directory), deps.UI).Run()

	case *AddBlobOpts:
		return NewAddBlobCmd(c.blobsDir(opts.Directory), deps.FS, deps.UI).Run(*opts)

	case *RemoveBlobOpts:
		return NewRemoveBlobCmd(c.blobsDir(opts.Directory), deps.UI).Run(*opts)

	case *UploadBlobsOpts:
		return NewUploadBlobsCmd(c.blobsDir(opts.Directory)).Run()

	case *SyncBlobsOpts:
		return NewSyncBlobsCmd(c.blobsDir(opts.Directory)).Run()

	default:
		return fmt.Errorf("Unhandled command: %#v", c.Opts)
	}
}

func (c Cmd) configureUI() {
	c.deps.UI.EnableTTY(c.BoshOpts.TTYOpt)

	c.deps.UI.EnableColor()

	if c.BoshOpts.JSONOpt {
		c.deps.UI.EnableJSON()
	}

	if c.BoshOpts.NonInteractiveOpt {
		c.deps.UI.EnableNonInteractive()
	}
}

func (c Cmd) configureFS() {
	tmpDirPath, err := c.deps.FS.ExpandPath("~/.bosh/tmp")
	c.panicIfErr(err)

	err = c.deps.FS.ChangeTempRoot(tmpDirPath)
	c.panicIfErr(err)
}

func (c Cmd) config() cmdconf.Config {
	config, err := cmdconf.NewFSConfigFromPath(c.BoshOpts.ConfigPathOpt, c.deps.FS)
	c.panicIfErr(err)

	return config
}

func (c Cmd) session() Session {
	return NewSessionFromOpts(*c.BoshOpts, c.config(), c.deps.UI, true, true, c.deps.FS, c.deps.Logger)
}

func (c Cmd) director() boshdir.Director {
	director, err := c.session().Director()
	c.panicIfErr(err)

	return director
}

func (c Cmd) deployment() boshdir.Deployment {
	deployment, err := c.session().Deployment()
	c.panicIfErr(err)

	return deployment
}

func (c Cmd) directorAndDeployment() (boshdir.Director, boshdir.Deployment) {
	sess := c.session()

	director, err := sess.Director()
	c.panicIfErr(err)

	deployment, err := sess.Deployment()
	c.panicIfErr(err)

	return director, deployment
}

func (c Cmd) releaseProviders() (boshrel.Provider, boshreldir.Provider) {
	indexReporter := boshui.NewIndexReporter(c.deps.UI)
	blobsReporter := boshui.NewBlobsReporter(c.deps.UI)
	releaseIndexReporter := boshui.NewReleaseIndexReporter(c.deps.UI)

	releaseProvider := boshrel.NewProvider(
		c.deps.CmdRunner, c.deps.Compressor, c.deps.SHA1Calc, c.deps.FS, c.deps.Logger)

	releaseDirProvider := boshreldir.NewProvider(
		indexReporter, releaseIndexReporter, blobsReporter, releaseProvider,
		c.deps.SHA1Calc, c.deps.CmdRunner, c.deps.UUIDGen, c.deps.FS, c.deps.Logger)

	return releaseProvider, releaseDirProvider
}

func (c Cmd) blobsDir(dir DirOrCWDArg) boshreldir.BlobsDir {
	_, relDirProv := c.releaseProviders()
	return relDirProv.NewFSBlobsDir(dir.Path)
}

func (c Cmd) releaseDir(dir DirOrCWDArg) boshreldir.ReleaseDir {
	_, relDirProv := c.releaseProviders()
	return relDirProv.NewFSReleaseDir(dir.Path)
}

func (c Cmd) panicIfErr(err error) {
	if err != nil {
		panic(cmdConveniencePanic{err})
	}
}
