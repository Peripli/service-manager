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

package serverfakes

import "github.com/Peripli/service-manager/api/info"

func ConfigureEnvironmentUnmarshalInfo(environment *FakeEnvironment) {
	response := info.Details{TokenIssuer: "https://example.com"}
	environment.UnmarshalStub = func(value interface{}) error {
		val, ok := value.(*info.Details)
		if ok {
			*val = response
		}
		return nil
	}
}
