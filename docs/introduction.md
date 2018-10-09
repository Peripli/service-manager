## Motivation

With Cloud Landscapes becoming bigger and more diverse, managing services is getting more difficult and new challenges arise:

* Cloud providers are facing an increasing number of Platform Types, Platform Instances, supported IaaS and regions.
At the same time, the number of services is increasing.
Registering and managing a big amount of service brokers at a huge number of Platform Instances is infeasible.
A central, Platform Instance independent component is needed to allow a sane management of service brokers.

* So far, service instances are only accessible in the silo (platform) where they have been created.
But there are use-cases that require sharing of a service instance across platforms.
For example, a database created in Kubernetes should be accessible in Cloud Foundry.

<img src="Services-Platforms.png" alt="Services and platforms diagram" width="300"/>

A standardized way is needed for managing service broker registrations and propagating them to the registered Platform Instances when necessary.
Also there should be a mechanism for tracking service instances creation that allows sharing of service instances across Platform Instances.

<img src="Services-SM-Platforms.png" alt="SM between services and platforms diagram" width="300"/>

## How it works
![Service Manager diagram](SM-overview.png)

The Service Manager consists of multiple parts.
The main part is the core component.
It is the central registry for service broker and platform registration, as well as for tracking of all service instances.
This core component communicates with the registered brokers and acts as a platform per Open Service Broker specification for them.

In each Platform Instance resides a component called the Service Broker Proxy.
It is the substitute for all brokers registered at the Service Manager in order to replicate broker registration and access visibility changes in the corresponding Platform Instance. It also  delegates lifecycle operations to create/delete/bind/unbind service instances from the corresponding Platform Instance to the Service Manager and the services registered there.

When a broker is registered or deregistered with the Service Manager, the Service Broker Proxy registers or deregisters itself with the Platform Instance on behalf of this service broker.
From a Platform Instance point of view, the broker proxy is indistinguishable from the real broker because both implement the OSB API.

When the Platform Instance makes a call to the service broker, for example to provision a service instance, the broker proxy accepts the call, forwards it to the Service Manager, which in turn forwards it to the real broker.
The response follows the same path back to the Platform Instance.
Because all OSB calls go through the Service Manager, it can track all service instances and share them between Platform Instances.
The Service Manager can also enforce different policies at a central place, e.g. service visibility, quota checks, etc. To achieve this, the Service Manager can be extended with custom logic via plugins.
