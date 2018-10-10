# Running tests

For testing we use the frameworks - [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/)

## Prerequisites

* PostgreSQL running
    - Standalone or...
    - In a docker container
    ```console
    $ docker run -d -p5432:5432 --name prod_postgres --user postgres postgres
    ```

If PostgreSQL is not running with default settings, the connection URI can be changed in file: [application.yml](https://github.com/Peripli/service-manager/blob/master/test/common/application.yml#L9)

## Unit and integration tests

Currently unit and integration tests are run with one command.

To execute tests, run the following command:
```console
$ make test
```

## Coverage

To generate test coverage report run the following command:
```console
$ make coverage
```

The above command will create a file called `coverage.html` at the root of the project.
