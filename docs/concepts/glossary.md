# Glossary

This Service Manager documents use the terminology of the [Open Service Broker (OSB) API](https://github.com/openservicebrokerapi/servicebroker/).

Additionally, the following terms are used throughout the documents:

- *Platform Type*: A concrete implementation of a platform. Examples are Cloud Foundry and Kubernetes.

- *Platform Instance*: A deployment of a platform. For example, a Kubernetes cluster.

- *Cloud Landscape*: A collection of Platform Instances of the same or different types.
  In many cases, a Cloud Landscape is operated by one cloud provider,
  but the services may be also consumable by platforms outside the landscape.

- *Service Manager*: A component that acts as a platform as per OSB API and exposes a platform API.
  It allows the management and registration of service brokers and platform instances.
  Also acts as a central service broker.

- *Service Broker Proxy*:	A component that is associated with a platform that is registered at the Service Manager
  and is a service broker as per the OSB API specification.
  The proxy replicates the changes in the Service Manager into the platform instance in which the proxy resides.
  Service Brokers Proxies are in charge of registering and deregistering themselves at the platform it is responsible for.