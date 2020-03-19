package osb

import (
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"log"
	"net/url"
)

var _ = Describe("OSB Controller test", func() {

	var brokerTLS types.ServiceBroker

	BeforeEach(func() {
		brokerTLS = types.ServiceBroker{
			Base: types.Base{
				ID:     "123",
				Labels: map[string][]string{},
				Ready:  true,
			},
			Name:      "tls-broker",
			BrokerURL: "url",
			Credentials: &types.Credentials{
				Basic: &types.Basic{
					Username: "user",
					Password: "pass",
				},
				TLS: &types.TLS{
					Certificate: common.ClientCertificate,
					Key:         common.ClientKey,
				},
			},
		}
	})

	Describe("test osb create proxy", func() {
		logger := logrus.Entry{}
		targetBrokerURL, err := url.Parse("http://example.com/proxy/")
		if err != nil {
			log.Fatal(err)
		}

		It("create proxy with tls should return a new reverse proxy with its own tls setting", func() {
			reverseProxy, _ := buildProxy(targetBrokerURL, &logger, &brokerTLS)
			Expect(reverseProxy).NotTo(Equal(nil))
			reverseProxy2, _ := buildProxy(targetBrokerURL, &logger, &brokerTLS)
			Expect(reverseProxy2.Transport == reverseProxy.Transport).To(Equal(false))
		})
	})
})
