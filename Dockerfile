FROM golang:1.25.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum* go.work go.work.sum* ./
COPY pkg ./pkg
COPY contracts ./contracts
COPY services ./services
ARG SERVICE
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -trimpath -o /server ./services/${SERVICE}/cmd/server
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
COPY --from=build /server /server
USER app
EXPOSE 8080 9090
ENTRYPOINT ["/server"]

