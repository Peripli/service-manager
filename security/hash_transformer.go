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

package security

import (
	"code.cloudfoundry.org/cli/cf/errors"
	"golang.org/x/crypto/bcrypt"
)

type HashTransformer struct {
}

func (*HashTransformer) Transform(secret []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(secret, bcrypt.DefaultCost)
}

func (*HashTransformer) Reverse(cipher []byte) ([]byte, error) {
	return nil, errors.New("Hashing is a non-reversible operation")
}
