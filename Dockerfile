# --- Tahap 1: Build Image ---
# Gunakan image Go resmi sebagai base build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Salin modul dan unduh dependensi (memanfaatkan Docker cache)
COPY go.mod go.sum ./
RUN go mod download

# Salin kode sumber
COPY . .

# Bangun aplikasi (CGO_ENABLED=0 menghasilkan binary statis)
RUN CGO_ENABLED=0 GOOS=linux go build -o edc-pinpad-gateway ./cmd/main.go

# --- Tahap 2: Production Image ---
# Gunakan image Alpine yang sangat ringan
FROM alpine:latest

# Instal sertifikat CA (penting untuk koneksi eksternal/HTTPS jika ada)
RUN apk --no-cache add ca-certificates
RUN apk --no-cache add tzdata

WORKDIR /root/

# Salin binary yang sudah terkompilasi dari tahap 'builder'
COPY --from=builder /app/edc-pinpad-gateway .

# Jalankan binary
CMD ["./edc-pinpad-gateway"]