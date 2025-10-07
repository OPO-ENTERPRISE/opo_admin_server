# 🚀 Despliegue de opo-admin-server en Google Cloud Run

Este documento explica cómo desplegar el servidor de administración en Google Cloud Run.

## 📋 Prerrequisitos

1. **Google Cloud SDK** instalado y configurado
2. **Cuenta de Google Cloud** con facturación habilitada
3. **Proyecto de Google Cloud** creado
4. **MongoDB Atlas** configurado (compartido con opo_server_base)

## ⚙️ Configuración Inicial

### 1. Instalar Google Cloud SDK (si no lo tienes)

**Windows:**
```bash
# Descargar e instalar desde: https://cloud.google.com/sdk/docs/install
```

**Linux/Mac:**
```bash
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
```

### 2. Configurar proyecto

```bash
# Autenticarse en Google Cloud
gcloud auth login

# Configurar proyecto
gcloud config set project TU_PROJECT_ID

# Configurar región por defecto
gcloud config set run/region europe-southwest1
```

### 3. Habilitar APIs necesarias

```bash
# Habilitar Cloud Run API
gcloud services enable run.googleapis.com

# Habilitar Cloud Build API
gcloud services enable cloudbuild.googleapis.com

# Habilitar Artifact Registry API
gcloud services enable artifactregistry.googleapis.com
```

### 4. Crear Artifact Registry (si no existe)

```bash
# Crear repositorio para imágenes Docker
gcloud artifacts repositories create blog-repository \
    --repository-format=docker \
    --location=europe-southwest1 \
    --description="Repositorio de imágenes Docker para opo apps"
```

## 🚀 Despliegue

### Opción 1: Script Automatizado (Recomendado)

**Linux/Mac:**
```bash
# Hacer ejecutable el script
chmod +x deploy-admin.sh

# Ejecutar despliegue
./deploy-admin.sh
```

**Windows:**
```bash
# Ejecutar script de Windows
deploy-admin.bat
```

### Opción 2: Cloud Build Manual

```bash
# Desde la carpeta admin/
gcloud builds submit --config cloudbuild.yaml
```

### Opción 3: Despliegue Directo (sin Cloud Build)

```bash
# Construir imagen localmente
docker build -t europe-southwest1-docker.pkg.dev/TU_PROJECT_ID/blog-repository/opo-admin-server:latest -f Dockerfile.prod .

# Autenticar Docker con Google Cloud
gcloud auth configure-docker europe-southwest1-docker.pkg.dev

# Subir imagen
docker push europe-southwest1-docker.pkg.dev/TU_PROJECT_ID/blog-repository/opo-admin-server:latest

# Desplegar en Cloud Run
gcloud run deploy opo-admin-server \
    --image=europe-southwest1-docker.pkg.dev/TU_PROJECT_ID/blog-repository/opo-admin-server:latest \
    --region=europe-southwest1 \
    --platform=managed \
    --allow-unauthenticated \
    --memory=512Mi \
    --cpu=1
```

## 🔐 Configurar Variables de Entorno

### Método 1: Durante el despliegue

```bash
gcloud run deploy opo-admin-server \
    --image=... \
    --set-env-vars="PORT=8080,API_BASE_PATH=/api/v1,JWT_SECRET=tu-secret,DB_URL=tu-mongo-url,CORS_ALLOWED_ORIGINS=https://tu-admin-frontend.com"
```

### Método 2: Actualizar servicio existente

```bash
gcloud run services update opo-admin-server \
    --region=europe-southwest1 \
    --set-env-vars="CORS_ALLOWED_ORIGINS=https://nuevo-dominio.com,https://otro-dominio.com"
```

### Método 3: Usar Secret Manager (Recomendado para producción)

```bash
# Crear secretos
echo -n "tu-jwt-secret-super-seguro" | gcloud secrets create admin-jwt-secret --data-file=-
echo -n "mongodb+srv://user:pass@cluster.mongodb.net/opo" | gcloud secrets create admin-db-url --data-file=-

# Dar permisos a la cuenta de servicio
gcloud secrets add-iam-policy-binding admin-jwt-secret \
    --member="serviceAccount:blog-255@blog-panel-460312.iam.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor"

gcloud secrets add-iam-policy-binding admin-db-url \
    --member="serviceAccount:blog-255@blog-panel-460312.iam.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor"

# Actualizar Cloud Run para usar secretos
gcloud run services update opo-admin-server \
    --region=europe-southwest1 \
    --set-secrets="JWT_SECRET=admin-jwt-secret:latest,DB_URL=admin-db-url:latest"
```

## 🔍 Verificación del Despliegue

### 1. Obtener URL del servicio

```bash
gcloud run services describe opo-admin-server \
    --region=europe-southwest1 \
    --format="value(status.url)"
```

### 2. Probar endpoint de salud

```bash
# Obtener URL
URL=$(gcloud run services describe opo-admin-server --region=europe-southwest1 --format="value(status.url)")

# Probar health check
curl $URL/api/v1/healthz
```

Respuesta esperada:
```json
{"status":"ok","ts":"2024-01-01T00:00:00Z"}
```

### 3. Ver logs

```bash
# Logs en tiempo real
gcloud run services logs tail opo-admin-server --region=europe-southwest1

# Últimos 50 logs
gcloud run services logs read opo-admin-server --region=europe-southwest1 --limit=50
```

## 📊 Configuración Recomendada

### Recursos por Entorno

**Desarrollo/Testing:**
- Memory: 256Mi
- CPU: 1
- Min instances: 0
- Max instances: 2

**Producción:**
- Memory: 512Mi
- CPU: 1
- Min instances: 1 (para evitar cold starts)
- Max instances: 10

### Variables de Entorno Críticas

```bash
# Obligatorias
PORT=8080                           # Puerto interno de Cloud Run
DB_URL=mongodb+srv://...            # URL de MongoDB Atlas
JWT_SECRET=secret-super-seguro      # Secret para firmar tokens

# Opcionales pero recomendadas
API_BASE_PATH=/api/v1
DB_NAME=opo
CORS_ALLOWED_ORIGINS=https://tu-frontend.com
```

## 🔧 Configuración CORS

Para que el frontend pueda conectarse al admin desplegado:

```bash
# Actualizar CORS con los dominios del frontend
gcloud run services update opo-admin-server \
    --region=europe-southwest1 \
    --set-env-vars="CORS_ALLOWED_ORIGINS=https://tu-admin-frontend.netlify.app,https://tu-admin-frontend.vercel.app"
```

## 📝 Estructura de Archivos de Despliegue

```
admin/
├── Dockerfile.prod         # Dockerfile optimizado para producción
├── cloudbuild.yaml         # Configuración de Cloud Build
├── deploy-admin.sh         # Script de despliegue (Linux/Mac)
├── deploy-admin.bat        # Script de despliegue (Windows)
├── .gcloudignore          # Archivos a ignorar en el build
└── README-DEPLOY-CLOUDRUN.md  # Este archivo
```

## 🛠️ Comandos Útiles

### Ver todos los servicios
```bash
gcloud run services list --region=europe-southwest1
```

### Ver detalles del servicio
```bash
gcloud run services describe opo-admin-server --region=europe-southwest1
```

### Actualizar recursos
```bash
gcloud run services update opo-admin-server \
    --region=europe-southwest1 \
    --memory=1Gi \
    --cpu=2
```

### Eliminar servicio
```bash
gcloud run services delete opo-admin-server --region=europe-southwest1
```

### Ver métricas y tráfico
```bash
# Ver URL pública
gcloud run services describe opo-admin-server \
    --region=europe-southwest1 \
    --format="value(status.url)"

# Ver métricas
gcloud run services describe opo-admin-server \
    --region=europe-southwest1 \
    --format="value(status.traffic)"
```

## 🔒 Seguridad

### 1. Usar Secret Manager para datos sensibles

Nunca hardcodees en el código:
- JWT_SECRET
- DB_URL (con credenciales)
- Cualquier API key

### 2. Configurar autenticación (opcional)

Si quieres que el servicio requiera autenticación:

```bash
gcloud run services update opo-admin-server \
    --region=europe-southwest1 \
    --no-allow-unauthenticated
```

### 3. HTTPS automático

Cloud Run proporciona HTTPS automáticamente con certificados gestionados.

## 💰 Costos Estimados

Cloud Run cobra por:
- **Tiempo de CPU**: $0.00002400/vCPU-segundo
- **Memoria**: $0.00000250/GiB-segundo
- **Requests**: $0.40 por millón

**Estimación para admin (bajo tráfico):**
- ~1,000 requests/día
- ~512Mi memoria
- **Costo mensual**: < $5 USD

## 🆘 Solución de Problemas

### Error: "The user-provided container failed to start"
- Verifica que el binario se compile correctamente
- Revisa logs: `gcloud run services logs read opo-admin-server`
- Asegúrate que el puerto interno sea 8080

### Error: "Error connecting to MongoDB"
- Verifica DB_URL en variables de entorno
- Confirma que MongoDB Atlas permita conexiones desde Cloud Run
- Considera usar VPC Connector para IPs fijas

### Error: "CORS error from frontend"
- Verifica CORS_ALLOWED_ORIGINS incluya tu dominio frontend
- Usa HTTPS en producción
- Revisa que las URLs no tengan espacios extras

## 📞 Siguiente Paso: Conectar Frontend

Después del despliegue, actualiza tu frontend Angular con la URL:

```typescript
// adminFront/admin-panel/src/environments/environment.prod.ts
export const environment = {
  production: true,
  apiUrl: 'https://opo-admin-server-xxx.run.app/api/v1'
};
```

## 🔗 Enlaces Útiles

- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud Build Documentation](https://cloud.google.com/build/docs)
- [Secret Manager](https://cloud.google.com/secret-manager/docs)
- [Artifact Registry](https://cloud.google.com/artifact-registry/docs)

