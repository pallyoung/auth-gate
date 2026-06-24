# Build stage
FROM node:18-alpine AS web-builder
WORKDIR /app
COPY packages/web/package*.json ./
RUN npm ci
COPY packages/web/ ./
RUN npm run build

# Server stage
FROM golang:1.21-alpine AS server-builder
WORKDIR /app
COPY packages/server/go.mod packages/server/go.sum ./
RUN go mod download
COPY packages/server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-gate ./cmd/server

# Final stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=server-builder /app/auth-gate ./
COPY --from=web-builder /app/dist ./dist
COPY packages/server/configs/config.yaml ./

EXPOSE 8080

ENTRYPOINT ["./auth-gate", "start", "-f"]
