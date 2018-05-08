# Run Service Manager on CF

## Prerequisites

You need to install and configure docker on the machine that you are building.

## Build docker image

```sh
docker build -t "<repo_url>/<user_name>/<image_name>:<tag>" -f Dockerfile .
```

## Push image to a docker repository

```sh
docker push "<repo_url>/<user_name>/<image_name>:<tag>"
```

## Create PostgreSQL service instance in your CF environment

```sh
cf create-service <postgres_service_name> <plan_name> <postgre_instance_name>
```

## Create manifest.yml file

Rename `manifest.yml.template` to `manifest.yml` and replace:

 * <application_name> with the desired name of your Service Manager application.
 * <repo_url>/<user_name>/<image_name>:<tag> with your docker image url.
 * <postgre_instance_name> with the instance name of you PostgreSQL service.

## Push the application

```sh
cf push -f manifest.yml
```
