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




func GetInstance(doRequestWithClient util.DoRequestWithClientFunc, brokerAPIVersion string, ctx context.Context, broker *types.ServiceBroker, instanceId string) ([]byte, error) {

	log.C(ctx).Debugf("Attempting to fetch service instance from broker with name %s and URL %s", broker.Name, broker.BrokerURL)
	brokerClient, err := client.NewBrokerClient(broker, doRequestWithClient)
	if err != nil {
		return nil, err
	}

	response, err := brokerClient.SendRequest(ctx, http.MethodGet, fmt.Sprintf(serviceInstanceURL, broker.BrokerURL, instanceId),
		map[string]string{}, nil, map[string]string{
			brokerAPIVersionHeader: brokerAPIVersion,
		})
	if err != nil {
		log.C(ctx).WithError(err).Errorf("Error while forwarding request to service broker %s", broker.Name)
		return nil, &util.HTTPError{
			ErrorType:   "ServiceBrokerErr",
			Description: fmt.Sprintf("could not reach service broker %s at %s", broker.Name, broker.BrokerURL),
			StatusCode:  http.StatusBadGateway,
		}
	}

	if response.StatusCode != http.StatusOK {
		log.C(ctx).WithError(err).Errorf("error fetching service instance %s from broker with name %s: %s", instanceId, broker.Name, util.HandleResponseError(response))
		return nil, &util.HTTPError{
			ErrorType:   "ServiceBrokerErr",
			Description: fmt.Sprintf("error fetching service instance %s from broker with name %s: %s", instanceId, broker.Name, response.Status),
			StatusCode:  http.StatusBadRequest,
		}
	}

	var responseBytes []byte
	if responseBytes, err = util.BodyToBytes(response.Body); err != nil {
		if nErr, ok := err.(net.Error); ok && nErr.Timeout() {
			log.C(ctx).WithError(err).Errorf("error fetching service instance %s from broker with name %s: %s: time out", instanceId, broker.Name, err)
			return nil, &util.HTTPError{
				ErrorType:   "ServiceBrokerErr",
				Description: fmt.Sprintf("error fetching service instance %s from broker with name %s: timed out", instanceId, broker.Name),
				StatusCode:  http.StatusGatewayTimeout,
			}
		}
		return nil, fmt.Errorf("error getting content from body of response with status %s: %s", response.Status, err)
	}

	log.C(ctx).Debugf("Successfully fetched service instance %s from broker with name %s and URL %s", instanceId, broker.Name, broker.BrokerURL)

	return responseBytes, nil

}
