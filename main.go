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
	"net/http"
	"os"

	"github.com/Peripli/service-manager/pkg/caller"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
)

type afilter struct {
	name string
}

func (s *afilter) Name() string {
	return s.name
}

func (*afilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	panic("implement me")
}

func (*afilter) FilterMatchers() []web.FilterMatcher {
	panic("implement me")
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stdout)
	config := caller.DefaultConfig("testCall")
	config.FallbackHandler = func(e error) error {
		fmt.Printf("Handled error %s", e)
		return nil
	}
	caller, err := caller.New(config)
	if err != nil {
		panic(err)
	}
	request, err := http.NewRequest(http.MethodGet, "https://sdfsdf.com/BB%D0%B8-280-%D0%B3%D1%80-", nil)
	if err != nil {
		panic(err)
	}
	r := &web.Request{Request: request}
	response, err := caller.Call(r)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v", response)
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	//
	// env := sm.DefaultEnv()
	// builder := sm.New(ctx, cancel, env)
	// builder.RegisterFilters(&afilter{"Filter:Name"}, &afilter{"A   USD:woijf:afasd"})
	// serviceManager := builder.Build()
	// serviceManager.Run()
}
