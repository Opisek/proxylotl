#
# Build
#
FROM golang:1.26.0-alpine3.23 AS build

WORKDIR /app

COPY src ./src
RUN cd src && go build

#
# Runtime
#
FROM alpine:3.23 AS runtime

WORKDIR /app

RUN mkdir ./scripts
RUN touch config.yml

# Add potentially useful utilities for start-up and shut-down commands
RUN apk update
RUN apk add curl wget bash

# Copy the executable
COPY --from=build /app/src/mginx .

# Non-root
RUN addgroup -S mginx && adduser -S mginx -G mginx
USER mginx

ENTRYPOINT [ "./mginx" ]
