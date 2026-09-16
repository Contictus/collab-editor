# api-go — multi-stage. Distroless has no shell for healthchecks, so the final
# stage is alpine with a nonroot user and a /health probe.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 apigo
COPY --from=build /out/server /server
USER apigo
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health | grep -q '"status":"ok"'
ENTRYPOINT ["/server"]
CMD ["serve-ws"]
