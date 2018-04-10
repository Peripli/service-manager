package broker

import (
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
	rest.ReadJSONBody(request, &broker)

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
		return err
	}

	return nil
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
	rest.SendJSON(response, 200, broker)
	return nil
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
	rest.SendJSON(response, 200, brokerResponse{brokers})
	return nil
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
	rest.SendJSON(response, 200, map[string]int{})
	return nil
}

func (brokerCtrl *Controller) updateBroker(response http.ResponseWriter, request *http.Request) error {
	return nil
}
