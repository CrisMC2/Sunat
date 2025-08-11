# Etapa de compilación
FROM golang:1.21-alpine AS builder

# Instalar Chromium y dependencias necesarias
RUN apk add --no-cache \
    chromium \
    git \
    gcc \
    g++ \
    make \
    libc-dev

WORKDIR /app

# Copiar dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Compilar el binario
RUN go build -o scraper cmd/scraper-completo/main.go

# Etapa final
FROM alpine:latest

# Instalar Chromium y dependencias
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    bash \
    curl \
    ca-certificates \
    libstdc++

# Crear directorio de trabajo
WORKDIR /app
RUN mkdir -p /app/output

# Copiar binario compilado
COPY --from=builder /app/scraper .

# Variable para que rod sepa dónde está chromium
ENV ROD_BROWSER_PATH=/usr/bin/chromium-browser

# Comando por defecto
ENTRYPOINT ["./scraper"]
