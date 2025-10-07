# Dockerfile para el servidor de administración
# Usa MongoDB Atlas en la nube - no incluye MongoDB local
FROM golang:1.21-alpine AS builder

# Instalar dependencias del sistema
RUN apk add --no-cache git

# Establecer directorio de trabajo
WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o admin-server cmd/admin/main.go

# Imagen final
FROM alpine:latest

# Instalar ca-certificates para HTTPS
RUN apk --no-cache add ca-certificates

# Crear usuario no-root
RUN adduser -D -s /bin/sh appuser

# Establecer directorio de trabajo
WORKDIR /app

# Copiar binario desde builder
COPY --from=builder /app/admin-server .

# Cambiar propietario del archivo
RUN chown appuser:appuser admin-server

# Cambiar a usuario no-root
USER appuser

# Exponer puerto
EXPOSE 8081

# Comando por defecto
CMD ["./admin-server"]
