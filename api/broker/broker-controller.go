package broker

import (
	"errors"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/sirupsen/logrus"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// Controller broker controller
type Controller struct {
	BrokerStorage storage.Broker
}

func (brokerCtrl *Controller) addBroker(response http.ResponseWriter, request *http.Request) error {
	logrus.Debugf("POST %s", request.RequestURI)

	broker := rest.Broker{}
	if err := rest.ReadJSONBody(request, &broker); err != nil {
		return err
	}

	if broker.Name == "" || broker.BrokerURL == "" || broker.Credentials == nil {
		return rest.CreateErrorResponse(
			errors.New("Not all required properties are provided in the request body"),
			http.StatusBadRequest,
			"BadRequest")
	}

	uuid, err := uuid.NewV4()
	if err != nil {
		logrus.Error("Could not generate GUID")
		return err
	}

	broker.ID = uuid.String()

	currentTime := time.Now().UTC()
	broker.CreatedAt = currentTime
	broker.UpdatedAt = currentTime

	brokerStorage := brokerCtrl.BrokerStorage
	err = brokerStorage.Create(&broker)
	if err != nil {
		if err == storage.ErrUniqueViolation {
			return rest.CreateErrorResponse(err, http.StatusConflict, "Conflict")
		}
		return err
	}

	broker.Credentials = nil
	return rest.SendJSON(response, http.StatusCreated, broker)
}

func (brokerCtrl *Controller) getBroker(response http.ResponseWriter, request *http.Request) error {
	logrus.Debugf("GET %s", request.RequestURI)

	brokerID := mux.Vars(request)["broker_id"]
	brokerStorage := brokerCtrl.BrokerStorage
	broker, err := brokerStorage.Get(brokerID)
	if err != nil {
		if err == storage.ErrNotFound {
			return rest.CreateErrorResponse(err, http.StatusNotFound, "NotFound")
		}
		return err
	}
	return rest.SendJSON(response, http.StatusOK, broker)
}

func (brokerCtrl *Controller) getAllBrokers(response http.ResponseWriter, request *http.Request) error {
	logrus.Debugf("GET %s", request.RequestURI)

	brokerStorage := brokerCtrl.BrokerStorage
	brokers, err := brokerStorage.GetAll()
	if err != nil {
		return err
	}

	type brokerResponse struct {
		Brokers []rest.Broker `json:"brokers"`
	}
	return rest.SendJSON(response, http.StatusOK, brokerResponse{brokers})
}

func (brokerCtrl *Controller) deleteBroker(response http.ResponseWriter, request *http.Request) error {
	logrus.Debugf("DELETE %s", request.RequestURI)

	brokerID := mux.Vars(request)["broker_id"]
	brokerStorage := brokerCtrl.BrokerStorage
	if err := brokerStorage.Delete(brokerID); err != nil {
		if err == storage.ErrNotFound {
			return rest.CreateErrorResponse(err, http.StatusNotFound, "NotFound")
		}
		return err
	}
	return rest.SendJSON(response, http.StatusOK, map[string]int{})
}

func (brokerCtrl *Controller) updateBroker(response http.ResponseWriter, request *http.Request) error {
	logrus.Debugf("PATCH %s", request.RequestURI)

	broker := rest.Broker{}
	if err := rest.ReadJSONBody(request, &broker); err != nil {
		logrus.Error("Invalid request body")
		return err
	}

	broker.ID = mux.Vars(request)["broker_id"]
	broker.UpdatedAt = time.Now().UTC()

	brokerStorage := brokerCtrl.BrokerStorage
	if err := brokerStorage.Update(&broker); err != nil {
		if err == storage.ErrNotFound {
			return rest.CreateErrorResponse(err, http.StatusNotFound, "NotFound")
		} else if err == storage.ErrUniqueViolation {
			return rest.CreateErrorResponse(err, http.StatusConflict, "Conflict")
		}
		return err
	}
	updatedBroker, err := brokerStorage.Get(broker.ID)
	if err != nil {
		logrus.Error("Failed to retrieve updated broker")
		return err
	}

	return rest.SendJSON(response, http.StatusOK, updatedBroker)
}
