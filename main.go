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
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-lib/metrics"

	"github.com/Peripli/service-manager/api/extensions/security"

	"github.com/Peripli/service-manager/pkg/env"

	"github.com/Peripli/service-manager/config"

	"github.com/Peripli/service-manager/pkg/sm"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
)

func main() {

	jagerCfg :=  jaegercfg.Configuration{
		ServiceName: "service_manager",
		Sampler:     &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter:    &jaegercfg.ReporterConfig{
			LogSpans: true,
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, err := jagerCfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(tracer)

	defer closer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env, err := env.Default(ctx, config.AddPFlags)
	if err != nil {
		panic(err)
	}

	cfg, err := config.New(env)
	if err != nil {
		panic(err)
	}

	serviceManager, err := sm.New(ctx, cancel, env, cfg)
	if err != nil {
		panic(err)
	}
	if err := security.Register(ctx, cfg, serviceManager); err != nil {
		panic(err)
	}

	serviceManager.Build().Run()
}
