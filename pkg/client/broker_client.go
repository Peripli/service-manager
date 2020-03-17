package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Peripli/service-manager/pkg/httpclient"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"net/http"
)

type BrokerClient struct {
	tlsConfig                *tls.Config
	broker                   *types.ServiceBroker
	requestWithClientHandler util.DoRequestWithClientFunc
	requestHandlerDecorated  util.DoRequestFunc
}

func New(broker *types.ServiceBroker, requestHandler util.DoRequestWithClientFunc) (*BrokerClient, error) {
	tlsConfig, err := broker.GetTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get client for broker %s: %v", broker.Name, err)
	}
	bc := &BrokerClient{}
	bc.tlsConfig = tlsConfig
	bc.broker = broker
	bc.requestWithClientHandler = requestHandler
	bc.requestHandlerDecorated = bc.authAndTlsDecorator()
	return bc, nil
}

func (bc *BrokerClient) addBasicAuth(req *http.Request) *BrokerClient {
	req.SetBasicAuth(bc.broker.Credentials.Basic.Username, bc.broker.Credentials.Basic.Password)
	return bc
}

func (bc *BrokerClient) GetTransportWithTLS() (bool, *http.Transport) {

	if len(bc.tlsConfig.Certificates) > 0 {
		transport := http.Transport{}
		httpclient.Configure(&transport)
		transport.TLSClientConfig = &tls.Config{}
		transport.TLSClientConfig.Certificates = bc.tlsConfig.Certificates

		//prevents keeping idle connections when accessing to different broker hosts
		transport.DisableKeepAlives = true
		return true, &transport
	}

	return false, nil
}

func (bc *BrokerClient) authAndTlsDecorator() util.DoRequestFunc {
	return func(req *http.Request) (*http.Response, error) {
		client := http.DefaultClient

		if bc.broker.Credentials.Basic.Username != "" && bc.broker.Credentials.Basic.Password != "" {
			bc.addBasicAuth(req)
		}

		useDedicatedClient, transportWithTLS := bc.GetTransportWithTLS()

		if useDedicatedClient {
			client = &http.Client{}
			client.Transport = transportWithTLS
			return bc.requestWithClientHandler(req, client)
		}

		return bc.requestWithClientHandler(req, client)
	}
}

func (bc *BrokerClient) SendRequest(ctx context.Context, method, url string, params map[string]string, body interface{}, headers map[string]string) (*http.Response, error) {
	return util.SendRequestWithHeaders(ctx, bc.requestHandlerDecorated, method, url, params, body, headers)
}
