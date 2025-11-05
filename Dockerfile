# Dockerfile para el servidor de administración
# Usa MongoDB Atlas en la nube - no incluye MongoDB local
# Usamos Debian en lugar de Alpine porque go-fitz requiere glibc
FROM golang:1.23-bullseye AS builder

# Instalar dependencias del sistema necesarias para compilación con CGO
# go-fitz requiere libffi y otras librerías
RUN apt-get update && apt-get install -y \
    git \
    gcc \
    g++ \
    libffi-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Establecer directorio de trabajo
WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar la aplicación con CGO habilitado (necesario para go-fitz)
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o admin-server cmd/admin/main.go

# Imagen final - usar Debian slim para compatibilidad con glibc
FROM debian:bullseye-slim

# Instalar ca-certificates y librerías runtime necesarias para go-fitz
# libc6 (glibc) viene incluido en Debian, solo necesitamos libffi
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libffi8 \
    && rm -rf /var/lib/apt/lists/*

# Crear usuario no-root
RUN useradd -r -s /bin/bash appuser

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
