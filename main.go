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
	"context"
	"fmt"

	"os"
	"os/signal"

	"github.com/Peripli/service-manager/app"
	"github.com/Peripli/service-manager/cf"
	"github.com/Peripli/service-manager/config"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	set := config.SMFlagSet()
	config.AddPFlags(set)

	env, err := config.NewEnv(set)
	if err != nil {
		panic(fmt.Sprintf("error loading environment: %s", err))
	}
	if err := cf.SetCFOverrides(env); err != nil {
		panic(fmt.Sprintf("error setting CF environment values: %s", err))
	}
	cfg, err := config.New(env)
	if err != nil {
		panic(fmt.Sprintf("error loading configuration: %s", err))
	}

	parameters := &app.Parameters{
		Settings: cfg,
	}
	srv, err := app.New(ctx, parameters)
	if err != nil {
		panic(fmt.Sprintf("error creating SM server: %s", err))
	}

	srv.Run(ctx)
}

func handleInterrupts(ctx context.Context, cancelFunc context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt)
	go func() {
		select {
		case <-term:
			logrus.Error("Received OS interrupt, exiting gracefully...")
			cancelFunc()
		case <-ctx.Done():
			return
		}
	}()
}
