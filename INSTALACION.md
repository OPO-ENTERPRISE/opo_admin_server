# 🛠️ Guía de Instalación - Servidor de Administración OPO

## 📋 Requisitos Previos

- **Go 1.21 o superior**: [Descargar Go](https://golang.org/dl/)
- **MongoDB**: Base de datos MongoDB (local o Atlas)
- **Git**: Para clonar el repositorio

## 🚀 Instalación Rápida

### 1. Configurar Variables de Entorno

```bash
# Copiar archivo de ejemplo
cp env.example .env

# Editar configuración
nano .env  # Linux/Mac
notepad .env  # Windows
```

**Configuración mínima requerida en `.env`:**
```env
# Puerto del servidor (diferente del servidor principal)
PORT=8081

# URL de conexión a MongoDB
DB_URL=mongodb+srv://usuario:password@cluster.mongodb.net/opo?retryWrites=true&w=majority

# Nombre de la base de datos
DB_NAME=opo

# Secret para JWT (genera uno seguro)
JWT_SECRET=admin-jwt-secret-super-seguro-aqui

# Orígenes permitidos para CORS
CORS_ALLOWED_ORIGINS=http://localhost:8100,https://localhost:8100,capacitor://localhost,ionic://localhost
```

### 2. Instalar Dependencias

```bash
go mod tidy
```

### 3. Inicializar Usuario Administrador

```bash
# Linux/Mac
./scripts/init-admin-user.sh admin@example.com password123 1

# Windows
scripts\init-admin-user.bat admin@example.com password123 1

# O manualmente
go run scripts/init-admin-user.go admin@example.com password123 1
```

**Parámetros:**
- `email`: Email del administrador
- `password`: Contraseña (mínimo 6 caracteres)
- `appId`: `1` para PN (Policía Nacional) o `2` para PS (Policía Local/Guardia Civil)

### 4. Iniciar Servidor

```bash
# Linux/Mac
./start-server.sh

# Windows
start-server.bat

# O manualmente
go run cmd/admin/main.go
```

## 🔧 Configuración Detallada

### Variables de Entorno

| Variable | Descripción | Valor por Defecto | Requerido |
|----------|-------------|-------------------|-----------|
| `PORT` | Puerto del servidor | `8081` | No |
| `API_BASE_PATH` | Ruta base de la API | `/api/v1` | No |
| `JWT_SECRET` | Secret para firmar JWT | `admin-secret-key` | No |
| `CORS_ALLOWED_ORIGINS` | Orígenes permitidos para CORS | `http://localhost:8100` | No |
| `DB_URL` | URL de conexión a MongoDB | - | **Sí** |
| `DB_NAME` | Nombre de la base de datos | `opo` | No |

### Configuración de MongoDB

#### MongoDB Atlas (Recomendado)

1. Crear cuenta en [MongoDB Atlas](https://www.mongodb.com/atlas)
2. Crear un cluster
3. Obtener la cadena de conexión
4. Configurar en `DB_URL`

#### MongoDB Local

```env
DB_URL=mongodb://localhost:27017/opo
```

### Configuración de CORS

Para aplicaciones móviles (Ionic/Capacitor):
```env
CORS_ALLOWED_ORIGINS=capacitor://localhost,http://localhost,ionic://localhost,https://localhost
```

Para desarrollo web:
```env
CORS_ALLOWED_ORIGINS=http://localhost:8100,https://tu-dominio.com
```

## 🐳 Instalación con Docker

### 1. Usar Docker Compose

```bash
# Crear archivo .env con la configuración
cp env.example .env

# Editar .env con tu configuración de MongoDB
nano .env

# Iniciar servicios
docker-compose up -d
```

### 2. Construir Imagen Manualmente

```bash
# Construir imagen
docker build -t opo-admin-server .

# Ejecutar contenedor
docker run -p 8081:8081 --env-file .env opo-admin-server
```

## 🔍 Verificación de la Instalación

### 1. Verificar que el Servidor Está Funcionando

```bash
# Health check
curl http://localhost:8081/api/v1/healthz

# Respuesta esperada:
# {"status":"ok","ts":"2024-01-01T00:00:00Z"}
```

### 2. Probar Autenticación

```bash
# Login
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "password123"
  }'

# Respuesta esperada:
# {
#   "user": {
#     "id": "...",
#     "name": "Administrador",
#     "email": "admin@example.com",
#     "appId": "1"
#   },
#   "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
# }
```

### 3. Probar Endpoint Protegido

```bash
# Usar el token obtenido en el login
curl -X GET http://localhost:8081/api/v1/admin/user \
  -H "Authorization: Bearer TU_JWT_TOKEN"

# Respuesta esperada:
# {
#   "_id": "...",
#   "name": "Administrador",
#   "email": "admin@example.com",
#   "appId": "1"
# }
```

## 🚨 Solución de Problemas

### Error: "No se encontró DB_URL"

**Problema:** La variable de entorno `DB_URL` no está configurada.

**Solución:**
1. Verificar que el archivo `.env` existe
2. Verificar que contiene `DB_URL=mongodb://...`
3. Reiniciar el servidor

### Error: "Error conectando a MongoDB"

**Problema:** No se puede conectar a la base de datos.

**Soluciones:**
1. Verificar que la URL de MongoDB es correcta
2. Verificar que MongoDB está ejecutándose
3. Verificar credenciales de acceso
4. Verificar configuración de red/firewall

### Error: "Usuario administrador no encontrado"

**Problema:** No se ha inicializado el usuario administrador.

**Solución:**
```bash
go run scripts/init-admin-user.go admin@example.com password123 1
```

### Error: "Invalid credentials"

**Problema:** Credenciales incorrectas en el login.

**Soluciones:**
1. Verificar email y contraseña
2. Reinicializar usuario administrador si es necesario

### Error: "CORS error"

**Problema:** El frontend no puede acceder a la API.

**Solución:**
1. Verificar `CORS_ALLOWED_ORIGINS` en `.env`
2. Agregar el dominio del frontend a la lista
3. Reiniciar el servidor

## 📚 Próximos Pasos

1. **Configurar Frontend**: Actualizar la URL de la API en el frontend
2. **Crear Topics**: Usar los endpoints de administración para crear topics
3. **Configurar Áreas**: Asegurar que los topics tienen el campo `area` correcto
4. **Probar Filtrado**: Verificar que el filtrado por área funciona correctamente

## 🔗 Enlaces Útiles

- [Documentación de la API](README.md)
- [MongoDB Atlas](https://www.mongodb.com/atlas)
- [Go Documentation](https://golang.org/doc/)
- [Chi Router](https://github.com/go-chi/chi)
- [MongoDB Go Driver](https://docs.mongodb.com/drivers/go/)
