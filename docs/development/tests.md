# Running tests

For testing we use the frameworks - [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/)

## Prerequisites

* PostgreSQL running standalone or in a docker container
    ```console
    docker run -d -p 5432:5432 --name prod_postgres --user postgres postgres
    ```

## Unit and integration tests

Currently unit and integration tests are run with one command.

To execute tests, run the following command:

```console
make test
```

If PostgreSQL is not running with default settings, the connection URI can be changed by providing a commandline argument:

```console
make test TEST_FLAGS="--storage.uri=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
```

## Coverage

To generate test coverage report run the following command:

```console
make coverage
```

If PostgreSQL is not running with default settings, the connection URI can be changed by providing a commandline argument:

```console
make coverage TEST_FLAGS="--storage.uri=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
```

The above command will create a file called `coverage.html` at the root of the project.

**Note**: All commandline arguments that can be used to configure the Service Manager on startup can also be passed to `make test` and `make coverage` via `TEST_FLAGS`.