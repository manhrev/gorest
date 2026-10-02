# Build from repo root: docker build -f balancechange/Dockerfile .
# GOWORK=off: go.work lists every module in the repo; this image only
# carries balancechange; mkit comes from the module proxy.
FROM golang:1.27 AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src
COPY balancechange/go.mod balancechange/go.sum balancechange/
RUN cd balancechange && go mod download
COPY balancechange balancechange
RUN cd balancechange && go build -trimpath -ldflags="-s -w" -o /app ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
# WORKDIR / so mkit/postgres finds "migrations" relative to CWD.
WORKDIR /
COPY --from=build /app /app
COPY balancechange/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/app"]
