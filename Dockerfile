FROM golang:1.24.2-alpine3.21 AS go-build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd
COPY internal/ ./internal
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o server ./cmd/server/main.go
RUN chmod +x server

FROM node:23.11.0-alpine3.21 AS npm-install
WORKDIR /build
COPY ./web/package*.json .
RUN npm ci

FROM alpine:3.21.3
WORKDIR /device-sharing
COPY --from=go-build /build/server ./server
COPY --from=npm-install /build/node_modules ./web/node_modules
COPY web/templates ./web/templates
EXPOSE 8080
CMD ["/device-sharing/server"]
