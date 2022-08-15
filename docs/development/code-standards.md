# Code Standards

The following page describes our expectations in terms of code quality and code standards.

While we do not currently aim to adhere completely to the [Kubernetes coding conventions](https://github.com/kubernetes/community/blob/master/contributors/guide/coding-conventions.md),
we aspire to adhere as closely as possible.

## Code Quality

* Travis CI is configured to run `gometalinter` for static code checks with [the following configuration](https://github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/blob/master/.gometalinter.json). Any errors or warnings would cause a build failure.

* All code must be formated with [gofmt](https://golang.org/cmd/gofmt/).

## GoDoc and Comments

* Any exported symbols (types, interfaces, funcs, structs, etc...) must have Godoc compatible comments associated with them.

* Unexported symbols must be commented sufficiently to provide direction and context to a developer who didn't write the code

* Inline code should be commented sufficiently to explain what complex code is doing. It's up to the developer and reviewers how much and what kind of documentation is necessary

## Tests and Coverage

* Unit and Integration Tests should be written for all changes.
* Code Coverage of changes should be above `85%`
* It's the reviewers' responsibility to ensure that proper/ enough tests have been provided.
