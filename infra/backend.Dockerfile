FROM golang:1.26.1-alpine AS source
ENV GOWORK=off GOMAXPROCS=2 GOMEMLIMIT=512MiB
WORKDIR /src
COPY platform/go.mod platform/go.sum ./platform/
COPY services/consumo/go.mod services/consumo/go.sum ./services/consumo/
COPY services/electrodomesticos/go.mod services/electrodomesticos/go.sum ./services/electrodomesticos/
COPY services/solar/go.mod services/solar/go.sum ./services/solar/
COPY services/usuarios/go.mod services/usuarios/go.sum ./services/usuarios/
COPY gateway/go.mod gateway/go.sum ./gateway/
RUN cd platform && go mod download && cd ../services/consumo && go mod download && cd ../electrodomesticos && go mod download && cd ../solar && go mod download && cd ../usuarios && go mod download && cd ../../gateway && go mod download
COPY platform ./platform
COPY services ./services
COPY gateway ./gateway
COPY docs/contracts ./docs/contracts

FROM source AS test
RUN apk add --no-cache gcc musl-dev
CMD ["sh", "-c", "set -e; for component in platform gateway services/usuarios services/electrodomesticos services/consumo services/solar; do (cd /src/$component && go test -tags=nomsgpack -p 1 -cover ./...); done; for component in platform gateway/internal/balancing services/consumo/internal/integration services/solar/internal/adapters/external; do (cd /src/$component && go test -tags=nomsgpack -p 1 -race ./...); done"]

FROM source AS build
ARG SERVICE=services/consumo
ARG COMMAND=server
RUN --mount=type=cache,target=/root/.cache/go-build cd /src/${SERVICE} && CGO_ENABLED=0 go build -tags=nomsgpack -p 2 -trimpath -o /service ./cmd/${COMMAND}
RUN --mount=type=cache,target=/root/.cache/go-build cd /src/platform && CGO_ENABLED=0 go build -tags=nomsgpack -p 2 -trimpath -o /healthcheck ./cmd/healthcheck

FROM scratch
COPY --from=build /service /service
COPY --from=build /healthcheck /healthcheck
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/service"]
