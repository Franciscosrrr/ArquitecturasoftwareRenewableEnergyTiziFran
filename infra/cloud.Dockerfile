FROM golang:1.26.1-alpine AS build
ENV GOWORK=off GOMAXPROCS=2 GOMEMLIMIT=512MiB
WORKDIR /src
COPY platform ./platform
COPY gateway ./gateway
COPY services/consumo ./services/consumo
RUN cd services/consumo && go mod download && CGO_ENABLED=0 go build -tags=nomsgpack -p 2 -trimpath -o /consumo ./cmd/server
RUN cd gateway && go mod download && CGO_ENABLED=0 go build -tags=nomsgpack -p 2 -trimpath -o /gateway ./cmd/server

FROM alpine:3.22.1
RUN apk add --no-cache ca-certificates tini
COPY --from=build /consumo /consumo
COPY --from=build /gateway /gateway
COPY docs/contracts /contracts
COPY infra/cloud-entrypoint.sh /cloud-entrypoint.sh
RUN chmod 0555 /cloud-entrypoint.sh
ENV CAPABILITY_ONLY=true CONSUMO_URL=http://127.0.0.1:8081 SOLAR_URLS="" IDENTITY_URL="" DOCS_DIR=/contracts
USER 65532:65532
EXPOSE 10000
ENTRYPOINT ["/sbin/tini","--","/cloud-entrypoint.sh"]
