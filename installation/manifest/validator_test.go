package manifest_test

import (
	. "github.com/cloudfoundry/bosh-init/installation/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	birelmanifest "github.com/cloudfoundry/bosh-init/release/manifest"
	birelsetmanifest "github.com/cloudfoundry/bosh-init/release/set/manifest"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
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

		It("validates cloud_provider.mbus is not blank", func() {
			manifest := Manifest{Mbus: ""}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.mbus must be provided"))
		})

		It("validates mbus is a valid URL", func() {
			manifest := Manifest{Mbus: "invalid-url"}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.mbus must be a valid URL"))
			Expect(err.Error()).NotTo(ContainSubstring("cloud_provider.properties.agent.mbus must be a valid URL"))
		})

		It("validates mbus URL and agnet mbus URL are valid URLs (if both specified)", func() {
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
			Expect(err.Error()).To(ContainSubstring("cloud_provider.mbus must be a valid URL"))
			Expect(err.Error()).To(ContainSubstring("cloud_provider.properties.agent.mbus must be a valid URL"))
		})

		It("validates mbus and agent mbus URLs use https protocol (if both specified)", func() {
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

		It("validates mbus and agent mbus URLs use the same port", func() {
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
			Expect(err.Error()).To(ContainSubstring("cloud_provider.properties.agent.mbus and cloud_provider.mbus must have the same ports"))
		})

		It("validates mbus and agent mbus URLs use the same port", func() {
			manifest := Manifest{
				Mbus: "https://user1:pass1@valid-url:3000",
				Properties: biproperty.Map{
					"agent": biproperty.Map{
						"mbus": "https://user2:pass2@valid-url:3000",
					},
				},
			}

			err := validator.Validate(manifest, releaseSetManifest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cloud_provider.properties.agent.mbus and cloud_provider.mbus must have the same password and username"))
		})

	})
})
