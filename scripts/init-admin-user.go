package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"opo_admin_server/internal/config"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Cargar configuración
	cfg := config.Load()

	// Conectar a MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.DBURL))
	if err != nil {
		log.Fatalf("Error conectando a MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	// Obtener argumentos de línea de comandos
	if len(os.Args) < 4 {
		fmt.Println("Uso: go run init-admin-user.go <email> <password> <appId>")
		fmt.Println("Ejemplo: go run init-admin-user.go admin@example.com password123 1")
		fmt.Println("appId: 1=PN (Policía Nacional), 2=PS (Policía Local/Guardia Civil)")
		os.Exit(1)
	}

	email := os.Args[1]
	password := os.Args[2]
	appId := os.Args[3]

	if appId != "1" && appId != "2" {
		log.Fatal("appId debe ser 1 (PN) o 2 (PS)")
	}

	// Hash de la contraseña
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Error generando hash de contraseña: %v", err)
	}

	// Crear usuario administrador
	now := time.Now()
	user := bson.M{
		"_id":       uuid.NewString(),
		"name":      "Administrador",
		"email":     email,
		"password":  string(hash),
		"appId":     appId,
		"lastLogin": now,
		"createdAt": now,
		"updatedAt": now,
	}

	// Insertar en la colección user
	collection := client.Database(cfg.DBName).Collection("user")

	// Eliminar usuario existente si existe
	_, err = collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Printf("Advertencia: Error eliminando usuario existente: %v", err)
	}

	// Insertar nuevo usuario
	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		log.Fatalf("Error insertando usuario administrador: %v", err)
	}

	fmt.Printf("✅ Usuario administrador creado exitosamente!\n")
	fmt.Printf("📧 Email: %s\n", email)
	fmt.Printf("🏢 App ID: %s (%s)\n", appId, map[string]string{"1": "PN (Policía Nacional)", "2": "PS (Policía Local/Guardia Civil)"}[appId])
	fmt.Printf("🆔 ID: %s\n", result.InsertedID)
	fmt.Printf("\n🚀 Ahora puedes iniciar sesión en el servidor de administración con estas credenciales.\n")
}
