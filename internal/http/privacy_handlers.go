package http

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"opo_admin_server/internal/config"
	"opo_admin_server/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ========== Handlers de Políticas de Privacidad ==========

// AdminPrivacyList - Listar todas las políticas de privacidad
func AdminPrivacyList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("privacy_policies")

		// Obtener todas las políticas ordenadas por área
		opts := options.Find().SetSort(bson.D{{Key: "area", Value: 1}})
		cur, err := col.Find(ctx, bson.M{}, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var policies []domain.PrivacyPolicy
		if err := cur.All(ctx, &policies); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, policies)
	}
}

// AdminPrivacyGetByArea - Obtener política de privacidad por área
func AdminPrivacyGetByArea(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaId := chi.URLParam(r, "areaId")

		areaInt, err := strconv.Atoi(areaId)
		if err != nil || (areaInt != 1 && areaInt != 2) {
			writeError(w, http.StatusBadRequest, "invalid_area", "area debe ser 1 o 2")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("privacy_policies")

		var policy domain.PrivacyPolicy
		if err := col.FindOne(ctx, bson.M{"area": areaInt}).Decode(&policy); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "política de privacidad no encontrada para esta área")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, policy)
	}
}

// AdminPrivacyCreate - Crear nueva política de privacidad
func AdminPrivacyCreate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.CreatePrivacyPolicyRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validaciones
		if req.Area != 1 && req.Area != 2 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "area debe ser 1 (PN) o 2 (PS)")
			return
		}

		req.HTML = strings.TrimSpace(req.HTML)
		if req.HTML == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "html es requerido")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("privacy_policies")

		// Verificar si ya existe una política para esta área
		var existing domain.PrivacyPolicy
		if err := col.FindOne(ctx, bson.M{"area": req.Area}).Decode(&existing); err == nil {
			writeError(w, http.StatusConflict, "already_exists", "ya existe una política de privacidad para esta área")
			return
		} else if err != mongo.ErrNoDocuments {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Crear nueva política
		now := time.Now()
		policy := domain.PrivacyPolicy{
			ID:        uuid.New().String(),
			Area:      req.Area,
			HTML:      req.HTML,
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, err = col.InsertOne(ctx, policy)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ Política de privacidad creada: Área %d (ID: %s)", req.Area, policy.ID)

		writeJSON(w, http.StatusCreated, policy)
	}
}

// AdminPrivacyUpdate - Actualizar política de privacidad por área
func AdminPrivacyUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaId := chi.URLParam(r, "areaId")

		areaInt, err := strconv.Atoi(areaId)
		if err != nil || (areaInt != 1 && areaInt != 2) {
			writeError(w, http.StatusBadRequest, "invalid_area", "area debe ser 1 o 2")
			return
		}

		var req domain.UpdatePrivacyPolicyRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.HTML = strings.TrimSpace(req.HTML)
		if req.HTML == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "html es requerido")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("privacy_policies")

		// Verificar que existe
		var existing domain.PrivacyPolicy
		if err := col.FindOne(ctx, bson.M{"area": areaInt}).Decode(&existing); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "política de privacidad no encontrada para esta área")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Actualizar
		update := bson.M{
			"$set": bson.M{
				"html":      req.HTML,
				"updatedAt": time.Now(),
			},
		}

		result, err := col.UpdateOne(ctx, bson.M{"area": areaInt}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "política de privacidad no encontrada")
			return
		}

		// Obtener la política actualizada
		var updated domain.PrivacyPolicy
		if err := col.FindOne(ctx, bson.M{"area": areaInt}).Decode(&updated); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ Política de privacidad actualizada: Área %d (ID: %s)", areaInt, updated.ID)

		writeJSON(w, http.StatusOK, updated)
	}
}

// AdminPrivacyDelete - Eliminar política de privacidad por área
func AdminPrivacyDelete(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaId := chi.URLParam(r, "areaId")

		areaInt, err := strconv.Atoi(areaId)
		if err != nil || (areaInt != 1 && areaInt != 2) {
			writeError(w, http.StatusBadRequest, "invalid_area", "area debe ser 1 o 2")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("privacy_policies")

		result, err := col.DeleteOne(ctx, bson.M{"area": areaInt})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.DeletedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "política de privacidad no encontrada para esta área")
			return
		}

		log.Printf("✅ Política de privacidad eliminada: Área %d", areaInt)

		w.WriteHeader(http.StatusNoContent)
	}
}

// PrivacyPolicyPublic - Endpoint público para ver política de privacidad
func PrivacyPolicyPublic(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaParam := chi.URLParam(r, "area")
		areaParam = strings.ToLower(areaParam)

		var areaInt int
		if areaParam == "pn" {
			areaInt = 1
		} else if areaParam == "ps" {
			areaInt = 2
		} else {
			renderPrivacyPolicyError(w, "Área inválida. Use 'pn' o 'ps'")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			log.Printf("❌ Error conectando a MongoDB: %v", err)
			renderPrivacyPolicyError(w, "Error interno del servidor")
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("privacy_policies")

		var policy domain.PrivacyPolicy
		if err := col.FindOne(ctx, bson.M{"area": areaInt}).Decode(&policy); err != nil {
			if err == mongo.ErrNoDocuments {
				renderPrivacyPolicyError(w, "Política de privacidad no encontrada para esta área")
				return
			}
			log.Printf("❌ Error buscando política: %v", err)
			renderPrivacyPolicyError(w, "Error interno del servidor")
			return
		}

		// Renderizar HTML directamente
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(policy.HTML))
	}
}

// renderPrivacyPolicyError renderiza una página de error HTML
func renderPrivacyPolicyError(w http.ResponseWriter, message string) {
	tmpl := `<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Error - Política de Privacidad</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background-color: #ffffff;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            max-width: 500px;
            width: 100%;
            padding: 40px;
            text-align: center;
        }
        .icon {
            width: 80px;
            height: 80px;
            margin: 0 auto 30px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 40px;
            background-color: #f8d7da;
            color: #721c24;
        }
        h1 {
            color: #2c3e50;
            margin-bottom: 20px;
            font-size: 28px;
        }
        .message {
            color: #555;
            font-size: 16px;
            line-height: 1.6;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✗</div>
        <h1>Error</h1>
        <p class="message">{{.Message}}</p>
    </div>
</body>
</html>`

	t, err := template.New("error").Parse(tmpl)
	if err != nil {
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)

	data := map[string]interface{}{
		"Message": message,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("❌ Error ejecutando template: %v", err)
		http.Error(w, "Error interno", http.StatusInternalServerError)
	}
}

