# Running tests

For testing we use the frameworks - [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/)

## Unit and integration tests

### Prerequisites

* PostgreSQL running
    - Standalone or...
    - In a docker container
    ```sh
    docker run -d -p5432:5432 --name prod_postgres --user postgres postgres
    ```
    Set the connection uri to the Postgres in file: `test/common/application.yml`

Currently unit and integration tests are run with one command.

To execute tests, run the following command:
```sh
make test
```

## Coverage

To generate test coverage report run the following command:
```sh
make coverage
```

The above command will create a file called `coverage.html` at the root of the project.
