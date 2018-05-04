#########################################################
# Build the sources and provide the result in a multi stage
# docker container. The alpine build image has to match
# the alpine image in the referencing runtime container.
#########################################################
FROM golang:1.10.1-alpine3.7 AS build-env

# Set all env variables needed for go
ENV GOBIN /go/bin
ENV GOPATH /go

# We need so that dep can fetch it's dependencies
RUN apk --no-cache add git

# Directory in workspace
RUN mkdir -p "/go/src/github.com/Peripli/service-manager"
COPY . "/go/src/github.com/Peripli/service-manager"
WORKDIR "/go/src/github.com/Peripli/service-manager"

# Install dep, dependencies and build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go get github.com/golang/dep/cmd/dep && \
    dep ensure -v && \
    go build -o /main cmd/main.go

# RUN go build -o /main cmd/main.go

########################################################
# Build the runtime container
########################################################
FROM alpine:3.7
WORKDIR /app

# Copy the executable file
COPY --from=build-env /main /app/

# Copy the application.yml file to the executable container
COPY --from=build-env /go/src/github.com/Peripli/service-manager/application.yml /app/

# Create folder for migration scripts
RUN mkdir -p "/app/storage/postgres/migrations"

# Copy migration scripts
COPY --from=build-env /go/src/github.com/Peripli/service-manager/storage/postgres/migrations/ /app/storage/postgres/migrations/

ARG SM_RUN_ENV=local
ENV SM_RUN_ENV=${SM_RUN_ENV}

ENTRYPOINT [ "./main" ]
EXPOSE 8080
