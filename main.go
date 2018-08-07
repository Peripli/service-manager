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

package main

import (
		"fmt"

			"github.com/Peripli/service-manager/pkg/web"
)

type LoggingFilter struct {
}

func (*LoggingFilter) Name() string {
	return "LoggingFilter"
}

func (*LoggingFilter) Run(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		fmt.Println("!!!! Entered in filter !!!!")
		return next.Handle(request)
	})
}

func (*LoggingFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("**"),
			},
		},
	}
}

func main() {
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	//
	// env := sm.DefaultEnv()
	// builder := sm.New(ctx, cancel, env)
	// builder.ReplaceFilter(authn.BasicAuthnFilterName, &LoggingFilter{})
	// serviceManager := builder.Build()
	// serviceManager.Run()
	digits := []int{1,2,3,4,5,6,7,8,9,10}
	digits = append(digits[:2], digits[5:]...)
	fmt.Printf("%v", digits)
}
