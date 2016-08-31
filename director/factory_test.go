package director_test

import (
	"crypto/tls"
	"net/http"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/cloudfoundry/bosh-init/director"
)

var _ = Describe("Factory", func() {
	Describe("New", func() {
		It("returns error if config is invalid", func() {
			_, err := NewFactory(boshlog.NewLogger(boshlog.LevelNone)).New(Config{}, nil, nil)
			Expect(err).To(HaveOccurred())
		})

		Context("with valid TLS server", func() {
			var (
				server *ghttp.Server
			)

			BeforeEach(func() {
				server = ghttp.NewUnstartedServer()

				cert, err := tls.X509KeyPair(validCert, validKey)
				Expect(err).ToNot(HaveOccurred())

				server.HTTPTestServer.TLS = &tls.Config{
					Certificates: []tls.Certificate{cert},
				}

				server.HTTPTestServer.StartTLS()
			})

			AfterEach(func() {
				server.Close()
			})

			DirectorRedirect := func(config Config) http.Header {
				h := http.Header{}
				// URL does not include port, creds
				h.Add("Location", "https://"+config.Host+"/info")
				h.Add("Referer", "referer")
				return h
			}

			VerifyHeaderDoesNotExist := func(key string) http.HandlerFunc {
				cKey := http.CanonicalHeaderKey(key)
				return func(w http.ResponseWriter, req *http.Request) {
					for k, _ := range req.Header {
						Expect(k).ToNot(Equal(cKey), "Header '%s' must not exist", cKey)
					}
				}
			}

			It("succeeds making requests and follow redirects with basic auth creds", func() {
				config, err := NewConfigFromURL(server.URL())
				Expect(err).ToNot(HaveOccurred())

				config.Username = "username"
				config.Password = "password"
				config.CACert = validCACert

				logger := boshlog.NewLogger(boshlog.LevelNone)

				director, err := NewFactory(logger).New(config, nil, nil)
				Expect(err).ToNot(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.RespondWith(http.StatusFound, nil, DirectorRedirect(config)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						ghttp.VerifyBasicAuth("username", "password"),
						VerifyHeaderDoesNotExist("Referer"),
						ghttp.RespondWith(http.StatusOK, `{}`),
					),
				)

				_, err = director.Info()
				Expect(err).ToNot(HaveOccurred())
			})

			It("succeeds making requests and follow redirects with token", func() {
				config, err := NewConfigFromURL(server.URL())
				Expect(err).ToNot(HaveOccurred())

				var tokenRetries []bool

				config.TokenFunc = func(retried bool) (string, error) {
					tokenRetries = append(tokenRetries, retried)
					return "auth", nil
				}
				config.CACert = validCACert

				logger := boshlog.NewLogger(boshlog.LevelNone)

				director, err := NewFactory(logger).New(config, nil, nil)
				Expect(err).ToNot(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{"auth"}}),
						ghttp.RespondWith(http.StatusFound, nil, DirectorRedirect(config)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						ghttp.VerifyHeader(http.Header{"Authorization": []string{"auth"}}),
						VerifyHeaderDoesNotExist("Referer"),
						ghttp.RespondWith(http.StatusOK, `{}`),
					),
				)

				_, err = director.Info()
				Expect(err).ToNot(HaveOccurred())

				// First token is fetched without retrying,
				// and on first redirect we forcefully retry
				// since redirects are not currently retried.
				Expect(tokenRetries).To(Equal([]bool{false, true}))
			})

			It("succeeds making requests and follow redirects without any auth", func() {
				config, err := NewConfigFromURL(server.URL())
				Expect(err).ToNot(HaveOccurred())

				config.CACert = validCACert

				logger := boshlog.NewLogger(boshlog.LevelNone)

				director, err := NewFactory(logger).New(config, nil, nil)
				Expect(err).ToNot(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						VerifyHeaderDoesNotExist("Authorization"),
						ghttp.RespondWith(http.StatusFound, nil, DirectorRedirect(config)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						VerifyHeaderDoesNotExist("Authorization"),
						VerifyHeaderDoesNotExist("Referer"),
						ghttp.RespondWith(http.StatusOK, `{}`),
					),
				)

				_, err = director.Info()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with TLS server with invalid certificate", func() {
			var (
				server *ghttp.Server
			)

			BeforeEach(func() {
				server = ghttp.NewUnstartedServer()
				server.HTTPTestServer.StartTLS()
			})

			AfterEach(func() {
				server.Close()
			})

			DirectorRedirect := func(config Config) http.Header {
				h := http.Header{}
				// URL does not include port, creds
				h.Add("Location", "https://"+config.Host+"/info")
				h.Add("Referer", "referer")
				return h
			}

			VerifyHeaderDoesNotExist := func(key string) http.HandlerFunc {
				cKey := http.CanonicalHeaderKey(key)
				return func(w http.ResponseWriter, req *http.Request) {
					for k, _ := range req.Header {
						Expect(k).ToNot(Equal(cKey), "Header '%s' must not exist", cKey)
					}
				}
			}

			It("fails making requests without skip-ssl-validation", func() {
				config, err := NewConfigFromURL(server.URL())
				Expect(err).ToNot(HaveOccurred())

				config.Username = "username"
				config.Password = "password"

				logger := boshlog.NewLogger(boshlog.LevelNone)

				director, err := NewFactory(logger).New(config, nil, nil)
				Expect(err).ToNot(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.RespondWith(http.StatusFound, nil, DirectorRedirect(config)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						ghttp.VerifyBasicAuth("username", "password"),
						VerifyHeaderDoesNotExist("Referer"),
						ghttp.RespondWith(http.StatusOK, `{}`),
					),
				)

				_, err = director.Info()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("x509: certificate signed by unknown authority"))
			})

			It("succeeds making requests with skip-ssl-validation", func() {
				config, err := NewConfigFromURL(server.URL())
				Expect(err).ToNot(HaveOccurred())

				config.Username = "username"
				config.Password = "password"
				config.SkipSslValidation = true

				logger := boshlog.NewLogger(boshlog.LevelNone)

				director, err := NewFactory(logger).New(config, nil, nil)
				Expect(err).ToNot(HaveOccurred())

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						ghttp.VerifyBasicAuth("username", "password"),
						ghttp.RespondWith(http.StatusFound, nil, DirectorRedirect(config)),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/info"),
						ghttp.VerifyBasicAuth("username", "password"),
						VerifyHeaderDoesNotExist("Referer"),
						ghttp.RespondWith(http.StatusOK, `{}`),
					),
				)

				_, err = director.Info()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
