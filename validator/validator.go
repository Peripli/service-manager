/*
 * Copyright 2018 The Service Manager Authors
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

package validator

import (
	"strings"
)

var (
	reservedSymbolsRFC3986 = strings.Join([]string{
		":", "/", "?", "#", "[", "]", "@", "!", "$", "&", "'", "(", ")", "*", "+", ",", ";", "=",
	}, "")
)

// HasRFC3986ReservedSymbols returns true if input contains any reserver characters as defined in RFC3986 section 2.2
func HasRFC3986ReservedSymbols(input string) bool {
	return strings.ContainsAny(input, reservedSymbolsRFC3986)
}
