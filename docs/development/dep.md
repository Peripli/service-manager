# Manage Dependencies
We use [dep](https://golang.github.io/dep) to manage the project dependencies.

Currently we do not commit the vendor directory. If a new branch is pulled, delete your local vendor directory and then run `dep ensure -v --vendor-only` to install the dependencies.

* Gopkg.toml - the dep manifest, this is intended to be hand-edited and contains a set of constraints and other rules for dep to apply when selecting appropriate versions of dependencies.
* Gopkg.lock - the dep lockfile, do not edit because it is a generated file.

If you use [VS Code](https://code.visualstudio.com), we recommend installing the [dep extension](https://marketplace.visualstudio.com/items?itemName=carolynvs.dep).
It provides snippets and improved highlighting that makes it easier to work with dep.

## Selecting the version for a dependency

* Use released versions of a dependency, for example v1.2.3.
* Use the master branch when a dependency does not tag releases, or we require an unreleased change.
* Include an explanatory comment with a link to any relevant issues anytime a dependency is
  pinned to a specific revision in Gopkg.toml.

## Add a new dependency

1. Run `dep ensure -add github.com/example/project/pkg/foo`. This adds a constraint to Gopkg.toml and downloads the dependency to vendor/.
2. Import the package in the code and use it.
3. Run `dep ensure -v` to sync Gopkg.lock and vendor/ with your changes.

## Change the version of a dependency

1. Edit Gopkg.toml and update the version for the project. If the project is not in Gopkg.toml already, add a constraint for it and set the version.
2. Run `dep ensure -v` to sync Gopkg.lock and vendor/ with the updated version.
