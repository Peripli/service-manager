# Example Scenarios

The following page contains some example scenarios in which the Service Manager can be used.

## Enforcing Policies

1. Service Manager manages all registered brokers and platforms. It can enforce policies as to which services and plans are visible in each platform.

2. Service Manager also manages all service instances - it can enforce policies as to which instance can be shared across the managed plaforms.

3. As all OSB calls go through the Service Manager, it can also enforce quota limits (for example how many instances of a particular plan one can create/consume).

## Service Sharing Examples

1. An application running in SAPCP CloudFoundry can consume services provided by the different evironments offered by the SAPCP Multicloud (e.g. from multiple CF installations as well as K8S clusters).

2. The same is valid for scenarios accross cloud providers - the app very well consume service-x from Azure, service-y from GCP and service-z from SAPCP. The only requirement is that the services offered by these cloud providers are exposed via service brokers.

## Service Instance Sharing Examples

1. An application may consist of multiple microservices running on different platforms. Sharing the same RabbitMQ service instance would allow pubsub communication between those microservices. They can also share the same Postgresql service instance if needed.

2. The application can be moved to a different platform and still use the same service instances thatit previously used.
