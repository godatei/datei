FROM golang:1.26.0-alpine3.23 AS builder

RUN apk add --no-cache \
    gcc \
    g++ \
    musl-dev \
    pkgconf \
    tesseract-ocr-dev \
    imagemagick-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags "-s" -o ocrserver .


FROM alpine:3.23 AS runtime

RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    imagemagick \
    ghostscript

COPY --from=builder /app/ocrserver /usr/local/bin/ocrserver

EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/ocrserver"]
