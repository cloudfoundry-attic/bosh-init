package config_test

import (
	"errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/cmd/config"
)

var _ = Describe("NewFSConfigFromPath", func() {
	It("expands config path", func() {
		fs := fakesys.NewFakeFileSystem()
		fs.ExpandPathExpanded = "/expanded_config"

		config, err := NewFSConfigFromPath("/config", fs)
		Expect(err).ToNot(HaveOccurred())

		err = config.Save()
		Expect(err).ToNot(HaveOccurred())
		Expect(fs.FileExists("/expanded_config")).To(BeTrue())
	})

	It("returns empty config if file does not exist", func() {
		fs := fakesys.NewFakeFileSystem()

		config, err := NewFSConfigFromPath("/no_config", fs)
		Expect(err).ToNot(HaveOccurred())
		Expect(config.Environment()).To(BeEmpty())
		Expect(config.Environments()).To(BeEmpty())
	})

	It("returns error if expanding path fails", func() {
		fs := fakesys.NewFakeFileSystem()
		fs.ExpandPathErr = errors.New("fake-err")

		_, err := NewFSConfigFromPath("/config", fs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake-err"))
	})

	It("returns error if reading file fails", func() {
		fs := fakesys.NewFakeFileSystem()
		fs.WriteFileString("/config", "")
		fs.ReadFileError = errors.New("fake-err")

		_, err := NewFSConfigFromPath("/config", fs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake-err"))
	})

	It("returns error if config file cannot be unmarshaled", func() {
		fs := fakesys.NewFakeFileSystem()
		fs.WriteFileString("/config", "-")

		_, err := NewFSConfigFromPath("/config", fs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("line 1"))
	})
})

var _ = Describe("FSConfig", func() {
	var (
		fs     *fakesys.FakeFileSystem
		config FSConfig
	)

	readConfig := func() FSConfig {
		config, err := NewFSConfigFromPath("/dir/sub-dir/config", fs)
		Expect(err).ToNot(HaveOccurred())

		return config
	}

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		config = readConfig()
	})

	Describe("Environments", func() {
		It("returns empty list if there are no remembered environments", func() {
			Expect(config.Environments()).To(BeEmpty())
		})

		It("returns list of previously remembered environments", func() {
			updatedConfig := config.SetEnvironment("url1", "", "")
			updatedConfig = updatedConfig.SetEnvironment("url2", "", "")
			updatedConfig = updatedConfig.SetEnvironment("url3", "alias3", "")
			Expect(updatedConfig.Environments()).To(Equal([]Environment{
				Environment{URL: "url1", Alias: ""},
				Environment{URL: "url2", Alias: ""},
				Environment{URL: "url3", Alias: "alias3"},
			}))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Environments()).To(Equal([]Environment{
				Environment{URL: "url1", Alias: ""},
				Environment{URL: "url2", Alias: ""},
				Environment{URL: "url3", Alias: "alias3"},
			}))
		})
	})

	Describe("SetEnvironment/Environment/CACert", func() {
		It("returns empty if file does not exist", func() {
			Expect(config.Environment()).To(Equal(""))
		})

		It("returns saved url", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			Expect(updatedConfig.Environment()).To(Equal("url"))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Environment()).To(Equal("url"))
		})

		It("returns saved url based on the alias, resolving to previously saved url", func() {
			updatedConfig := config.SetEnvironment("url1", "alias1", "")
			updatedConfig = updatedConfig.SetEnvironment("url2", "alias2", "")
			updatedConfig = updatedConfig.SetEnvironment("alias1", "", "")
			Expect(updatedConfig.Environment()).To(Equal("url1"))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Environment()).To(Equal("url1"))
		})

		It("saves empty CA certificate", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			Expect(updatedConfig.Environment()).To(Equal("url"))
			Expect(updatedConfig.CACert("url")).To(Equal(""))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.CACert("url")).To(Equal(""))
		})

		// Valid from pov of FSConfig
		validCACert := "BEGIN\nca-cert"

		It("saves non-empty CA certificate and then unsets it", func() {
			updatedConfig := config.SetEnvironment("url", "", validCACert)
			Expect(updatedConfig.Environment()).To(Equal("url"))
			Expect(updatedConfig.CACert("url")).To(Equal(validCACert))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.CACert("url")).To(Equal(validCACert))

			updatedConfig = reloadedConfig.SetEnvironment("url", "", "")
			Expect(updatedConfig.Environment()).To(Equal("url"))
			Expect(updatedConfig.CACert("url")).To(Equal(""))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.CACert("url")).To(Equal(""))
		})

		It("saves CA cert via file path and does not need file system later", func() {
			fs.WriteFileString("/ca-cert", validCACert)

			updatedConfig := config.SetEnvironment("url", "", "/ca-cert")
			Expect(updatedConfig.CACert("url")).To(Equal(validCACert))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			err = fs.RemoveAll("/ca-cert")
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.CACert("url")).To(Equal(validCACert))
		})

		It("returns CA cert for alias", func() {
			updatedConfig := config.SetEnvironment("url", "alias", validCACert)
			Expect(updatedConfig.Environment()).To(Equal("url"))
			Expect(updatedConfig.CACert("alias")).To(Equal(validCACert))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.CACert("alias")).To(Equal(validCACert))

			updatedConfig = reloadedConfig.SetEnvironment("url", "alias", "")
			Expect(updatedConfig.Environment()).To(Equal("url"))
			Expect(updatedConfig.CACert("alias")).To(Equal(""))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.CACert("alias")).To(Equal(""))
		})

		It("does not update existing config when environment is set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			Expect(updatedConfig.Environment()).To(Equal("url"))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Environment()).To(Equal(""))
		})
	})

	Describe("ResolveEnvironment", func() {
		It("returns url if it's a known url", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			Expect(updatedConfig.ResolveEnvironment("url")).To(Equal("url"))
		})

		It("returns aliased url", func() {
			updatedConfig := config.SetEnvironment("url", "alias", "")
			updatedConfig = updatedConfig.SetEnvironment("url2", "alias2", "")
			Expect(updatedConfig.ResolveEnvironment("alias")).To(Equal("url"))
		})

		It("returns input when it's not an alias or url", func() {
			Expect(config.ResolveEnvironment("unknown")).To(Equal("unknown"))
		})
	})

	Describe("SetCredentials/Credentials/UnsetCredentials", func() {
		It("returns empty if environment is not found", func() {
			Expect(config.Credentials("url")).To(Equal(Creds{}))
		})

		It("returns empty if environment is found but creds are not set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{}))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{}))

			updatedConfig = reloadedConfig.UnsetCredentials("url")
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{}))
		})

		It("returns creds with username/password if environment is found and basic creds are set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			updatedConfig = config.SetCredentials("url", Creds{Username: "user", Password: "pass"})
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{Username: "user", Password: "pass"}))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{Username: "user", Password: "pass"}))

			updatedConfig = reloadedConfig.UnsetCredentials("url")
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{}))
		})

		It("returns creds with token if environment is found and token is set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			updatedConfig = config.SetCredentials("url", Creds{RefreshToken: "token"})
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{RefreshToken: "token"}))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{RefreshToken: "token"}))

			updatedConfig = reloadedConfig.UnsetCredentials("url")
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Credentials("url")).To(Equal(Creds{}))
		})

		It("returns creds for alias if environment is found and token is set", func() {
			updatedConfig := config.SetEnvironment("url", "alias", "")
			updatedConfig = config.SetCredentials("alias", Creds{RefreshToken: "token"})
			Expect(updatedConfig.Credentials("alias")).To(Equal(Creds{RefreshToken: "token"}))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Credentials("alias")).To(Equal(Creds{RefreshToken: "token"}))

			updatedConfig = reloadedConfig.UnsetCredentials("alias")
			Expect(updatedConfig.Credentials("alias")).To(Equal(Creds{}))

			err = updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig = readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Credentials("alias")).To(Equal(Creds{}))
		})

		It("does not update existing config when creds are set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			updatedConfig = config.SetCredentials("url", Creds{Username: "user"})
			Expect(updatedConfig.Credentials("url")).To(Equal(Creds{Username: "user"}))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Credentials("url")).To(Equal(Creds{}))
		})
	})

	Describe("SetDeployment/SkipSslValidation", func() {
		It("returns false if environment is not found", func() {
			Expect(config.SkipSslValidation("url")).To(Equal(false))
		})

		It("returns false if environment is found but skip-ssl-validation is not set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			Expect(updatedConfig.SkipSslValidation("url")).To(Equal(false))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.SkipSslValidation("url")).To(Equal(false))
		})

		It("returns true if environment is found and skip-ssl-validation is set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			updatedConfig = config.SetSkipSslValidation("url", true)
			Expect(updatedConfig.SkipSslValidation("url")).To(Equal(true))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.SkipSslValidation("url")).To(Equal(true))
		})

		It("returns true for alias if environment is found and skip-ssl-validation is set", func() {
			updatedConfig := config.SetEnvironment("url", "alias", "")
			updatedConfig = config.SetSkipSslValidation("alias", true)
			Expect(updatedConfig.SkipSslValidation("alias")).To(Equal(true))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.SkipSslValidation("alias")).To(Equal(true))
		})
	})

	Describe("SetDeployment/Deployment", func() {
		It("returns empty if environment is not found", func() {
			Expect(config.Deployment("url")).To(Equal(""))
		})

		It("returns empty if environment is found but deployment is not set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			Expect(updatedConfig.Deployment("url")).To(Equal(""))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Deployment("url")).To(Equal(""))
		})

		It("returns deployment if environment is found and deployment is set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			updatedConfig = config.SetDeployment("url", "deployment")
			Expect(updatedConfig.Deployment("url")).To(Equal("deployment"))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Deployment("url")).To(Equal("deployment"))
		})

		It("returns deployment for alias if environment is found and deployment is set", func() {
			updatedConfig := config.SetEnvironment("url", "alias", "")
			updatedConfig = config.SetDeployment("alias", "deployment")
			Expect(updatedConfig.Deployment("alias")).To(Equal("deployment"))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			reloadedConfig := readConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedConfig.Deployment("alias")).To(Equal("deployment"))
		})

		It("does not update existing config when deployment is set", func() {
			updatedConfig := config.SetEnvironment("url", "", "")
			updatedConfig = config.SetDeployment("url", "deployment")
			Expect(updatedConfig.Deployment("url")).To(Equal("deployment"))

			err := updatedConfig.Save()
			Expect(err).ToNot(HaveOccurred())

			Expect(config.Deployment("url")).To(Equal(""))
		})
	})

	Describe("Save", func() {
		It("returns error if writing file fails", func() {
			fs.WriteFileError = errors.New("fake-err")

			config := readConfig()
			err := config.Save()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
