# Multi-stage build: frontend bundle -> Go binary -> minimal runtime image.

FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -o /spacedchess ./cmd/spacedchess

FROM alpine:3.21
RUN adduser -D -H app
USER app
COPY --from=api /spacedchess /app/spacedchess
COPY --from=web /src/web/dist /app/web
ENV STATIC_DIR=/app/web ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/spacedchess"]
