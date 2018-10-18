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
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/spf13/pflag"
)

func main() {
	ctx, cancel := util.HandleInterrupts()
	defer cancel()

	env := sm.DefaultEnv(func(set *pflag.FlagSet) {
		set.Set("file.location", "cmd/service-manager")
	})
	serviceManager := sm.New(ctx, env).Build()
	serviceManager.Run()
}
