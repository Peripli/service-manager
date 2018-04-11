package broker

import (
	"errors"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/util"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

type Controller struct{}

func (brokerCtrl *Controller) addBroker(response http.ResponseWriter, request *http.Request) error {
	logrus.Debugf("POST %s", request.RequestURI)

	broker := rest.Broker{}
	if err := rest.ReadJSONBody(request, &broker); err != nil {
		logrus.Debug("Invalid request body")
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
		logrus.Debug("Could not generate GUID")
		return err
	}

	broker.ID = uuid.String()

	currentTime := time.Now().UTC()
	broker.CreatedAt = util.ToRFCFormat(currentTime)
	broker.UpdatedAt = util.ToRFCFormat(currentTime)

	brokerStorage := storage.Get().Broker()
	err = brokerStorage.Create(&broker)
	if err != nil {
		logrus.Debug("Could not persist broker")
		if err == storage.UniqueViolationError {
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
	brokerStorage := storage.Get().Broker()
	broker, err := brokerStorage.Get(brokerID)
	if err != nil {
		if err == storage.NotFoundError {
			return rest.CreateErrorResponse(err, http.StatusNotFound, "NotFound")
		}
		return err
	}
	return rest.SendJSON(response, http.StatusOK, broker)
}

func (brokerCtrl *Controller) getAllBrokers(response http.ResponseWriter, request *http.Request) error {
	logrus.Debugf("GET %s", request.RequestURI)

	brokerStorage := storage.Get().Broker()
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
	brokerStorage := storage.Get().Broker()
	if err := brokerStorage.Delete(brokerID); err != nil {
		if err == storage.NotFoundError {
			return rest.CreateErrorResponse(err, http.StatusNotFound, "NotFound")
		}
		return err
	}
	return rest.SendJSON(response, http.StatusOK, map[string]int{})
}

func (brokerCtrl *Controller) updateBroker(response http.ResponseWriter, request *http.Request) error {
	return nil
}
