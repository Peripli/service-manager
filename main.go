package main

import (
	"context"

	"github.com/Peripli/service-manager/pkg/servicemanager"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sm := servicemanager.New(ctx, cancel)
	sm.Run()
}
