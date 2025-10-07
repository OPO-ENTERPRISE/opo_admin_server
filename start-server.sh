#!/bin/bash

echo "🚀 Iniciando servidor de administración OPO..."
echo

# Verificar que existe el archivo .env
if [ ! -f .env ]; then
    echo "⚠️  No se encontró archivo .env"
    echo "📝 Copiando archivo de ejemplo..."
    cp env.example .env
    echo
    echo "🔧 Por favor, edita el archivo .env con tu configuración de MongoDB"
    echo "📖 Consulta env.example para ver las variables necesarias"
    echo
    read -p "Presiona Enter para continuar..."
    exit 1
fi

echo "✅ Archivo .env encontrado"
echo

# Verificar que Go está instalado
if ! command -v go &> /dev/null; then
    echo "❌ Go no está instalado o no está en el PATH"
    echo "📥 Descarga Go desde: https://golang.org/dl/"
    exit 1
fi

echo "✅ Go está instalado"
echo

# Instalar dependencias
echo "📦 Instalando dependencias..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ Error instalando dependencias"
    exit 1
fi

echo "✅ Dependencias instaladas"
echo

# Iniciar servidor
echo "🚀 Iniciando servidor en puerto 8081..."
echo "📡 API Base Path: /api/v1"
echo "🌐 URL: http://localhost:8081/api/v1/healthz"
echo
echo "💡 Para detener el servidor, presiona Ctrl+C"
echo

go run cmd/admin/main.go
