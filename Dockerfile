#########################################################
# Build the sources and provide the result in a multi stage
# docker container. The alpine build image has to match
# the alpine image in the referencing runtime container.
#########################################################
FROM golang:1.11.2-alpine3.7 AS builder

# We need so that dep can fetch it's dependencies
RUN apk --no-cache add git
RUN go get github.com/golang/dep/cmd/dep

# Directory in workspace
WORKDIR "/go/src/github.com/Peripli/service-manager"

# Copy dep files only and ensure dependencies are satisfied
COPY Gopkg.lock Gopkg.toml ./
RUN dep ensure --vendor-only -v

# Copy and build source code
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags "$(build/ldflags)" -o /main main.go

########################################################
# Build the runtime container
########################################################
FROM alpine:3.7 AS package_step

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the executable file
COPY --from=builder /main /app/

# Copy migration scripts
COPY --from=builder /go/src/github.com/Peripli/service-manager/storage/postgres/migrations/ /app/

# If one wants to use migrations scripts from somewhere else, overriding this env var would override the scripts from the image
ENV STORAGE_MIGRATIONS_URL=file:///app

EXPOSE 8080

ENTRYPOINT [ "./main" ]
