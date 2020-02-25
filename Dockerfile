#########################################################
# Build the sources and provide the result in a multi stage
# docker container. The alpine build image has to match
# the alpine image in the referencing runtime container.
#########################################################
FROM golang:1.12.7-alpine3.10 AS builder

# We need so that dep can fetch it's dependencies
RUN apk --no-cache add git

# Directory in workspace
WORKDIR "/go/src/github.com/Peripli/service-manager"

RUN go mod vendor

# Copy and build source code
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags "$(build/ldflags)" -o /main main.go

########################################################
# Build the runtime container
########################################################
FROM alpine:3.10 AS package_step

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the executable file
COPY --from=builder /main /app/

# Copy migration scripts
COPY --from=builder /go/src/github.com/Peripli/service-manager/storage/postgres/migrations/ /app/
COPY --from=builder /go/src/github.com/Peripli/service-manager/application.yml /app/

# If one wants to use migrations scripts from somewhere else, overriding this env var would override the scripts from the image
ENV STORAGE_MIGRATIONS_URL=file:///app

EXPOSE 8080

ENTRYPOINT [ "./main" ]
