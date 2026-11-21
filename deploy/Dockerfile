# NOTARIUS — satu image: Go service + Tesseract OCR (bahasa Indonesia).
# Build di server Linux:  docker build -t notarius:latest -f deploy/Dockerfile .
# Jalan: docker run --env-file /etc/notarius/env -p 8080:8080 -v notarius-storage:/data notarius:latest
# OCR_BINARY default di dalam image: /usr/bin/tesseract (timpa via env bila perlu).

FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/notaryos ./cmd/notaryos

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates tzdata \
      tesseract-ocr tesseract-ocr-ind tesseract-ocr-eng \
    && rm -rf /var/lib/apt/lists/*
ENV TZ=Asia/Jakarta \
    ADDR=:8080 \
    STORAGE_DIR=/data \
    OCR_BINARY=/usr/bin/tesseract
WORKDIR /app
COPY --from=build /out/notaryos /app/notaryos
COPY migrations /app/migrations
COPY web/assets /app/web/assets
RUN mkdir -p /data && useradd -r -s /usr/sbin/nologin notarius && chown -R notarius:notarius /app /data
USER notarius
EXPOSE 8080
ENTRYPOINT ["/app/notaryos"]
