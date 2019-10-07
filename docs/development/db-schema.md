# Database Schema

Currently Service Manager supports only PostgreSQL as a database.

The database schema is built using a sequence of SQL scripts located in
[storage/postgres/migrations](../../storage/postgres/migrations).
For each schema version there are two scripts - _up_ and _down_.
So it is possible also to downgrade the schema to a an older version.
Each file name starts with some numbers - these represent the schema version.

These scripts are executed in a specific or by [migrate](https://github.com/golang-migrate/migrate) library
during SM start up. This library records in the database the current schema version,
so subsequent executions do not attempt to recreate the same db objects.
It also uses db locks to prevent parallel migrations in case multple instances of SM start at the same time.

## Upgrade / downgrade db schema
Normally SM upgrade the db schea automatically during startup.
Still in rare cases it is necessary to do this manually. Here is how to do it.

The _migrate_ library also offers a [CLI tool](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate).

Set the path to the [migration scripts](../../storage/postgres/migrations) in `MIGRATIONS` variable.

Set the URL to the database including the credentials in `DATABASE` variable. It should be in this format:
```
"postgres://<username>:<password>@<host>:<port>/<database-name>"
```

If SM is deployed on Cloud Foundry you can do this.
Get the environment of SM application via `cf env` command and find the `uri` parameter within the Postgres service binding.
To access the database from your local machine, you may need to [open a tunnel](https://docs.cloudfoundry.org/devguide/deploy-apps/ssh-apps.html#ssh-common-flags)
like this:
```sh
cf ssh <sm-app> -L <local-port>:<remote-host>:<remote-port>
```

On Windows adapt the shell variable syntax accordingly.

Get the curent schema version:
```sh
migrate -source "file://$MIGRATIONS" -database "$DATABASE" version
```
Go to specific version:
```sh
migrate -source "file://$MIGRATIONS" -database "$DATABASE" goto $SCHEMA_VERSION
```

## Troubleshooting
A common problem when switching SM versions is this error during startup:
```
ERR panic: unbale to construct service-manager builder: error opening storage: error opening storage: could not update database schema: file does not exist
```
The reason usually is that an older version of SM was started after a newer one has updated the db schema.
The older version is unaware of the new schema version and bails out with this error.
Normally SM should be updated only to newer versions.
If for some reason it is necessary to downgrade it to an older version,
then the db schame should be downgraded to the corresponding version too.
See above how to do that.
