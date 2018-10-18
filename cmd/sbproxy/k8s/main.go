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

package main

import (
	"fmt"

	"github.com/Peripli/service-manager/cmd/sbproxy/k8s/app"
	"github.com/Peripli/service-manager/pkg/sbproxy"
	"github.com/Peripli/service-manager/pkg/util"

	"github.com/spf13/pflag"
)

func main() {
	ctx, cancel := util.HandleInterrupts()
	defer cancel()

	env := sbproxy.DefaultEnv(func(set *pflag.FlagSet) {
		set.Set("file.location", "cmd/sbproxy/k8s")
		app.CreatePFlagsForK8SClient(set)
	})

	platformConfig, err := app.NewConfig(env)
	if err != nil {
		panic(fmt.Errorf("error loading config: %s", err))
	}

	platformClient, err := app.NewClient(platformConfig)
	if err != nil {
		panic(fmt.Errorf("error creating K8S client: %s", err))
	}

	proxyBuilder := sbproxy.New(ctx, env, platformClient)
	proxy := proxyBuilder.Build()

	//proxy.Server.Use(middleware.BasicAuth(platformConfig.Reg.User, platformConfig.Reg.Password))

	proxy.Run()
}
