FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.21
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
