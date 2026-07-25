FROM node:20-alpine AS web
WORKDIR /web
COPY services/twig-web/package.json services/twig-web/package-lock.json ./
RUN npm ci
COPY services/twig-web/ ./
RUN npm run build

FROM golang:1.26-alpine AS builder
ARG CLI_VERSION=dev
WORKDIR /app
# Copy the api module (shared dependency) first for layer caching
COPY api/go.mod api/go.sum ./api/
RUN cd api && go mod download
# Copy the server module
COPY services/twig/go.mod services/twig/go.sum ./services/twig/
RUN cd services/twig && GOWORK=off go mod download
# Copy the CLI module for cross-compilation
COPY go.mod go.sum ./
RUN GOWORK=off go mod download
# Copy source
COPY api/ ./api/
COPY services/twig/ ./services/twig/
COPY cmd/ ./cmd/
COPY internal/ ./internal/
# Build the server
WORKDIR /app/services/twig
ENV GOWORK=off
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server
# Cross-compile the CLI binary for linux/amd64
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /cli/twig ./cmd/twig && \
    echo "${CLI_VERSION}" > /cli/version

FROM gcr.io/distroless/static-debian12
COPY --from=builder /server /server
COPY --from=builder /app/services/twig/db/migrations /db/migrations
COPY --from=builder /cli /cli
COPY --from=web /web/dist /web
EXPOSE 8080
ENTRYPOINT ["/server"]
