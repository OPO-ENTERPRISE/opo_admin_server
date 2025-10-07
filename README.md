# 🛡️ Servidor de Administración - OPO

Servidor de administración separado para gestionar usuarios y topics de la aplicación de tests de oposiciones.

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Deploy Status](https://img.shields.io/badge/Deploy-Google%20Cloud%20Run-orange.svg)](https://cloud.google.com/run)

> **Repositorio**: [https://github.com/JoseCamposDeveloper/opo_admin_server.git](https://github.com/JoseCamposDeveloper/opo_admin_server.git)

## 🎯 Características

- **Usuario administrador único**: Solo un usuario puede acceder al panel de administración
- **Gestión de topics**: CRUD completo con jerarquía de temas principales y subtemas
- **Filtrado por área**: Topics organizados por área (PN=1, PS=2)
- **Autenticación JWT**: Sistema seguro de autenticación
- **API RESTful**: Endpoints bien documentados y estructurados

## 🚀 Instalación y Configuración

### 1. Configurar variables de entorno

```bash
# Copiar archivo de ejemplo
cp env.example .env

# Editar configuración
nano .env
```

### 2. Instalar dependencias

```bash
go mod tidy
```

### 3. Ejecutar servidor

```bash
# Desarrollo
go run cmd/admin/main.go

# Compilar
go build -o admin-server cmd/admin/main.go

# Ejecutar binario
./admin-server
```

## 📡 Endpoints de la API

### Públicos

- `GET /api/v1/healthz` - Health check
- `POST /api/v1/auth/login` - Autenticación del administrador
- `GET /api/v1/topics/area/{areaId}` - Listar topics por área (para frontend)

### Protegidos (requieren JWT)

#### Gestión del Usuario Administrador
- `GET /api/v1/admin/user` - Obtener información del administrador
- `PUT /api/v1/admin/user` - Actualizar información del administrador
- `POST /api/v1/admin/user/reset-password` - Cambiar contraseña

#### Administración de Topics
- `GET /api/v1/admin/topics` - Listar topics (solo temas principales)
- `GET /api/v1/admin/topics/{id}` - Obtener topic específico
- `GET /api/v1/admin/topics/{id}/subtopics` - Obtener subtemas
- `POST /api/v1/admin/topics` - Crear nuevo topic
- `PUT /api/v1/admin/topics/{id}` - Actualizar topic
- `PATCH /api/v1/admin/topics/{id}/enabled` - Toggle enabled/disabled
- `DELETE /api/v1/admin/topics/{id}` - Eliminar topic

#### Estadísticas
- `GET /api/v1/admin/stats/user` - Estadísticas del administrador
- `GET /api/v1/admin/stats/topics` - Estadísticas de topics

## 🗄️ Estructura de Base de Datos

### Colección: `user` (Usuario Administrador Único)
```json
{
  "_id": "ObjectId",
  "name": "string",
  "email": "string (único)",
  "password": "string (hash bcrypt)",
  "appId": "string (1=PN, 2=PS)",
  "lastLogin": "string (ISO 8601)",
  "createdAt": "string (ISO 8601)",
  "updatedAt": "string (ISO 8601)"
}
```

### Colección: `topics_uuid_map`
```json
{
  "_id": "ObjectId",
  "id": "string",
  "uuid": "string (único)",
  "rootId": "string",
  "rootUuid": "string",
  "area": "string (1=PN, 2=PS)",
  "title": "string",
  "description": "string (opcional)",
  "imageUrl": "string (opcional)",
  "enabled": "boolean",
  "order": "string",
  "parentUuid": "string (opcional)",
  "createdAt": "string (ISO 8601)",
  "updatedAt": "string (ISO 8601)"
}
```

## 🔐 Autenticación

### Login
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "password123"
  }'
```

### Usar token en requests protegidos
```bash
curl -X GET http://localhost:8081/api/v1/admin/user \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 🏗️ Jerarquía de Topics

- **Tema Principal**: `id === rootId`
- **Subtema**: `id !== rootId`

El endpoint `/admin/topics` lista solo temas principales. Los subtemas se obtienen mediante `/admin/topics/{id}/subtopics`.

## 🌐 Filtrado por Área

El frontend selecciona un área (PN=1, PS=2) y se guarda en `localStorage` con la clave `'pn'`. El backend filtra automáticamente los topics según el campo `area` en la colección `topics_uuid_map`.

## 📊 Estadísticas

### Usuario Administrador
- Información del usuario
- Estadísticas del sistema (total topics, habilitados, deshabilitados)

### Topics
- Total de topics
- Topics por área (PN/PS)
- Topics habilitados vs deshabilitados

## 🔧 Desarrollo

### Estructura del proyecto
```
admin/
├── cmd/admin/main.go          # Punto de entrada
├── internal/
│   ├── config/config.go       # Configuración
│   ├── domain/models.go       # Modelos de datos
│   └── http/
│       ├── router.go          # Configuración de rutas
│       ├── handlers.go        # Handlers públicos
│       ├── admin_handlers.go  # Handlers de administración
│       └── middleware.go      # Middlewares
├── go.mod                     # Dependencias
└── env.example               # Variables de entorno
```

### Agregar nuevos endpoints

1. Definir handler en `admin_handlers.go`
2. Agregar ruta en `router.go`
3. Actualizar documentación

## 🚀 Deployment

### Docker
```bash
# Construir imagen
docker build -t opo-admin-server .

# Ejecutar contenedor
docker run -p 8081:8081 --env-file .env opo-admin-server
```

### Cloud Run
```bash
# Desplegar en Google Cloud Run
gcloud run deploy opo-admin-server \
  --source . \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

## 📝 Notas

- El servidor de administración usa el puerto 8081 por defecto
- Comparte la misma base de datos que el servidor principal
- Solo hay un usuario administrador en el sistema
- Los topics se organizan jerárquicamente (temas principales y subtemas)
- El filtrado por área es automático basado en el campo `area`
