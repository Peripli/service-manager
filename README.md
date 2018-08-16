# Service Manager

[![Build Status](https://travis-ci.org/Peripli/service-manager.svg?branch=master)](https://travis-ci.org/Peripli/service-manager)
[![Go Report Card](https://goreportcard.com/badge/github.com/Peripli/service-manager)](https://goreportcard.com/report/github.com/Peripli/service-manager)
[![Coverage Status](https://coveralls.io/repos/github/Peripli/service-manager/badge.svg)](https://coveralls.io/github/Peripli/service-manager)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/Peripli/service-manager/blob/master/LICENSE)
[![Docker Repository on Quay](https://quay.io/repository/service-manager/core/status "Docker Repository on Quay")](https://quay.io/repository/service-manager/core)

## What is Service Manager

Service Manager is a central registry for service brokers and platforms registration. It tracks service instances creation and allows sharing of service instances between different Platform Instances.

## Setup Service Manager Components

The overall setup of the solution consist of a single installation of Service Manager component and one or more Service Broker Proxy components that runs on each one of the registered platforms.
For more information check the [specification page](https://github.com/Peripli/specification#how-it-works).

### Run the Service Manager

Currently the Service Manager can be run on CF or on Kubernetes.

For more information see:

* [Run on CF](deployment/cf/README.md)
* [Run on Kubernetes](deployment/k8s/README.md)

As a result of this step you will get a url address of the running Service Manager component.

### Run the Service Broker Proxies

Follow the links to get details how to run Service Broker Proxy component on CF or Kubernetes.

* [Run Service Broker Proxy on CF](https://github.com/Peripli/service-broker-proxy-cf)
* [Run Service Broker Proxy on Kubernetes](https://github.com/Peripli/service-broker-proxy-k8s)

You need to make sure that the Service Manager is visible from the Service Broker Proxies.

### Install smctl

You need to install the [command line tools](https://github.com/Peripli/service-manager-cli) and login to the Service Manager using the `smctl login` command.

### Test the setup

If you deployed the Service Manager, smctl and at least one Service Broker Proxy you can register an OSB compliant broker using the *smctl register-broker* command.

```sh
smctl register-broker <broker_name> <broker_url> <description>
```

The services provided by the broker should appear in the platform instances in which the proxies reside.
For example, in CF you can check with:

```sh
cf service-brokers
```

**Note:** It could take a couple of minutes before the broker appears in the platform instance.


[Run Service Manager on Kubernetes](deployment/k8s/README.md)

## Plugins and Filters

[Plugins and Filters](docs/plugins.md)
