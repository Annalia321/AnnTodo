# Build from the repository root:
#   docker build -f docker/Dockerfile -t anntodo .
# Run:
#   docker run -d -p 8080:8080 -v anntodo-data:/app/data --name anntodo anntodo

# ---- Frontend ----
FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---- Backend ----
FROM golang:1.27-alpine AS server
ARG VERSION=dev
ARG GOPROXY=https://proxy.golang.org,direct
ENV CGO_ENABLED=0 GOPROXY=${GOPROXY} GOTOOLCHAIN=local
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY web/embed.go web/
COPY --from=web /src/web/dist web/dist
RUN go build -trimpath -ldflags "-s -w -X anntodo/pkg/version.Version=${VERSION}" -o /out/anntodo ./cmd/anntodo

# ---- Runtime ----
FROM alpine:3
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S anntodo && adduser -S -G anntodo -h /app anntodo
WORKDIR /app
COPY --from=server /out/anntodo /app/anntodo
COPY conf/anntodo.ini /app/conf/anntodo.ini
RUN mkdir -p /app/data && chown -R anntodo:anntodo /app/data
USER anntodo
ENV ANNTODO_SERVER_HOST=0.0.0.0 ANNTODO_SERVER_PORT=8080
EXPOSE 8080
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8080/api/v1/system/status >/dev/null || exit 1
ENTRYPOINT ["/app/anntodo"]
CMD ["run"]
