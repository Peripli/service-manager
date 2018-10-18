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

all: build test

BINDIR ?= bin
TEST_PROFILE ?= $(CURDIR)/profile.cov
COVERAGE ?= $(CURDIR)/coverage.html
PROJECT_PKG = github.com/Peripli/service-manager/cmd/service-manager

PLATFORM ?= linux
ARCH     ?= amd64

BUILD_LDFLAGS =

#  GOFLAGS -  extra "go build" flags to use - e.g. -v   (for verbose)
GO_BUILD = env CGO_ENABLED=0 GOOS=$(PLATFORM) GOARCH=$(ARCH) \
           go build $(GOFLAGS) -ldflags '-s -w $(BUILD_LDFLAGS)'

build: .init dep-vendor service-manager

dep-check:
	@which dep 2>/dev/null || (echo dep is required to build the project; exit 1)

dep: dep-check
	@dep ensure -v

dep-vendor: dep-check
	@dep ensure --vendor-only -v

dep-reload: dep-check clean-vendor dep

service-manager: $(BINDIR)/service-manager

# Build serivce-manager under ./bin/service-manager
$(BINDIR)/service-manager: .init .
	 $(GO_BUILD) -o $@ $(PROJECT_PKG)

# init creates the bin dir
.init: $(BINDIR)

$(BINDIR):
	mkdir -p $@

test: build
	@echo Running tests:
	@go test ./... -p 1 -coverpkg $(shell go list ./... | egrep -v "fakes|test" | paste -sd "," -) -coverprofile=$(TEST_PROFILE)

coverage: build test
	@go tool cover -html=$(TEST_PROFILE) -o "$(COVERAGE)"

clean: clean-bin clean-test clean-coverage

clean-bin:
	rm -rf $(BINDIR)

clean-test:
	rm -f $(TEST_PROFILE)

clean-coverage:
	rm -f $(COVERAGE)

clean-vendor:
	rm -rf vendor
	@echo > Gopkg.lock