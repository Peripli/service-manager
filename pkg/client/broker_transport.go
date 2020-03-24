package client

import (
	"crypto/tls"
	"github.com/Peripli/service-manager/pkg/httpclient"
	"net/http"
)

type BrokerTransport struct {
	tlsConfig *tls.Config
}

func NewBrokerTransport(tlsConfig *tls.Config) *BrokerTransport {
	bt := &BrokerTransport{}
	bt.tlsConfig = tlsConfig
	return bt
}

func (bt *BrokerTransport) GetTransportWithTLS() (bool, *http.Transport) {

	if len(bt.tlsConfig.Certificates) > 0 {
		transport := http.Transport{}
		httpclient.Configure(&transport)
		transport.TLSClientConfig.Certificates = bt.tlsConfig.Certificates

		//prevents keeping idle connections when accessing to different broker hosts
		transport.DisableKeepAlives = true
		return true, &transport
	}

	return false, nil
}
