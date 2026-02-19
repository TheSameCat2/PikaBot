FROM golang:1.26-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/palbot ./cmd/palbot

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/palbot /palbot
VOLUME ["/data"]
ENTRYPOINT ["/palbot"]
