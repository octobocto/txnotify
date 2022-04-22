FROM golang:1.15 AS builder

# set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
# We don't want to copy everything, but exclusions are handled by the .dockerignore file
COPY . .

# build the go app
RUN CGO_ENABLED=0 go build -mod readonly -o txnotify .

# Make it executable
RUN chmod a+x txnotify

FROM alpine

RUN apk add bash

COPY --from=builder /app/txnotify .
COPY --from=builder /app/db/migrations db/migrations

# scratch doesn't include SSL certs by default
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT [ "./txnotify" ]