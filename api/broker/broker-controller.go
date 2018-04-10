package broker

import (
	"encoding/json"
	"net/http"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/util"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

type Controller struct{}

func (brokerCtrl *Controller) addBroker(w http.ResponseWriter, r *http.Request) error {
	decoder := json.NewDecoder(r.Body)
	broker := rest.Broker{}
	err := decoder.Decode(&broker)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	return nil
}

func (brokerCtrl *Controller) getBroker(w http.ResponseWriter, r *http.Request) error {
	logrus.Debugf("GET %s", r.RequestURI)

	brokerID := mux.Vars(r)["broker_id"]
	brokerStorage := storage.Get().Broker()
	broker, err := brokerStorage.Get(brokerID)
	if err != nil {
		if err == storage.NotFoundError {
			return rest.CreateErrorResponse(err, http.StatusNotFound, "NotFound")
		}
		return err
	}
	util.SendJSON(w, 200, broker)
	return nil
}

func (brokerCtrl *Controller) getAllBrokers(w http.ResponseWriter, r *http.Request) error {
	logrus.Debugf("GET %s", r.RequestURI)

	brokerStorage := storage.Get().Broker()
	brokers, err := brokerStorage.GetAll()
	if err != nil {
		return err
	}

	type response struct {
		Brokers []rest.Broker `json:"brokers"`
	}
	util.SendJSON(w, 200, response{brokers})
	return nil
}

func (brokerCtrl *Controller) deleteBroker(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (brokerCtrl *Controller) updateBroker(w http.ResponseWriter, r *http.Request) error {
	return nil
}
