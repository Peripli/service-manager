# Run Service Manager on CF

## Create PostgreSQL service instance in your CF environment

```sh
cf create-service <postgres_service_name> <plan_name> <postgre_instance_name>
```

## Update manifest.yml file

Replace in `manifest.yml`:

 * <application_name> with the desired name of your Service Manager application.
 * <postgre_instance_name> with the instance name of you PostgreSQL service.

## Push the application

From the root of your project execute:

Linux:
```sh
cf push -f deployment/cf/manifest.yml
```

Windows:
```sh
cf push -f deployment\cf\manifest.yml
```
