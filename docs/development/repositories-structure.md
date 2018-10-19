# Repository Structure

## Peripli/service-manager

This repository contains the source code for the Service Manager component. It also contains reuse logic (pkg) that is used in the proxy components' source code.

    .
    ├── api                         # Service Manager API controllers and filters
    │   ├── broker                  # SM Service Broker API controller
    │   ├── catalog                 # SM Aggregated Catalog API controller
    │   ├── filters                 # SM Filters (authn, logging, recovery, etc...)
    │   ├── healthcheck             # SM Healthz API controller
    │   ├── info                    # SM Info API controller
    │   ├── osb                     # SM OSB API controller
    │   └── platform                # SM Platforms API controller
    ├── cf                          # cloudfoundry specific logic; required for running SM on CF
    ├── charts                      # Helm charts for deployment
    │   ├── service-broker-proxy    # Helm chart for deploying the k8s service broker proxy
    │   └── service-manager         # Helm chart for deploying the service manager
    ├── cmd                         # Commands for running the applications
    │   ├── sbproxy                 # Commands for running the service broker proxies
    │   │   ├── cf                  # Command for running the CloudFoundry service broker proxy   
    │   │   └── k8s                 # Command for running the Kubernetes service broker proxy
    │   └── service-manager         # Command for running the service-manager
    ├── config                      # SM options; startup config loaded from env, flags, cfg files
    ├── docs                        # Documentation
    ├── pkg                         # Contains reusable packages   
    │   ├── env                     # Environment abstraction; allows loading config from env, pflags and config files
    │   ├── health                  # Means to define health indicators
    │   ├── log                     # Logging abstraction; allows logging with CorrelationIDs and other request scoped fields
    │   ├── sbproxy                 # entrypoint of the framework for instantiation of the application
    │   │   ├── osb                 # Configurations for the OSB proxying API provided by `github.com/Peripli/service-manager/api/osb`
    │   │   ├── platform            # Interfaces to be implemented by consumers of this framework
    │   │   ├── reconcile           # reconcilation job; reconciles the current state (platform) and the desired state (obtained from SM)
    │   │   └── sm                  # client logic for requesting the desired state from SM
    │   ├── server                  # Allows to set up a server with an API that handles graceful shutdowns
    │   ├── sm                      # Creates the service manager application
    │   ├── types                   # Types used in the Service Manager
    │   ├── util                    # Helpers for handling errors, sending and processing requests and responses
    │   └── web                     # Extension points of the Service Manager
    ├── security                    # Contains logic for setting security
    ├── storage                     # Abstract logic around the Service Manager persistent storage
    │   ├── postgres                # PostgreSQL implementation of the storage
    └── test                        # Integration and e2e tests
 
**Note:** vendor folder is not checked out in scm. After cloning the repository `dep ensure --vendor-only` is required.

## Peripli/service-manager-cli

This repository contains the implementation for the Service Manager CLI `smctl`.

**Note:** vendor folder is not checked out in scm. After cloning the repository `dep ensure --vendor-only` is required.
