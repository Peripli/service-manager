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

// Decision represents a decision to allow or deny further
// processing or to abstain from taking a decision
type Decision int

var decisions = []string{"Allow", "Deny", "Abstain"}

const (
	// Allow represents decision to allow to proceed
	Allow Decision = iota

	// Deny represents decision to deny to proceed
	Deny

	// Abstain represents a decision to abstain from deciding - let another component decide
	Abstain
)

// String implements Stringer and converts the decision to human-readable value
func (a Decision) String() string {
	return decisions[a]
}
