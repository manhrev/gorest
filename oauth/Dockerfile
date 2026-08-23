FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum go.work ./
COPY oauth/go.mod oauth/go.sum oauth/
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app ./oauth/cmd

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /app /app
EXPOSE 8081
ENTRYPOINT ["/app"]
