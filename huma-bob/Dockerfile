FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum go.work ./
COPY huma-bob/go.mod huma-bob/go.sum huma-bob/
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app ./huma-bob/cmd

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /app /app
COPY huma-bob/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/app"]
