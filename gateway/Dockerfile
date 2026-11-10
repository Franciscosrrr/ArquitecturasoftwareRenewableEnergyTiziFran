FROM node:22.16.0-alpine AS web
WORKDIR /web
COPY frontend/package*.json ./
RUN npm ci --ignore-scripts
COPY frontend/ ./
RUN npm run build

FROM golang:1.26.1-alpine AS build
ENV GOWORK=off GOMAXPROCS=2 GOMEMLIMIT=512MiB
WORKDIR /src
COPY platform/go.mod platform/go.sum ./platform/
COPY gateway/go.mod gateway/go.sum ./gateway/
RUN cd gateway && go mod download
COPY platform ./platform
COPY gateway ./gateway
RUN --mount=type=cache,target=/root/.cache/go-build cd gateway && CGO_ENABLED=0 go build -tags=nomsgpack -p 2 -trimpath -o /service ./cmd/server
RUN --mount=type=cache,target=/root/.cache/go-build cd platform && CGO_ENABLED=0 go build -tags=nomsgpack -p 2 -trimpath -o /healthcheck ./cmd/healthcheck

FROM scratch
COPY --from=build /service /service
COPY --from=build /healthcheck /healthcheck
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=web /web/dist /web
COPY docs/contracts /contracts
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/service"]
