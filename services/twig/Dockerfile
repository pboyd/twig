FROM golang:1.25-alpine AS builder
WORKDIR /app
# Copy the api module (shared dependency) first for layer caching
COPY api/go.mod api/go.sum ./api/
RUN cd api && go mod download
# Copy the server module
COPY services/twig/go.mod services/twig/go.sum ./services/twig/
RUN cd services/twig && GOWORK=off go mod download
# Copy source
COPY api/ ./api/
COPY services/twig/ ./services/twig/
WORKDIR /app/services/twig
ENV GOWORK=off
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /server /server
COPY --from=builder /app/services/twig/db/migrations /db/migrations
EXPOSE 8080
ENTRYPOINT ["/server"]
