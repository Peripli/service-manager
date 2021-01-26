# Manage Dependencies
We use `go mod` to manage the project dependencies.

Currently we do not commit the vendor directory. If a new branch is pulled, delete your local vendor directory and then run `go mod vendor` to install the dependencies.

* go.mod - defines the module’s module path, which is also the import path used for the root directory, and its dependency requirements, which are the other modules needed for a successful build. Each dependency requirement is written as a module path and a specific semantic version.
* go.sum- containing the expected cryptographic hashes of the content of specific module versions

## Selecting the version for a dependency

* Use released versions of a dependency, for example v1.2.3.
* Use the master branch when a dependency does not tag releases, or we require an unreleased change.
* Include an explanatory comment with a link to any relevant issues anytime a dependency is
  pinned to a specific revision in go.sum.

## Add a new dependency

[How to add new dependencies](https://blog.golang.org/using-go-modules) to go.mod as needed.

## Change the version of a dependency

1. Edit go.mod and update the version for the project.
2. Delete go.sum
3. Run `go mod vendor` to sync go.sum and vendor/ with the updated version.
