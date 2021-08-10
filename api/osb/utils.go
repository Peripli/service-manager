package osb

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/client"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"net"
	"net/http"
)

func Get(doRequestWithClient util.DoRequestWithClientFunc, brokerAPIVersion string, ctx context.Context, broker *types.ServiceBroker, url string, resourceType string) ([]byte, error) {

	log.C(ctx).Debugf("attempting to fetch %s from URL %s and broker with name %s", resourceType, url, broker.Name)
	brokerClient, err := client.NewBrokerClient(broker, doRequestWithClient,ctx)
	if err != nil {
		return nil, err
	}
	response, err := brokerClient.SendRequest(ctx, http.MethodGet, url,
		map[string]string{}, nil, map[string]string{
			brokerAPIVersionHeader: brokerAPIVersion,
		})
	if err != nil {
		log.C(ctx).WithError(err).Errorf("error while forwarding request to service broker %s", broker.Name)
		return nil, &util.HTTPError{
			ErrorType:   "ServiceBrokerErr",
			Description: fmt.Sprintf("could not reach service broker %s at %s", broker.Name, broker.BrokerURL),
			StatusCode:  http.StatusBadGateway,
		}
	}

	if response.StatusCode != http.StatusOK {
		log.C(ctx).WithError(err).Errorf("error fetching %s from URL %s and broker with name %s: %s", resourceType, url, broker.Name, util.HandleResponseError(response))
		return nil, &util.HTTPError{
			ErrorType:   "ServiceBrokerErr",
			Description: fmt.Sprintf("error fetching %s from URL %s and broker with name %s: %s", resourceType, url, broker.Name, response.Status),
			StatusCode:  http.StatusBadRequest,
		}
	}

	var responseBytes []byte
	if responseBytes, err = util.BodyToBytes(response.Body); err != nil {
		if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
			log.C(ctx).WithError(err).Errorf("error fetching %s from URL %s and broker with name %s: %s: time out", resourceType, url, broker.Name, err)
			return nil, &util.HTTPError{
				ErrorType:   "ServiceBrokerErr",
				Description: fmt.Sprintf("error fetching %s from URL %s and broker with name %s: timed out", resourceType, url, broker.Name),
				StatusCode:  http.StatusGatewayTimeout,
			}
		}
		return nil, fmt.Errorf("error getting content from body of response from %s with status %s: %s", url, response.Status, err)
	}

	log.C(ctx).Debugf("successfully fetched %s from URL %s and broker with name %s", resourceType, url, broker.Name)

	return responseBytes, nil

}
