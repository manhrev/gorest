# Build from repo root: docker build -f balancechange/Dockerfile .
# GOWORK=off: go.work lists every module in the repo; this image only
# carries balancechange + common, resolved via go.mod's replace directive.
FROM golang:1.27 AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src
COPY common/go.mod common/go.sum common/
COPY balancechange/go.mod balancechange/go.sum balancechange/
RUN cd balancechange && go mod download
COPY common common
COPY balancechange balancechange
RUN cd balancechange && go build -trimpath -ldflags="-s -w" -o /app ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
# WORKDIR / so common/postgres finds "migrations" relative to CWD.
WORKDIR /
COPY --from=build /app /app
COPY balancechange/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/app"]
