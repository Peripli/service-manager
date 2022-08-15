# Repository Structure

## Peripli/service-manager

This repository contains the source code for the Service Manager component. It also contains reuse logic (pkg) that is used in the proxy components' source code.

    .
    ├── deployment              # Deployment configurations and srcipts
    │   └── cf                  # cloudfoundry deployment manifest
    │   └── k8s                 # The service catalog API server service-catalog command
    |       └── charts          # Helm charts for deployment
    ├── api                     # Service Manager API controllers and filters
    │   └── broker              # SM Service Broker API controller
    │   └── catalog             # SM Aggregated Catalog API controller
    │   └── filters             # SM Filters (authn, logging, recovery, etc...)
    │   └── healthcheck         # SM Healthz API controller
    │   └── info                # SM Info API controller
    │   └── osb                 # SM OSB API controller
    │   └── platform            # SM Platforms API controller
    ├── cf                      # cloudfoundry specific logic; required for running SM on CF
    ├── config                  # SM options; startup config loaded from env, flags, cfg files
    ├── storage                 # SM storage abstraction
    │   └── postgres            # Postgresql-specific implementation of the SM Storage abstraction
    ├── pkg                     # Contains reusable packages
    │   └── env                 # Environment abstraction; allows loading config from env, pflags and config files
    │   └── log                 # Logging abstraction; allows logging with CorrelationIDs and other request scoped fields
    |   └── security            # Security related components (authentication and authorization interfaces, encryptors, etc)
    │   └── server              # Allows to set up a server with an API that handles graceful shutdowns
    │   └── sm                  # Creates the service manager application
    │   └── types               # Types used in the Service Manager
    │   └── util                # Helpers for handling errors, sending and processing requests and responses
    │   └── web                 # Extension points of the Service Manager
    ├── docs                    # Documentation
    ├── test                    # Integration and e2e tests
    ├── application.yml         # config file with SM options values(lower priority than env vars & pflags)
    ├── go.mod                  # defines the module’s module path
    └── go.sum                  # containing the expected cryptographic hashes of the content of specific module versions

**Note:** vendor folder is not checked out in scm. After cloning the repository `go get` is required.

## Peripli/service-broker-proxy

This repository provides a framework for writing Service Manager Service Broker Proxies. It contains:

* the whole logic for the server, logging, graceful shutdown, configuration loading (from env, pflags, config files), etc...

* OSB API that takes care of proxying OSB calls from the Platform in which the proxy runs to the Service Manager

* state reconcilation job - provides logic for reconcilation of the state of service brokers and service access between the Service Manager (desired state) and platform (current state).

* interfaces that the `Peripli/service-broker-proxy-cf` and `Peripli/service-broker-proxy-k8s` implement in order to talk to the  corresponding platform controller during state reconcilation.

        .
        ├── pkg                     # Contains all reusable packages as part of the framework
        │   └── logging             # Logging extensions and hooks
        │   └── middleware          # Middleware to used as part of the proxy API
        │   └── osb                 # Configurations for the OSB proxying API provided by `github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/api/osb`
        │   └── platform            # Interfaces to be implemented by consumers of this framework
        │   └── sbproxy             # entrypoint of the framework for instantiation of the application
        |       └── reconcile       # reconcilation job; reconciles the current state (platform) and the desired state (obtained from SM)
        │   └── sm                  # client logic for requesting the desired state from SM
        ├── .travis.yml             # travis CI pipeline definition
        ├── go.mod                  # defines the module’s module path
        └── go.sum                  # containing the expected cryptographic hashes of the content of specific module versions

**Note:** vendor folder is not checked out in scm. After cloning the repository `go get` is required.

## Peripli/service-broker-proxy-cf

This repository contains a CF specific implementation of the `Peripli/service-broker-proxy`. It reuses the proxy framework and implements the
necessary interfaces in order to work with service brokers and service access.

    .
    ├── cf                      # Contains implementation of the interfaces specified in `Peripli/service-broker-proxy/tree/master/pkg/platform`
    ├── .travis.yml             # travis CI pipeline definition
    ├── go.mod                  # defines the module’s module path
    └── go.sum                  # containing the expected cryptographic hashes of the content of specific module versions

**Note:** vendor folder is not checked out in scm. After cloning the repository `go get` is required.

## Peripli/service-broker-proxy-k8s

This repository contains a K8S specific implementation of the `Peripli/service-broker-proxy`. It reuses the proxy framework and implements the necessary interfaces in order to work with service brokers.

    .
    ├── charts                  # Helm charts for deployment
    │   └── service-broker-proxy# Helm chart for deploying the k8s service broker proxy
    ├── k8s                     # Contains implementation of the interfaces specified in `Peripli/service-broker-proxy/tree/master/pkg/platform`
    ├── vendor                  # dep-managed dependencies
    ├── go.mod                  # defines the module’s module path
    └── go.sum                  # containing the expected cryptographic hashes of the content of specific module versions

## Peripli/service-manager-cli

This repository contains the implementation for the Service Manager CLI `smctl`.

**Note:** vendor folder is not checked out in scm. After cloning the repository `go get` is required.
