FROM node:22.16.0-alpine AS web
WORKDIR /web
COPY frontend/package*.json ./
RUN npm ci --ignore-scripts
COPY frontend/ ./
RUN npm run build

FROM golang:1.26.1-alpine AS build
ENV GOWORK=off
WORKDIR /src
COPY platform ./platform
COPY gateway ./gateway
RUN cd gateway && go mod download && CGO_ENABLED=0 go build -trimpath -o /service ./cmd/server
RUN cd platform && CGO_ENABLED=0 go build -trimpath -o /healthcheck ./cmd/healthcheck

FROM scratch
COPY --from=build /service /service
COPY --from=build /healthcheck /healthcheck
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=web /web/dist /web
COPY docs/contracts /contracts
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/service"]
