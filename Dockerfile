FROM golang:alpine AS builder

WORKDIR /build

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

RUN --mount=type=cache,target=/go/pkg/mod/ \
    go build -o tender ./cmd/tender
RUN --mount=type=cache,target=/go/pkg/mod/ \
    go build -o migrator ./cmd/migrator

FROM alpine AS final

RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add \
        ca-certificates \
        tzdata \
        && \
        update-ca-certificates

WORKDIR /tender

# copy executables
COPY --from=builder /build/tender /tender/tender
COPY --from=builder /build/migrator /tender/migrator

# copy migrations
COPY docs docs
COPY migrations migrations
COPY scripts scripts

EXPOSE 8080

ENTRYPOINT [ "sh", "./scripts/run.sh" ]