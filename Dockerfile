FROM golang:1.24.1-alpine as builder

RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o main main.go


FROM alpine

# Add edge/testing repository for FFmpeg
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories
RUN apk update && apk add --no-cache ffmpeg

WORKDIR /app

COPY --from=builder /app/main ./

CMD ["./main"]