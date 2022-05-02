# Copyright 2018 The Service Manager Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
PATHS = $(PATH):$(PWD)/bin:$(GOBIN):$(HOME)/go/bin:$(TEMPBIN)
all: build test-unit ## Default target that builds SM and runs unit-tests

GO 					?= go
GOFMT 				?= gofmt
BINDIR 				?= bin
PROJECT_PKG 		?= github.com/Peripli/service-manager
TEST_PROFILE_OUT=   cover.out

PLATFORM 			?= linux
ARCH     			?= amd64

INT_TEST_PROFILE 	?= $(CURDIR)/profile-int.cov
UNIT_TEST_PROFILE 	?= $(CURDIR)/profile-unit.cov
INT_BROKER_TEST_PROFILE ?= $(CURDIR)/profile-int-broker.cov
INT_OSB_AND_PLUGIN_TEST_PROFILE ?= $(CURDIR)/profile-int-osb-and-plugin.cov
INT_SERVICE_INSTANCE_AND_BINDINGS_TEST_PROFILE ?= $(CURDIR)/profile-int-service-instance-and-bindings.cov
INT_OTHER_TEST_PROFILE ?= $(CURDIR)/profile-int-other.cov
TEST_PROFILE 		?= $(CURDIR)/profile.cov
COVERAGE 			?= $(CURDIR)/coverage.html

VERSION          	?= $(shell git describe --tags --always --dirty)
DATE             	?= $(shell date -u '+%Y-%m-%d-%H%M UTC')
VERSION_FLAGS    	?= -X "main.Version=$(VERSION)" -X "main.BuildTime=$(DATE)"

# .go files - excludes fakes, mocks, generated files, etc...
SOURCE_FILES	= $(shell find . -type f -name '*.go' ! -name '*.gen.go' ! -name '*.pb.go' ! -name '*mock*.go' \
				! -name '*fake*.go' ! -path "./vendor/*" ! -path "./pkg/query/parser/*"  ! -path "*/*fakes*/*" \
				-exec grep -vrli 'Code generated by counterfeiter' {} \;)

# .go files with go:generate directives (currently files that contain interfaces for which counterfeiter fakes are generated)
GENERATE_PREREQ_FILES = $(shell find . -name "*.go" ! -path "./vendor/*" -exec grep "go:generate" -rli {} \;)

# GO_FLAGS - extra "go build" flags to use - e.g. -v (for verbose)
GO_BUILD 		= env CGO_ENABLED=0 GOOS=$(PLATFORM) GOARCH=$(ARCH) \
           		$(GO) build $(GO_FLAGS) -ldflags '-s -w $(BUILD_LDFLAGS) $(VERSION_FLAGS)'

# TEST_FLAGS - extra "go test" flags to use
GO_INT_TEST 	= $(GOBIN)/gotestsum  --debug --no-color --format standard-verbose -- ./test/... -coverprofile=$(UNIT_TEST_PROFILE)

GO_INT_TEST_OTHER = $(GOBIN)/gotestsum  --debug --no-color --format standard-verbose -- $(shell go list ./test/... | egrep -v "broker_test|osb_and_plugin_test|service_instance_and_binding_test") -coverprofile=$(INT_OTHER_TEST_PROFILE)

GO_INT_TEST_BROKER = $(GOBIN)/gotestsum  --debug --no-color --format standard-verbose -- ./test/broker_test/... -coverprofile=$(INT_BROKER_TEST_PROFILE)

GO_INT_TEST_OSB_AND_PLUGIN = $(GOBIN)/gotestsum  --debug --no-color --format standard-verbose -- ./test/osb_and_plugin_test/... -coverprofile=$(INT_OSB_AND_PLUGIN_TEST_PROFILE)

GO_INT_TEST_SERVICE_INSTANCE_AND_BINDING = $(GOBIN)/gotestsum  --debug --no-color --format standard-verbose -- ./test/service_instance_and_binding_test/... -coverprofile=$(INT_SERVICE_INSTANCE_AND_BINDINGS_TEST_PROFILE)

GO_UNIT_TEST 	= $(GOBIN)/gotestsum  --debug --no-color --format standard-verbose -- $(shell go list ./... | egrep -v "fakes|test|cmd|parser" | paste -sd " " -) \
				$(shell go list ./... | egrep -v "test") -coverprofile=$(UNIT_TEST_PROFILE)

COUNTERFEITER   ?= "v6.0.2"

#-----------------------------------------------------------------------------
# Prepare environment to be able to run other make targets
#-----------------------------------------------------------------------------

prepare-counterfeiter:
	@echo "Installing counterfeiter $(COUNTERFEITER)..."
	@GO111MODULE=off go get -u github.com/maxbrunsfeld/counterfeiter
	@chmod a+x $(GOPATH)/bin/counterfeiter

## Installs some tools (cover, goveralls)
prepare: prepare-counterfeiter build-gen-binary
ifeq ($(shell which cover),)
	@echo "Installing cover tool..."
	@go get -u golang.org/x/tools/cmd/cover
endif
ifeq ($(shell which goveralls),)
	@echo "Installing goveralls..."
	@go get github.com/mattn/goveralls
endif
ifeq ($(shell which golint),)
	@echo "Installing golint... "
	@go get -u golang.org/x/lint/golint
endif

#-----------------------------------------------------------------------------
# Builds and dependency management
#-----------------------------------------------------------------------------

build: .init gomod-vendor service-manager ## Downloads vendored dependecies and builds the service-manager binary

gomod-vendor:
	@go mod vendor

service-manager: $(BINDIR)/service-manager

# Build serivce-manager under ./bin/service-manager
$(BINDIR)/service-manager: FORCE | .init
	 $(GO_BUILD) -o $@ $(PROJECT_PKG)

# init creates the bin dir
.init: $(BINDIR)

# Force can be used as a prerequisite to a target and this will cause this target to always run
FORCE:

$(BINDIR):
	mkdir -p $@

clean-bin: ## Cleans up the binaries
	@echo Deleting $(CURDIR)/$(BINDIR) and built binaries...
	@rm -rf $(BINDIR)


clean-vendor: ## Cleans up the vendor folder and clears out the go.mod
	@echo Deleting vendor folder...
	@rm -rf vendor
	@echo > go.sum

build-gen-binary:
	@go install github.com/Peripli/service-manager/cmd/smgen

#-----------------------------------------------------------------------------
# Tests and coverage
#-----------------------------------------------------------------------------

generate: prepare-counterfeiter build-gen-binary $(GENERATE_PREREQ_FILES) ## Recreates gen files if any of the files containing go:generate directives have changed
	$(GO) list ./... | xargs $(GO) generate
	@touch $@

go-deps:
	go install gotest.tools/gotestsum@latest
	go install github.com/boumenot/gocover-cobertura@latest
	go install github.com/ggere/gototal-cobertura@latest
	go mod vendor

# Run tests
test-unit: go-deps
	@export PATH=$(PATHS)
	@rm -rf $(UNIT_TEST_PROFILE)
	@rm -rf coverage.xml
	@ulimit -n 10000 # Fix too many files open error
	@echo Running unit tests:
	$(GO_UNIT_TEST)
	@echo Total code coverage:
	@go tool cover -func $(UNIT_TEST_PROFILE) | grep total
	PATH=$(PATHS) $(GOBIN)/gocover-cobertura < $(UNIT_TEST_PROFILE) > coverage.xml

test-int: go-deps prepare build
	@export PATH=$(PATHS)
	@rm -rf $(INT_TEST_PROFILE)
	@rm -rf coverage.xml
	@ulimit -n 10000 # Fix too many files open error
	@echo Running integration tests:
#	$(GO_INT_TEST_BROKER)
#	@echo Running integration tests:
#	$(GO_INT_TEST_OSB_AND_PLUGIN)
	@echo Running integration tests:
	$(GO_INT_TEST_SERVICE_INSTANCE_AND_BINDING)
#	@echo Running integration tests:
#	$(GO_INT_TEST_OTHER)

	@echo Total code coverage:
	@go tool cover -func $(INT_TEST_PROFILE) | grep total
	PATH=$(PATHS) $(GOBIN)/gocover-cobertura < $(INT_TEST_PROFILE) > coverage.xml

run-test-all: test-int

# DB
start-db:
	docker-compose -p sm -f contrib/docker-compose.yml up -d
	@echo "Ready!"

stop-db:
	docker-compose -p sm -f contrib/docker-compose.yml down

# Tests

test: stop-db start-db run-test-all stop-db

test-int-other:
	@echo Running integration tests:
	$(GO_INT_TEST_OTHER)

test-int-broker:
	@echo Running integration tests:
	$(GO_INT_TEST_BROKER)

test-int-osb-and-plugin:
	@echo Running integration tests:
	$(GO_INT_TEST_OSB_AND_PLUGIN)

test-int-service-instance-and-binding:
	@echo Running integration tests:
	$(GO_INT_TEST_SERVICE_INSTANCE_AND_BINDING)

test-report: test-int test-unit
	@$(GO) get github.com/wadey/gocovmerge
	@gocovmerge $(CURDIR)/*.cov > $(TEST_PROFILE)


coverage: test-report ## Produces an HTML report containing code coverage details
	@go tool cover -html=$(TEST_PROFILE) -o $(COVERAGE)
	@echo Generated coverage report in $(COVERAGE).


clean-generate:
	@rm -f generate

clean-test-unit: clean-generate ## Cleans up unit test artifacts
	@echo Deleting $(UNIT_TEST_PROFILE)...
	@rm -f $(UNIT_TEST_PROFILE)

clean-test-int: clean-generate ## Cleans up integration test artifacts
	@echo Deleting $(INT_TEST_PROFILE)...
	@rm -f $(INT_TEST_PROFILE)

clean-test-report: clean-test-unit clean-test-int
	@echo Deleting $(TEST_PROFILE)...
	@rm -f $(TEST_PROFILE)

clean-coverage: clean-test-report ## Cleans up coverage artifacts
	@echo Deleting $(COVERAGE)...
	@rm -f $(COVERAGE)

#-----------------------------------------------------------------------------
# Formatting, Linting, Static code checks
#-----------------------------------------------------------------------------
precommit: build coverage format-check lint-check ## Run this before commiting (builds, recreates fakes, runs tests, checks linting and formating). This also runs integration tests - check test-int target for details
precommit-integration-tests-broker: build test-int-broker ## Run this before commiting (builds, recreates fakes, runs tests, checks linting and formating). This also runs integration tests - check test-int target for details
precommit-integration-tests-osb-and-plugin: build test-int-osb-and-plugin ## Run this before commiting (builds, recreates fakes, runs tests, checks linting and formating). This also runs integration tests - check test-int target for details
precommit-integration-tests-service-instance-and-binding: build test-int-service-instance-and-binding ## Run this before commiting (builds, recreates fakes, runs tests, checks linting and formating). This also runs integration tests - check test-int target for details
precommit-integration-tests-other: build test-int-other ## Run this before commiting (builds, recreates fakes, runs tests, checks linting and formating). This also runs integration tests - check test-int target for details
precommit-unit-tests: build test-unit format-check lint-check ## Run this before commiting (builds, recreates fakes, runs tests, checks linting and formating). This also runs integration tests - check test-int target for details
precommit-new-unit-tets: prepare build test-unit format-check lint-check
precommit-new-integration-tests-broker: prepare build  test-int-broker
precommit-new-integration-tests-osb-and-plugin: prepare build test-int-osb-and-plugin
precommit-new-integration-tests-service-instance-and-binding: prepare build test-int-service-instance-and-binding

format: ## Formats the source code files with gofmt
	@echo The following files were reformated:
	@$(GOFMT) -l -s -w $(SOURCE_FILES)

format-check: ## Checks for style violation using gofmt
	@echo Checking if there are files not formatted with gofmt...
	@$(GOFMT) -l -s $(SOURCE_FILES) | grep ".*\.go"; if [ "$$?" = "0" ]; then echo "Files need reformating! Run make format!" ; exit 1; fi

golangci-lint: $(BINDIR)/golangci-lint
$(BINDIR)/golangci-lint:
	@curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.45.2


lint-check: golangci-lint
	@echo Running linter checks...
	@$(BINDIR)/golangci-lint run --config .golangci.yml --issues-exit-code=0 --deadline=30m --out-format checkstyle ./... > checkstyle.xml

#-----------------------------------------------------------------------------
# Useful utility targets
#-----------------------------------------------------------------------------

clean: clean-bin clean-coverage ## Cleans up binaries, test and coverage artifacts

help: ## Displays documentation about the makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
