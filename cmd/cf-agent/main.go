package main

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/cmd/cf-agent/cf"
	"github.com/Peripli/service-manager/pkg/agent"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, err := cf.DefaultEnv(ctx)
	if err != nil {
		panic(fmt.Sprintf("error creating environment: %s", err))
	}

	proxySettings, err := cf.NewConfig(env)
	if err != nil {
		panic(fmt.Errorf("error loading config: %s", err))
	}

	platformClient, err := cf.NewClient(proxySettings)
	if err != nil {
		panic(fmt.Errorf("error creating CF client: %s", err))
	}

	proxyBuilder, err := agent.New(ctx, cancel, env, &proxySettings.Settings, platformClient)
	if err != nil {
		panic(fmt.Errorf("error creating sbproxy: %s", err))
	}

	proxyBuilder.Build().Run()
}
