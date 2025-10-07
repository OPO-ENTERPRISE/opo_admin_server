#!/bin/bash

# Script de despliegue de opo-admin-server en Google Cloud Run
# Este script usa Cloud Build para automatizar el proceso

set -e  # Salir si hay algún error

echo "=========================================="
echo "  Despliegue de opo-admin-server"
echo "  Google Cloud Run"
echo "=========================================="
echo ""

# Verificar que gcloud esté instalado
if ! command -v gcloud &> /dev/null; then
    echo "❌ Error: gcloud CLI no está instalado"
    echo "📥 Instala desde: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

echo "✅ gcloud CLI encontrado"
echo ""

# Verificar configuración de proyecto
PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
if [ -z "$PROJECT_ID" ]; then
    echo "❌ Error: No hay proyecto de Google Cloud configurado"
    echo "💡 Ejecuta: gcloud config set project TU_PROJECT_ID"
    exit 1
fi

echo "📦 Proyecto actual: $PROJECT_ID"
echo ""

# Confirmar despliegue
echo "⚠️  Este script desplegará opo-admin-server en Cloud Run"
echo "   - Región: europe-southwest1"
echo "   - Memoria: 512Mi"
echo "   - Instancias: 0-5 (autoscaling)"
echo ""
read -p "¿Deseas continuar? (y/n): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Despliegue cancelado"
    exit 1
fi

echo ""
echo "🚀 Iniciando despliegue con Cloud Build..."
echo ""

# Ejecutar Cloud Build
gcloud builds submit --config cloudbuild.yaml

echo ""
echo "=========================================="
echo "  ✅ Despliegue completado"
echo "=========================================="
echo ""
echo "📡 Para ver la URL del servicio:"
echo "   gcloud run services describe opo-admin-server --region=europe-southwest1 --format='value(status.url)'"
echo ""
echo "📊 Para ver los logs:"
echo "   gcloud run services logs read opo-admin-server --region=europe-southwest1 --limit=50"
echo ""
echo "🔧 Para actualizar variables de entorno:"
echo "   gcloud run services update opo-admin-server --region=europe-southwest1 --set-env-vars='KEY=VALUE'"
echo ""

