# create a docker file for the sasuke project use alpine golang 1.24
ARG BASE=golang:1.24-alpine3.22
FROM ${BASE} AS builder

ARG ALPINE_PKG_BASE="make git"
ARG ALPINE_PKG_EXTRA=""
ARG ADD_BUILD_TAGS=""

RUN apk add --no-cache ${ALPINE_PKG_BASE} ${ALPINE_PKG_EXTRA}
WORKDIR /app

COPY go.mod vendor* ./
RUN [ ! -d "vendor" ] && go mod download all || echo "skipping..."

COPY . .
ARG MAKE="make build"
RUN $MAKE

# final stage
FROM alpine:3.22
# RUN addgroup -S app && adduser -S -G app app
EXPOSE 50051
RUN apk --no-cache add ca-certificates dumb-init
WORKDIR /home/app
# USER app
LABEL license='MIT license'
LABEL Name=sasuke Version=${VERSION}

# Ensure using latest versions of all installed packages to avoid any recent CVEs
RUN apk --no-cache upgrade

COPY --from=builder /app/LICENSE /LICENSE
COPY --from=builder /app/res/ /res/
COPY --from=builder /app/sasuke /sasuke

CMD ["/sasuke"]