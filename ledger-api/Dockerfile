FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
USER app
COPY --from=build /out/api /api
# Settings (<NAME>_SECRET, <NAME>_MONGODB_URI, ...) come from the environment at run time.
EXPOSE 8080
ENTRYPOINT ["/api"]
