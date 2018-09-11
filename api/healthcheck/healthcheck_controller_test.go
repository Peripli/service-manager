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
package healthcheck

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage/storagefakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Healthcheck Controller Suite")
}

var _ = Describe("Healthcheck controller", func() {

	var availableResponse, unavailableResponse []byte

	BeforeSuite(func() {
		var err error
		availableResponse, err = json.Marshal(statusRunningResponse)
		Expect(err).ToNot(HaveOccurred())
		unavailableResponse, err = json.Marshal(statusStorageFailureResponse)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("healthCheck", func() {
		Context("when ping returns error", func() {
			It("should respond with 503", func() {
				resp, err := createController(errors.New("expected")).healthCheck(&web.Request{Request: &http.Request{}})
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
				Expect(string(resp.Body)).To(Equal(string(unavailableResponse)))
			})
		})

		Context("when ping returns nil", func() {
			It("should respond with 200", func() {
				resp, err := createController(nil).healthCheck(&web.Request{Request: &http.Request{}})
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(string(resp.Body)).To(Equal(string(availableResponse)))
			})
		})
	})

})

func createController(pingError error) *Controller {
	fakeStorage := &storagefakes.FakeStorage{}
	fakeStorage.PingReturns(pingError)
	return &Controller{
		Storage: fakeStorage,
	}
}
