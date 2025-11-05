# Dockerfile para el servidor de administración
# Usa MongoDB Atlas en la nube - no incluye MongoDB local
FROM golang:1.23-alpine AS builder

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
ENV PORT=8080 \
    API_BASE_PATH=/api/v1 \
    CORS_ALLOWED_ORIGINS=https://opo-admin-front-1059081962188.europe-west1.run.app,https://opo-admin-server-chhoc2a3ja-ew.a.run.app,https://localhost,http://localhost,capacitor://localhost,ionic://localhost,https://localhost:8100,http://localhost:8100 \
    JWT_SECRET=dev-secret \
    DB_URL=mongodb+srv://terro:Terro1975%24@cluster0.8s3fkqv.mongodb.net/opo?retryWrites=true&w=majority&appName=Cluster0 \
    DB_NAME=opo

# Exponer puerto
EXPOSE 8080

# Comando por defecto
CMD ["./admin-server"]
