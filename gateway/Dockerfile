FROM golang:1.26.1-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /service ./cmd/server
FROM scratch
COPY --from=build /service /service
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/service"]
