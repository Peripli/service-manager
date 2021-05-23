/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package client

import (
	"github.com/Peripli/service-manager/pkg/httpclient"
	"github.com/Peripli/service-manager/pkg/types"
	"net/http"
)

func GetTransportWithTLS(broker *types.ServiceBroker) (*http.Transport,error) {
	transport := http.Transport{}
	httpclient.ConfigureTransport(&transport)
	tlsConfig,err:=broker.GetTLSConfig()
	if err!=nil{
		return nil,err
	}
	if tlsConfig!=nil {
		if len(transport.TLSClientConfig.Certificates) > 0 {
			transport.TLSClientConfig.Certificates = append(transport.TLSClientConfig.Certificates, tlsConfig.Certificates...)
		}else{
			transport.TLSClientConfig.Certificates = tlsConfig.Certificates
		}
	}

	//prevents keeping idle connections when accessing to different broker hosts
	transport.DisableKeepAlives = true
	return &transport,nil
}
