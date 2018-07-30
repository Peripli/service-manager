# Run Service Manager on CF

## Prerequisites

The following must be fulfilled:

* You are logged in CF.
* Go buildpack is installed with Go version 1.10 support.
* PostgreSQL service is available or an external PostgreSQL is installed and accessible from the CF instance.

## Create PostgreSQL service instance in your CF environment

```sh
cf create-service <postgres_service_name> <plan_name> <postgre_instance_name>
```

Alternatively, you can use external PostgreSQL. In this case you need to have a PostgreSQL uri.

## Update manifest.yml file

Replace in `manifest.yml`:

* <postgre_instance_name> with the instance name of your PostgreSQL service. Alternatively, you can use the `STORAGE_URI` environment variable to set external PostgreSQL uri. In this case `STORAGE_NAME` environment variable must not be present and you need to remove the service from the manifest.yml.

## Push the application

From the root of your project execute:

```sh
cf push -f deployment/cf/manifest.yml
```
