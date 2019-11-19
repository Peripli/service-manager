package main

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/cmd/k8s-agent/k8s/client"
	"github.com/Peripli/service-manager/cmd/k8s-agent/k8s/config"

	"github.com/Peripli/service-manager/pkg/agent"

	"github.com/spf13/pflag"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, err := agent.DefaultEnv(ctx, func(set *pflag.FlagSet) {
		config.CreatePFlagsForK8SClient(set)
	})
	if err != nil {
		panic(fmt.Errorf("error creating environment: %s", err))
	}

	proxySettings, err := config.NewConfig(env)
	if err != nil {
		panic(fmt.Errorf("error loading config: %s", err))
	}

	platformClient, err := client.NewClient(proxySettings)
	if err != nil {
		panic(fmt.Errorf("error creating K8S client: %s", err))
	}

	proxyBuilder, err := agent.New(ctx, cancel, env, &proxySettings.Settings, platformClient)
	if err != nil {
		panic(fmt.Errorf("error creating sbproxy: %s", err))
	}

	proxyBuilder.Build().Run()
}
