# Components

Each component has its own source repository:

### [Service Manager Core](https://github.com/Peripli/service-manager)

The central registry for service broker and platform registration, as well as for tracking of all service instances. This component will act as the single source of truth for platforms integrated with the Service Manager with regards to existing brokers.

### [Service Manager command line tool (CLI)](https://github.com/Peripli/service-manager-cli)

The official tool to communicate directly with a Service Manager instance and manage service broker and platform registrations.

### [Cloud Foundry Broker Proxy](https://github.com/Peripli/service-broker-proxy-cf)

Cloud Foundry specific implementation of the **Common Broker Proxy library**.

This proxy implementation ensures the brokers registered in the Cloud Controller are in sync with those in the Service Manager.

### [Kubernetes Broker Proxy](https://github.com/Peripli/service-broker-proxy-k8s)

Kubernetes specific implementation of the **Common Broker Proxy library**.

This proxy implementation ensures the brokers registered in the Service Catalog are in sync with those in the Service Manager.

### [Common Broker Proxy library](https://github.com/Peripli/service-broker-proxy)

Contains code for writing Service Manager broker proxies. A broker proxy is deployed in each platform. Each proxy checks on regular intervals what brokers are registered in the Service Manager and makes sure the brokers in the specific platform reflect that by either creating new brokers, updating existing ones or deleting removed ones. Afterwards the broker proxy forwards OSB calls from the platform to the Service Manager.

> This component is used by both the Cloud Foundry and Kubernetes broker proxies.
