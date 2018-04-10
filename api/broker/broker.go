/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

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

const (
	ApiVersion = "v1"
	Root       = "service_brokers"
	Url        = "/" + ApiVersion + "/" + Root
)

type Controller struct{}

func (brokerCtrl *Controller) Routes() []rest.Route {
	return []rest.Route{
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodPost,
				Path:   Url,
			},
			Handler: brokerCtrl.addBroker,
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodGet,
				Path:   Url + "/{broker_id}",
			},
			Handler: brokerCtrl.getBroker,
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodGet,
				Path:   Url,
			},
			Handler: brokerCtrl.getAllBrokers,
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodDelete,
				Path:   Url + "/{broker_id}",
			},
			Handler: brokerCtrl.deleteBroker,
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodPatch,
				Path:   Url + "/{broker_id}",
			},
			Handler: brokerCtrl.updateBroker,
		},
	}
}

type allBrokersResponse struct {
	Brokers []*rest.Broker `json: "brokers"`
}

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
	logrus.Debugf("GET to %s", r.RequestURI)
	vars := mux.Vars(r)

	brokerStorage := storage.Get().Broker()
	broker, _ := brokerStorage.Get(vars["broker_id"])
	brokerModel := broker.ConvertToRestModel()
	brokerJSON, _ := json.Marshal(&brokerModel)
	util.SendJSON(w, 200, brokerJSON)
	return nil
}

func (brokerCtrl *Controller) getAllBrokers(w http.ResponseWriter, r *http.Request) error {
	/*
		logrus.Debugf("GET to %s", r.RequestURI)
		brokerStorage := storage.Get().Broker()
		brokers, err := brokerStorage.GetAll()
		if err != nil {
			return errors.New("Unable to get all brokers")
		}
		result, errJSON := json.Marshal(&allBrokersResponse{brokers})
		if errJSON != nil {
			return errors.New("Unable to serialize brokers")
		}

		util.SendJSON(w, 200, result)
	*/
	return nil
}

func (brokerCtrl *Controller) deleteBroker(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (brokerCtrl *Controller) updateBroker(w http.ResponseWriter, r *http.Request) error {
	return nil
}
