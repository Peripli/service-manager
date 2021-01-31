# Manage Dependencies
We use `go modules` to manage the project dependencies.

Currently we do not use a vendor directory. If a new branch is pulled, run `go get` to install the dependencies.

* go.mod - defines the dependency requirements, which are the other modules needed for a successful build. Each dependency requirement is written as a module path and a specific semantic version.
* go.sum- containing the expected cryptographic hashes of the content of specific module versions and ensures that this versions will be used on each build

## Selecting the version for a dependency

* Use released versions of a dependency, for example v1.2.3.
* Use the master branch when a dependency does not tag releases, or we require an unreleased change.
* Include an explanatory comment with a link to any relevant issues anytime a dependency is
  pinned to a specific revision in go.mod.

## Add a new dependency

[How to add new dependencies](https://blog.golang.org/using-go-modules) to go.mod as needed.

## Change the version of a dependency

1. Edit go.mod and update the version for the project.
2. Delete go.sum
3. Run `go get to sync go.sum and donwload the dependency version.
