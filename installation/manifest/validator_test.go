package manifest_test

import (
	. "github.com/cloudfoundry/bosh-init/installation/manifest"
	. "github.com/cloudfoundry/bosh-init/internal/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/bosh-init/internal/github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-init/internal/github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-init/internal/github.com/cloudfoundry/bosh-utils/property"
	birelmanifest "github.com/cloudfoundry/bosh-init/release/manifest"
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
)

var _ = Describe("Validator", func() {
	var (
		logger             boshlog.Logger
		releaseSetManifest birelsetmanifest.Manifest
		validator          Validator

		releases      []birelmanifest.ReleaseRef
		validManifest Manifest
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)

		releases = []birelmanifest.ReleaseRef{
			{Name: "provided-valid-release-name"},
		}

		validManifest = Manifest{
			Name: "fake-installation-name",
			Mbus: "https://user:pass@ip-address:4222",
			Template: ReleaseJobRef{
				Name:    "cpi",
				Release: "provided-valid-release-name",
			},
			Properties: biproperty.Map{
				"agent": biproperty.Map{
					"mbus": "https://user:pass@0.0.0.0:4222",
				},
				"fake-prop-key": "fake-prop-value",
				"fake-prop-map-key": biproperty.Map{
					"fake-prop-key": "fake-prop-value",
				},
			},
		}

		releaseSetManifest = birelsetmanifest.Manifest{
			Releases: releases,
		}

		validator = NewValidator(logger)
	})

	Describe("Validate", func() {
		It("does not error if deployment is valid", func() {
			manifest := validManifest

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).ToNot(HaveOccurred())
		})

		It("validates template must be fully specified", func() {
			manifest := Manifest{}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.name must be provided"))
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.release must be provided"))
		})

		It("validates template.name is not blank", func() {
			manifest := Manifest{
				Template: ReleaseJobRef{
					Name: " ",
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.name must be provided"))
		})

		It("validates template.release is not blank", func() {
			manifest := Manifest{
				Template: ReleaseJobRef{
					Release: " ",
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.release must be provided"))
		})

		It("validates the release is available", func() {
			manifest := Manifest{
				Template: ReleaseJobRef{
					Release: "not-provided-valid-release-name",
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.template.release 'not-provided-valid-release-name' must refer to a release in releases"))
		})

		It("validates mbus is not blank", func() {
			manifest := Manifest{Mbus: ""}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.mbus must be provided"))
		})

		It("validates agent properties are not specified", func() {
			manifest := Manifest{
				Properties: biproperty.Map{},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.properties.agent must be specified"))
		})

		It("validates agent mbus property is not empty", func() {
			manifest := Manifest{
				Mbus:       "some-url",
				Properties: biproperty.Map{"agent": biproperty.Map{}},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.properties.agent.mbus must be specified"))
		})

		It("validates mbus and agnet mbus are valid URLs", func() {
			manifest := Manifest{
				Mbus: "invalid-url",
				Properties: biproperty.Map{
					"agent": biproperty.Map{
						"mbus": "invalid-url",
					},
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.mbus should be a valid URL"))
			Expect(err.Error()).To(ContainSubstring("cloud_provider.properties.agent.mbus should be a valid URL"))
		})

		It("validates mbus and agent mbus URLs use https protocol", func() {
			manifest := Manifest{
				Mbus: "http://valid-url",
				Properties: biproperty.Map{
					"agent": biproperty.Map{
						"mbus": "http://valid-url",
					},
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.mbus must use https protocol"))
			Expect(err.Error()).To(ContainSubstring("cloud_provider.properties.agent.mbus must use https protocol"))
		})

		It("validates mbus and agent mbus URLs use https protocol", func() {
			manifest := Manifest{
				Mbus: "https://valid-url:3000",
				Properties: biproperty.Map{
					"agent": biproperty.Map{
						"mbus": "https://valid-url:3001",
					},
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.properties.agent.mbus and cloud_provider.mbus should have the same ports"))
		})

	})
})
