# Components

Each component has its own source repository:

### [Service Manager Core](https://github.com/Peripli/service-manager)

The central registry for service broker and platform registration, as well as for tracking of all service instances. This component will act as the single source of truth for platforms managed by the Service Manager with regards to existing brokers. It acts as a platform to the service brokers that are registered in it and as a Service Broker for the platforms that it manages.

### [Service Manager command line tool (CLI)](https://github.com/Peripli/service-manager-cli)

The official tool to communicate directly with the Service Manager and manage service broker and platform registrations.

### [Cloud Foundry Broker Proxy](https://github.com/Peripli/service-broker-proxy-cf)

Cloud Foundry specific implementation using the **Common Broker Proxy framework**.

This proxy implementation ensures the brokers registered in the Cloud Controller are in sync with those in the Service Manager and that their services are visible in the marketplace.

### [Kubernetes Broker Proxy](https://github.com/Peripli/service-broker-proxy-k8s)

Kubernetes specific implementation using the **Common Broker Proxy framework**.

This proxy implementation ensures the brokers registered in the Service Catalog are in sync with those in the Service Manager.

### [Common Broker Proxy framework](https://github.com/Peripli/service-broker-proxy)

Contains a base framework for writing Service Manager service broker proxies. It is meant to be used as a dependency for platform-specific wrappers. Each wrapper needs to have a main function that instantiates and runs an sbproxy (check [sbproxy.go](https://github.com/Peripli/service-broker-proxy/blob/master/pkg/sbproxy/sbproxy.go#L143) for details). The wrapper also has to implement the interfaces in the [platform package](https://github.com/Peripli/service-broker-proxy/tree/master/pkg/platform) in order to specify how propagating of broker registration and visiblity events should be sent to the corresponding platform that the proxy application will represent.

The proxy framework provides common logic for the service broker proxy application that includes setting up a server with graceful shutdown that has an OSB API which forwards calls to the specified Service Manager installation. It also includes the logic for configuring the proxy application - one can do that by using environment variables, pflags or the application.yml config file. Also, it includes the logic for a background task that takes care to fetch the desired state (service brokers with their catalogs) from the Service Manager and to apply it to the current state (in the actual platform).

In general, a broker proxy is deployed in each platform. Each proxy checks on regular intervals what brokers are registered in the Service Manager and makes sure the brokers in the specific platform reflect that by either registering itself in place of each new broker, updating existing broker registrations or deleting broker registrations that are no longer in SM, enabling/disabling service access, etc.
When calls from the platform hit the broker proxy, it forwards them to SM with information about the actual service broker that the call is meant for.
