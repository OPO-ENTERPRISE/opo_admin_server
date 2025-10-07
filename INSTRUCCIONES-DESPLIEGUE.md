# 🚀 Guía Rápida de Despliegue - opo-admin-server

## 📝 Pasos Previos (Solo Primera Vez)

### 1. Instalar Google Cloud SDK

**Windows:**
- Descargar desde: https://cloud.google.com/sdk/docs/install
- Ejecutar instalador y seguir instrucciones

**Verificar instalación:**
```bash
gcloud --version
```

### 2. Configurar Google Cloud

```bash
# Autenticarse
gcloud auth login

# Configurar proyecto (reemplaza con tu PROJECT_ID)
gcloud config set project TU_PROJECT_ID

# Configurar región
gcloud config set run/region europe-southwest1
```

### 3. Habilitar APIs necesarias

```bash
# Habilitar todas las APIs requeridas
gcloud services enable run.googleapis.com
gcloud services enable cloudbuild.googleapis.com
gcloud services enable artifactregistry.googleapis.com
```

### 4. Crear Artifact Registry

```bash
# Crear repositorio para imágenes Docker
gcloud artifacts repositories create blog-repository \
    --repository-format=docker \
    --location=europe-southwest1 \
    --description="Repositorio Docker para opo apps"
```

## 🚀 Despliegue Rápido

### Opción 1: Con Cloud Build (Recomendado)

**Antes de desplegar**, edita `cloudbuild.yaml` y actualiza:
- `PROJECT_ID`
- `JWT_SECRET` (genera uno seguro)
- `DB_URL` (tu MongoDB Atlas)
- `CORS_ALLOWED_ORIGINS` (dominios de tu frontend)

```bash
# Desde la carpeta admin/
cd admin

# Ejecutar despliegue
gcloud builds submit --config cloudbuild.yaml
```

### Opción 2: Con Script (Linux/Mac/Git Bash)

```bash
cd admin
chmod +x deploy-admin.sh
./deploy-admin.sh
```

### Opción 3: Con Script (Windows)

```bash
cd admin
deploy-admin.bat
```

## ✅ Verificar Despliegue

### 1. Obtener URL del servicio

```bash
gcloud run services describe opo-admin-server \
    --region=europe-southwest1 \
    --format="value(status.url)"
```

### 2. Probar endpoint de salud

```bash
# Copiar URL del comando anterior y probar
curl https://TU-URL.run.app/api/v1/healthz
```

Debe responder:
```json
{"status":"ok","ts":"..."}
```

## 🔐 Actualizar Variables (Después del Primer Despliegue)

```bash
# Actualizar CORS
gcloud run services update opo-admin-server \
    --region=europe-southwest1 \
    --set-env-vars="CORS_ALLOWED_ORIGINS=https://tu-frontend.com,https://otro-dominio.com"

# Actualizar JWT Secret
gcloud run services update opo-admin-server \
    --region=europe-southwest1 \
    --set-env-vars="JWT_SECRET=nuevo-secret-super-seguro"
```

## 📊 Ver Logs

```bash
# Logs en tiempo real
gcloud run services logs tail opo-admin-server --region=europe-southwest1

# Últimos 50 logs
gcloud run services logs read opo-admin-server --region=europe-southwest1 --limit=50
```

## 🔧 Conectar Frontend

Después del despliegue, obtén la URL y actualiza tu frontend:

```typescript
// adminFront/admin-panel/src/environments/environment.prod.ts
export const environment = {
  production: true,
  apiUrl: 'https://TU-URL.run.app/api/v1'
};
```

## ⚠️ Importante: Seguridad

### Usar Secret Manager para Producción

```bash
# Crear secret para JWT
echo -n "tu-jwt-secret" | gcloud secrets create admin-jwt-secret --data-file=-

# Crear secret para DB
echo -n "mongodb+srv://..." | gcloud secrets create admin-db-url --data-file=-

# Dar permisos
gcloud secrets add-iam-policy-binding admin-jwt-secret \
    --member="serviceAccount:TU_SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor"

# Usar en Cloud Run
gcloud run services update opo-admin-server \
    --region=europe-southwest1 \
    --set-secrets="JWT_SECRET=admin-jwt-secret:latest,DB_URL=admin-db-url:latest"
```

## 💰 Costo Estimado

Con configuración actual (512Mi, escala a 0):
- **Desarrollo/Testing**: < $2/mes
- **Producción (bajo tráfico)**: < $10/mes

## 🆘 Problemas Comunes

### Error: "Container failed to start"
- Verifica logs: `gcloud run services logs read opo-admin-server`
- Asegúrate que PORT=8080

### Error: "Permission denied"
- Verifica permisos IAM de tu cuenta
- Asegúrate que las APIs estén habilitadas

### Error: CORS
- Actualiza CORS_ALLOWED_ORIGINS con tu dominio frontend
- Verifica que uses HTTPS en producción

## 📚 Documentación Completa

Ver `README-DEPLOY-CLOUDRUN.md` para documentación detallada.

