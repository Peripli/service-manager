# Documentation

The `docs` folder contains end-to-end documentation on Service Manager and its components

## Motivation

With Cloud Landscapes becoming bigger and more diverse, managing services is getting more difficult and new challenges arise:

* Cloud providers are facing an increasing number of Platform Types, Platform Instances, supported IaaS and regions. At the same time, the number of services/brokers is increasing. Registering and managing a big amount of service brokers at a huge number of Platform Instances is infeasible. A central, Platform Instance independent component is needed to allow a sane management of service brokers.

* So far, service instances are only accessible in the silo (platform) where they have been created. But there are use-cases that require sharing of a service instance across platforms. For example, a database created in Kubernetes should be accessible in Cloud Foundry.

A standardized way is needed for managing broker registrations and propagating them to the registered Platform Instances when necessary. Also there should be a mechanism for tracking service instances creation that allows sharing of service instances across Platform Instances.

## Content

* [Installation Guide]()
* [Walkthrough]()
* [High-level architecture]()
* [Components]()
* [Notes on Security]()

* [Developer's Guide]()
* [Code Standards]()
* [Extensions]()

* [Additional Resources]()
* [Contributing]()