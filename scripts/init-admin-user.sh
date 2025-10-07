#!/bin/bash

# Script para inicializar el usuario administrador
# Uso: ./init-admin-user.sh <email> <password> <appId>

if [ $# -ne 3 ]; then
    echo "Uso: $0 <email> <password> <appId>"
    echo "Ejemplo: $0 admin@example.com password123 1"
    echo "appId: 1=PN (Policía Nacional), 2=PS (Policía Local/Guardia Civil)"
    exit 1
fi

EMAIL=$1
PASSWORD=$2
APPID=$3

echo "🔧 Inicializando usuario administrador..."
echo "📧 Email: $EMAIL"
echo "🏢 App ID: $APPID"

# Ejecutar el script Go
go run scripts/init-admin-user.go "$EMAIL" "$PASSWORD" "$APPID"

echo "✅ Usuario administrador inicializado!"
echo "🚀 Ahora puedes ejecutar el servidor con: go run cmd/admin/main.go"
