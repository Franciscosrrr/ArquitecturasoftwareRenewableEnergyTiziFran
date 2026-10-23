FROM golang:1.26.1-alpine AS source
ENV GOWORK=off
WORKDIR /src
COPY platform ./platform
COPY services ./services
COPY gateway ./gateway
COPY docs/contracts ./docs/contracts
RUN cd platform && go mod download && cd ../services/consumo && go mod download && cd ../electrodomesticos && go mod download && cd ../../gateway && go mod download

FROM source AS test
CMD ["sh", "-c", "set -e; for component in platform gateway services/usuarios services/electrodomesticos services/consumo services/solar; do (cd /src/$component && go test -cover ./...); done"]

FROM source AS build
ARG SERVICE=services/consumo
RUN cd /src/${SERVICE} && CGO_ENABLED=0 go build -trimpath -o /service ./cmd/server
RUN cd /src/platform && CGO_ENABLED=0 go build -trimpath -o /healthcheck ./cmd/healthcheck

FROM scratch
COPY --from=build /service /service
COPY --from=build /healthcheck /healthcheck
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/service"]
