package http

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	"golang.org/x/crypto/bcrypt"
)

// AdminUserGet - Obtener información del usuario administrador
func AdminUserGet(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		users := client.Database(cfg.DBName).Collection("user")

		// Obtener ID del usuario del contexto
		userID := r.Context().Value("user_id")
		if userID == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "user id not found in token")
			return
		}

		var user domain.User
		if err := users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "usuario administrador no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, user)
	}
}

// AdminUserUpdate - Actualizar información del usuario administrador
func AdminUserUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			AppID string `json:"appId"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		req.Name = strings.TrimSpace(req.Name)

		if req.Name == "" || req.Email == "" || (req.AppID != "1" && req.AppID != "2") {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name, email y appId (1 o 2) requeridos")
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

		users := client.Database(cfg.DBName).Collection("user")

		update := bson.M{
			"$set": bson.M{
				"name":      req.Name,
				"email":     req.Email,
				"appId":     req.AppID,
				"updatedAt": time.Now(),
			},
		}

		var user domain.User
		if err := users.FindOneAndUpdate(ctx, bson.M{}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "usuario administrador no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, user)
	}
}

// AdminUserResetPassword - Cambiar contraseña del usuario administrador
func AdminUserResetPassword(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CurrentPassword string `json:"currentPassword"`
			NewPassword     string `json:"newPassword"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		if req.CurrentPassword == "" || req.NewPassword == "" || len(req.NewPassword) < 6 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "currentPassword y newPassword (>=6) requeridos")
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

		users := client.Database(cfg.DBName).Collection("user")

		var user domain.User
		if err := users.FindOne(ctx, bson.M{}).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "usuario administrador no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)) != nil {
			writeError(w, http.StatusBadRequest, "invalid_password", "contraseña actual incorrecta")
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "no se pudo procesar la nueva contraseña")
			return
		}

		users.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
			"$set": bson.M{
				"password":  string(hash),
				"updatedAt": time.Now(),
			},
		})

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Contraseña actualizada exitosamente",
		})
	}
}

// AdminTopicsList - Listar topics (solo temas principales filtrados por área del usuario)
func AdminTopicsList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parsear parámetros de consulta
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}
		areaParam := r.URL.Query().Get("area")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		// Determinar el área a filtrar
		var filterArea int
		if areaParam != "" {
			// Si viene el parámetro area, usarlo
			filterArea, err = strconv.Atoi(areaParam)
			if err != nil {
				log.Printf("❌ AdminTopicsList - Error convirtiendo área: %v", err)
				writeError(w, http.StatusBadRequest, "invalid_area", "área debe ser un número válido")
				return
			}
			log.Printf("🔍 AdminTopicsList - Parámetro area recibido (string): '%s'", areaParam)
			log.Printf("🔍 AdminTopicsList - Usando área del parámetro (int): %d", filterArea)
		} else {
			// Si no viene area, usar el área del usuario (appId)
			userID := r.Context().Value("user_id")
			if userID == nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "user id not found in token")
				return
			}

			users := client.Database(cfg.DBName).Collection("user")
			var user domain.User
			if err := users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
				writeError(w, http.StatusInternalServerError, "server_error", "error obteniendo usuario: "+err.Error())
				return
			}

			if user.AppID == "1" {
				filterArea = 1
			} else if user.AppID == "2" {
				filterArea = 2
			} else {
				writeError(w, http.StatusForbidden, "forbidden", "usuario sin área asignada válida")
				return
			}
			log.Printf("🔍 AdminTopicsList - Usando área del usuario (appId): %s -> %d", user.AppID, filterArea)
		}

		// Construir filtro: solo temas principales donde id === rootId Y del área especificada
		// Usando $expr para comparar campos dentro del mismo documento
		filter := bson.M{
			"area": filterArea,
			"$expr": bson.M{
				"$eq": []interface{}{"$id", "$rootId"},
			},
		}

		// Agregar filtro de enabled si viene en los parámetros
		enabledParam := r.URL.Query().Get("enabled")
		if enabledParam != "" {
			enabled := enabledParam == "true"
			filter["enabled"] = enabled
			log.Printf("🔍 AdminTopicsList - Aplicando filtro enabled: %v", enabled)
		}

		// Agregar filtro de premium si viene en los parámetros
		premiumParam := r.URL.Query().Get("premium")
		if premiumParam != "" {
			premium := premiumParam == "true"
			filter["premium"] = premium
			log.Printf("🔍 AdminTopicsList - Aplicando filtro premium: %v", premium)
		}

		// Agregar filtro de type si viene en los parámetros
		typeParam := r.URL.Query().Get("type")
		if typeParam != "" {
			// Validar que el type sea válido
			if typeParam == "topic" || typeParam == "exam" || typeParam == "misc" {
				filter["type"] = typeParam
				log.Printf("🔍 AdminTopicsList - Aplicando filtro type: %s", typeParam)
			} else {
				log.Printf("⚠️ AdminTopicsList - Type inválido ignorado: %s", typeParam)
			}
		}

		// Agregar filtro de búsqueda si viene en los parámetros
		searchParam := r.URL.Query().Get("search")
		if searchParam != "" {
			filter["title"] = bson.M{"$regex": searchParam, "$options": "i"}
			log.Printf("🔍 AdminTopicsList - Aplicando filtro search: %s", searchParam)
		}

		log.Printf("🔍 AdminTopicsList - Filtro MongoDB final: %+v", filter)

		// Contar total de temas principales
		total, err := col.CountDocuments(ctx, filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("🔍 AdminTopicsList - Total de topics encontrados con filtro: %d", total)

		// Opciones de paginación con ordenamiento por order
		skip := (page - 1) * limit
		opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "order", Value: 1}})

		// Obtener temas principales
		cur, err := col.Find(ctx, filter, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var topics []domain.Topic
		if err := cur.All(ctx, &topics); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("🔍 AdminTopicsList - Topics recuperados: %d", len(topics))
		if len(topics) > 0 {
			log.Printf("🔍 AdminTopicsList - Primer topic - ID: %d, Área: %d, Premium: %v, Title: %s", topics[0].TopicID, topics[0].Area, topics[0].Premium, topics[0].Title)
		}

		// Calcular páginas totales
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		response := domain.PaginatedResponse{
			Items: topics,
			Pagination: domain.PaginationInfo{
				Page:       page,
				Limit:      limit,
				Total:      int(total),
				TotalPages: totalPages,
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminTopicsGetByID - Obtener topic específico
func AdminTopicsGetByID(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
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

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		var topic domain.Topic
		if err := col.FindOne(ctx, bson.M{"id": id}).Decode(&topic); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, topic)
	}
}

// AdminTopicsGetSubtopics - Obtener subtemas de un tema principal
func AdminTopicsGetSubtopics(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		// Obtener el tema principal
		var parentTopic domain.Topic
		if err := col.FindOne(ctx, bson.M{"id": id}).Decode(&parentTopic); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "tema principal no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Obtener subtemas
		filter := bson.M{
			"rootId": id,
			"id":     bson.M{"$ne": id}, // id !== rootId
		}

		cur, err := col.Find(ctx, filter, options.Find())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var subtopics []domain.Topic
		if err := cur.All(ctx, &subtopics); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		response := map[string]interface{}{
			"subtopics": subtopics,
			"parentTopic": map[string]interface{}{
				"_id":   parentTopic.ID,
				"id":    parentTopic.TopicID,
				"uuid":  parentTopic.UUID,
				"title": parentTopic.Title,
			},
			"total": len(subtopics),
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminTopicsCreate - Crear nuevo topic
func AdminTopicsCreate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.Topic

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validaciones básicas
		if req.TopicID == 0 || req.UUID == "" || req.Title == "" || req.Area == 0 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "id, uuid, title y area requeridos")
			return
		}

		if req.Area != 1 && req.Area != 2 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "area debe ser 1 o 2")
			return
		}

		// Validar tipo si se proporciona, sino establecer valor por defecto
		if req.Type == "" {
			req.Type = "topic" // Valor por defecto
		} else {
			if req.Type != "topic" && req.Type != "exam" && req.Type != "misc" {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "type debe ser 'topic', 'exam' o 'misc'")
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		// Verificar que no existe
		var existing domain.Topic
		if err := col.FindOne(ctx, bson.M{"id": req.TopicID}).Decode(&existing); err == nil {
			writeError(w, http.StatusConflict, "topic_exists", "topic con este ID ya existe")
			return
		}

		// Si es un tema principal, rootId = id
		if req.RootID == 0 {
			req.RootID = req.TopicID
		}
		if req.RootUUID == "" {
			req.RootUUID = req.UUID
		}

		now := time.Now()
		req.ID = uuid.NewString()
		req.CreatedAt = now
		req.UpdatedAt = now

		if _, err := col.InsertOne(ctx, req); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ AdminTopicsCreate - Topic %d creado con type: %s", req.TopicID, req.Type)
		writeJSON(w, http.StatusCreated, req)
	}
}

// AdminTopicsUpdate - Actualizar topic
func AdminTopicsUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
			return
		}

		var req domain.Topic

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validar área si se proporciona
		if req.Area != 0 {
			if req.Area != 1 && req.Area != 2 {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "area debe ser 1 o 2")
				return
			}
		}

		// Validar tipo si se proporciona
		if req.Type != "" {
			if req.Type != "topic" && req.Type != "exam" && req.Type != "misc" {
				writeError(w, http.StatusUnprocessableEntity, "validation_error", "type debe ser 'topic', 'exam' o 'misc'")
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		update := bson.M{
			"$set": bson.M{
				"title":       req.Title,
				"description": req.Description,
				"imageUrl":    req.ImageURL,
				"enabled":     req.Enabled,
				"order":       req.Order,
				"updatedAt":   time.Now(),
			},
		}

		// Agregar área al update si se proporciona
		if req.Area != 0 {
			update["$set"].(bson.M)["area"] = req.Area
			log.Printf("🔄 AdminTopicsUpdate - Actualizando área del topic %d a %d", id, req.Area)
		}

		// Agregar tipo al update si se proporciona
		if req.Type != "" {
			update["$set"].(bson.M)["type"] = req.Type
			log.Printf("🔄 AdminTopicsUpdate - Actualizando type del topic %d a %s", id, req.Type)
		}

		var topic domain.Topic
		if err := col.FindOneAndUpdate(ctx, bson.M{"id": id}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&topic); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Si se cambió el área o el tipo y es un tema principal, actualizar todos los subtopics
		if (req.Area != 0 || req.Type != "") && topic.IsMainTopic() {
			log.Printf("🔍 AdminTopicsUpdate - Es un tema principal, buscando subtopics con rootId=%d", id)

			// Buscar todos los subtopics (donde rootId == id del tema principal y id != rootId)
			subtopicsFilter := bson.M{
				"rootId": id,
				"id":     bson.M{"$ne": id}, // id !== rootId (son subtopics)
			}

			// Contar cuántos subtopics hay
			subtopicsCount, err := col.CountDocuments(ctx, subtopicsFilter)
			if err != nil {
				log.Printf("⚠️ AdminTopicsUpdate - Error contando subtopics: %v", err)
			} else {
				log.Printf("📊 AdminTopicsUpdate - Encontrados %d subtopics para actualizar", subtopicsCount)

				if subtopicsCount > 0 {
					// Preparar actualización de subtopics
					subtopicsUpdateFields := bson.M{
						"updatedAt": time.Now(),
					}

					// Agregar área si se cambió
					if req.Area != 0 {
						subtopicsUpdateFields["area"] = req.Area
					}

					// Agregar tipo si se cambió
					if req.Type != "" {
						subtopicsUpdateFields["type"] = req.Type
					}

					subtopicsUpdate := bson.M{
						"$set": subtopicsUpdateFields,
					}

					updateResult, err := col.UpdateMany(ctx, subtopicsFilter, subtopicsUpdate)
					if err != nil {
						log.Printf("❌ AdminTopicsUpdate - Error actualizando subtopics: %v", err)
						// No devolvemos error porque el topic principal sí se actualizó
					} else {
						if req.Area != 0 && req.Type != "" {
							log.Printf("✅ AdminTopicsUpdate - %d subtopics actualizados (área: %d, type: %s)", updateResult.ModifiedCount, req.Area, req.Type)
						} else if req.Area != 0 {
							log.Printf("✅ AdminTopicsUpdate - %d subtopics actualizados al área %d", updateResult.ModifiedCount, req.Area)
						} else if req.Type != "" {
							log.Printf("✅ AdminTopicsUpdate - %d subtopics actualizados al type %s", updateResult.ModifiedCount, req.Type)
						}
					}
				}
			}
		}

		log.Printf("✅ AdminTopicsUpdate - Topic %d actualizado exitosamente", id)
		writeJSON(w, http.StatusOK, topic)
	}
}

// AdminTopicsToggleEnabled - Cambiar estado enabled/disabled
func AdminTopicsToggleEnabled(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
			return
		}

		var req struct {
			Enabled bool `json:"enabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
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

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		update := bson.M{
			"$set": bson.M{
				"enabled":   req.Enabled,
				"updatedAt": time.Now(),
			},
		}

		result, err := col.UpdateOne(ctx, bson.M{"id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"_id":     idStr,
			"enabled": req.Enabled,
			"message": "Estado del topic actualizado exitosamente",
		})
	}
}

// AdminTopicsTogglePremium - Cambiar estado premium/no-premium
func AdminTopicsTogglePremium(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		log.Printf("🔍 Toggle Premium - ID string recibido: %s", idStr)

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Printf("❌ Toggle Premium - Error al convertir ID: %v", err)
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
			return
		}
		log.Printf("🔍 Toggle Premium - ID convertido a int: %d", id)

		var req struct {
			Premium bool `json:"premium"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("❌ Toggle Premium - Error al decodificar JSON: %v", err)
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}
		log.Printf("🔍 Toggle Premium - Valor premium recibido: %v", req.Premium)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		update := bson.M{
			"$set": bson.M{
				"premium":   req.Premium,
				"updatedAt": time.Now(),
			},
		}

		result, err := col.UpdateOne(ctx, bson.M{"id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"_id":     idStr,
			"premium": req.Premium,
			"message": "Estado premium del topic actualizado exitosamente",
		})
	}
}

// AdminTopicsDelete - Eliminar topic
func AdminTopicsDelete(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		// Convertir id de string a int
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "id debe ser un número")
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

		col := client.Database(cfg.DBName).Collection("topics_uuid_map")

		result, err := col.DeleteOne(ctx, bson.M{"id": id})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.DeletedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "topic no encontrado")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "Topic eliminado exitosamente",
			"deletedId": idStr,
		})
	}
}

// AdminStatsUser - Estadísticas del usuario administrador
func AdminStatsUser(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		// Obtener usuario
		users := client.Database(cfg.DBName).Collection("user")
		var user domain.User
		if err := users.FindOne(ctx, bson.M{}).Decode(&user); err != nil {
			writeError(w, http.StatusNotFound, "not_found", "usuario administrador no encontrado")
			return
		}

		// Obtener estadísticas de topics
		topics := client.Database(cfg.DBName).Collection("topics_uuid_map")

		totalTopics, _ := topics.CountDocuments(ctx, bson.M{})
		enabledTopics, _ := topics.CountDocuments(ctx, bson.M{"enabled": true})
		disabledTopics := totalTopics - enabledTopics

		response := domain.UserStats{
			User: user,
			SystemInfo: domain.SystemInfo{
				TotalTopics:    int(totalTopics),
				EnabledTopics:  int(enabledTopics),
				DisabledTopics: int(disabledTopics),
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminStatsArea - Estadísticas de un área específica
func AdminStatsArea(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaIdStr := chi.URLParam(r, "areaId")

		areaId, err := strconv.Atoi(areaIdStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_area", "área debe ser un número válido")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		// Obtener información del área
		areasCol := client.Database(cfg.DBName).Collection("apps")
		var area domain.App
		if err := areasCol.FindOne(ctx, bson.M{"id": areaIdStr}).Decode(&area); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Estadísticas de topics
		topicsCol := client.Database(cfg.DBName).Collection("topics_uuid_map")

		totalTopics, _ := topicsCol.CountDocuments(ctx, bson.M{"area": areaId})

		// Contar main topics (id === rootId)
		mainTopicsFilter := bson.M{
			"area": areaId,
			"$expr": bson.M{
				"$eq": []interface{}{"$id", "$rootId"},
			},
		}
		mainTopics, _ := topicsCol.CountDocuments(ctx, mainTopicsFilter)

		// Contar subtopics (id !== rootId)
		subtopicsFilter := bson.M{
			"area": areaId,
			"$expr": bson.M{
				"$ne": []interface{}{"$id", "$rootId"},
			},
		}
		subtopics, _ := topicsCol.CountDocuments(ctx, subtopicsFilter)

		// Lógica invertida: enabled:false = Habilitado, enabled:true = Deshabilitado
		enabledTopics, _ := topicsCol.CountDocuments(ctx, bson.M{"area": areaId, "enabled": false})
		disabledTopics := totalTopics - enabledTopics

		// Estadísticas de usuarios
		usersCol := client.Database(cfg.DBName).Collection("users")

		totalUsers, _ := usersCol.CountDocuments(ctx, bson.M{"area": areaId})
		// Lógica invertida: enabled:false = Habilitado, enabled:true = Deshabilitado
		enabledUsers, _ := usersCol.CountDocuments(ctx, bson.M{"area": areaId, "enabled": false})
		disabledUsers := totalUsers - enabledUsers

		response := map[string]interface{}{
			"areaId":   areaId,
			"areaName": area.Name,
			"topics": map[string]interface{}{
				"total":      int(totalTopics),
				"mainTopics": int(mainTopics),
				"subtopics":  int(subtopics),
				"enabled":    int(enabledTopics),
				"disabled":   int(disabledTopics),
			},
			"users": map[string]interface{}{
				"total":    int(totalUsers),
				"enabled":  int(enabledUsers),
				"disabled": int(disabledUsers),
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminStatsAllAreas - Estadísticas de todas las áreas
func AdminStatsAllAreas(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		// Obtener todas las áreas
		areasCol := client.Database(cfg.DBName).Collection("apps")
		cur, err := areasCol.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "order", Value: 1}}))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var areas []domain.App
		if err := cur.All(ctx, &areas); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Obtener estadísticas para cada área
		var allStats []map[string]interface{}

		topicsCol := client.Database(cfg.DBName).Collection("topics_uuid_map")
		usersCol := client.Database(cfg.DBName).Collection("users")

		for _, area := range areas {
			areaId, _ := strconv.Atoi(area.ID)

			// Estadísticas de topics
			totalTopics, _ := topicsCol.CountDocuments(ctx, bson.M{"area": areaId})

			mainTopicsFilter := bson.M{
				"area": areaId,
				"$expr": bson.M{
					"$eq": []interface{}{"$id", "$rootId"},
				},
			}
			mainTopics, _ := topicsCol.CountDocuments(ctx, mainTopicsFilter)

			subtopicsFilter := bson.M{
				"area": areaId,
				"$expr": bson.M{
					"$ne": []interface{}{"$id", "$rootId"},
				},
			}
			subtopics, _ := topicsCol.CountDocuments(ctx, subtopicsFilter)

			// Lógica invertida: enabled:false = Habilitado, enabled:true = Deshabilitado
			enabledTopics, _ := topicsCol.CountDocuments(ctx, bson.M{"area": areaId, "enabled": false})
			disabledTopics := totalTopics - enabledTopics

			// Estadísticas de usuarios
			totalUsers, _ := usersCol.CountDocuments(ctx, bson.M{"area": areaId})
			// Lógica invertida: enabled:false = Habilitado, enabled:true = Deshabilitado
			enabledUsers, _ := usersCol.CountDocuments(ctx, bson.M{"area": areaId, "enabled": false})
			disabledUsers := totalUsers - enabledUsers

			areaStats := map[string]interface{}{
				"areaId":   areaId,
				"areaName": area.Name,
				"topics": map[string]interface{}{
					"total":      int(totalTopics),
					"mainTopics": int(mainTopics),
					"subtopics":  int(subtopics),
					"enabled":    int(enabledTopics),
					"disabled":   int(disabledTopics),
				},
				"users": map[string]interface{}{
					"total":    int(totalUsers),
					"enabled":  int(enabledUsers),
					"disabled": int(disabledUsers),
				},
			}

			allStats = append(allStats, areaStats)
		}

		writeJSON(w, http.StatusOK, allStats)
	}
}

// AdminStatsTopics - Estadísticas de topics
func AdminStatsTopics(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		topics := client.Database(cfg.DBName).Collection("topics_uuid_map")

		totalTopics, _ := topics.CountDocuments(ctx, bson.M{})
		enabledTopics, _ := topics.CountDocuments(ctx, bson.M{"enabled": true})
		disabledTopics := totalTopics - enabledTopics

		// Topics por área
		topicsByArea := make(map[string]int)
		pnTopics, _ := topics.CountDocuments(ctx, bson.M{"area": 1})
		psTopics, _ := topics.CountDocuments(ctx, bson.M{"area": 2})
		topicsByArea["PN"] = int(pnTopics)
		topicsByArea["PS"] = int(psTopics)

		response := domain.TopicStats{
			TotalTopics:    int(totalTopics),
			TopicsByArea:   topicsByArea,
			EnabledTopics:  int(enabledTopics),
			DisabledTopics: int(disabledTopics),
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// ========== Handlers de Áreas (Apps) ==========

// AdminAreasList - Listar áreas con paginación
func AdminAreasList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parsear parámetros de consulta
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("apps")

		// Contar total de áreas
		total, err := col.CountDocuments(ctx, bson.M{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Opciones de paginación con ordenamiento por order
		skip := (page - 1) * limit
		opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "order", Value: 1}})

		// Obtener áreas
		cur, err := col.Find(ctx, bson.M{}, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var areas []domain.App
		if err := cur.All(ctx, &areas); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		// Calcular páginas totales
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		response := domain.PaginatedResponse{
			Items: areas,
			Pagination: domain.PaginationInfo{
				Page:       page,
				Limit:      limit,
				Total:      int(total),
				TotalPages: totalPages,
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminAreasGetByID - Obtener área por ID
func AdminAreasGetByID(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("apps")

		var area domain.App
		// Buscar por el campo id en lugar de _id
		if err := col.FindOne(ctx, bson.M{"id": id}).Decode(&area); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, area)
	}
}

// AdminAreasCreate - Crear nueva área
func AdminAreasCreate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.App

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validaciones básicas
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" || len(req.Name) < 3 || len(req.Name) > 100 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name debe tener entre 3 y 100 caracteres")
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

		col := client.Database(cfg.DBName).Collection("apps")

		// Generar IDs únicos
		now := time.Now()
		req.MongoID = uuid.NewString()

		// Generar ID numérico secuencial
		count, _ := col.CountDocuments(ctx, bson.M{})
		req.ID = strconv.FormatInt(count+1, 10)

		req.CreatedAt = now
		req.UpdatedAt = now

		if _, err := col.InsertOne(ctx, req); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, req)
	}
}

// AdminAreasUpdate - Actualizar área
func AdminAreasUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req domain.App

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		// Validaciones básicas
		req.Name = strings.TrimSpace(req.Name)
		if req.Name != "" && (len(req.Name) < 3 || len(req.Name) > 100) {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name debe tener entre 3 y 100 caracteres")
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

		col := client.Database(cfg.DBName).Collection("apps")

		update := bson.M{
			"$set": bson.M{
				"name":        req.Name,
				"description": req.Description,
				"order":       req.Order,
				"updatedAt":   time.Now(),
			},
		}

		var area domain.App
		// Buscar por el campo id en lugar de _id
		if err := col.FindOneAndUpdate(ctx, bson.M{"id": id}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&area); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, area)
	}
}

// AdminAreasToggleEnabled - Cambiar estado enabled/disabled
func AdminAreasToggleEnabled(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req struct {
			Enabled bool `json:"enabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
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

		col := client.Database(cfg.DBName).Collection("apps")

		update := bson.M{
			"$set": bson.M{
				"enabled":   req.Enabled,
				"updatedAt": time.Now(),
			},
		}

		// Buscar por el campo id en lugar de _id
		result, err := col.UpdateOne(ctx, bson.M{"id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"enabled": req.Enabled,
			"message": "Estado del área actualizado exitosamente",
		})
	}
}

// AdminAreasDelete - Eliminar área
func AdminAreasDelete(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("apps")

		// Buscar por el campo id en lugar de _id
		result, err := col.DeleteOne(ctx, bson.M{"id": id})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.DeletedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "área no encontrada")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":   "Área eliminada exitosamente",
			"deletedId": id,
		})
	}
}

// ========== Handlers de Usuarios ==========

// AdminUsersList - Listar usuarios filtrados por área
func AdminUsersList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parsear parámetros de consulta
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}
		areaParam := r.URL.Query().Get("area")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("users")

		// Determinar el área a filtrar
		var filterArea int
		if areaParam != "" {
			// Si viene el parámetro area, usarlo
			filterArea, err = strconv.Atoi(areaParam)
			if err != nil {
				log.Printf("❌ AdminUsersList - Error convirtiendo área: %v", err)
				writeError(w, http.StatusBadRequest, "invalid_area", "área debe ser un número válido")
				return
			}
			log.Printf("🔍 AdminUsersList - Usando área del parámetro: %d", filterArea)
		} else {
			// Si no viene area, usar el área del admin logueado
			userID := r.Context().Value("user_id")
			if userID == nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "user id not found in token")
				return
			}

			users := client.Database(cfg.DBName).Collection("user")
			var adminUser domain.User
			if err := users.FindOne(ctx, bson.M{"_id": userID}).Decode(&adminUser); err != nil {
				writeError(w, http.StatusInternalServerError, "server_error", "error obteniendo admin: "+err.Error())
				return
			}

			if adminUser.AppID == "1" {
				filterArea = 1
			} else if adminUser.AppID == "2" {
				filterArea = 2
			} else {
				writeError(w, http.StatusForbidden, "forbidden", "admin sin área asignada válida")
				return
			}
			log.Printf("🔍 AdminUsersList - Usando área del admin: %d", filterArea)
		}

		// Construir filtro por área
		filter := bson.M{
			"area": filterArea,
		}

		log.Printf("🔍 AdminUsersList - Filtro MongoDB: %+v", filter)

		// Contar total de usuarios
		total, err := col.CountDocuments(ctx, filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("🔍 AdminUsersList - Total usuarios encontrados: %d", total)

		// Opciones de paginación con ordenamiento por createdAt descendente
		skip := (page - 1) * limit
		opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "createdAt", Value: -1}})

		// Obtener usuarios
		cur, err := col.Find(ctx, filter, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var users []domain.User
		if err := cur.All(ctx, &users); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("🔍 AdminUsersList - Usuarios recuperados: %d", len(users))

		// Calcular páginas totales
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		response := domain.PaginatedResponse{
			Items: users,
			Pagination: domain.PaginationInfo{
				Page:       page,
				Limit:      limit,
				Total:      int(total),
				TotalPages: totalPages,
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminUsersToggleEnabled - Cambiar estado enabled/disabled de un usuario
func AdminUsersToggleEnabled(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req struct {
			Enabled bool `json:"enabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
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

		col := client.Database(cfg.DBName).Collection("users")

		update := bson.M{
			"$set": bson.M{
				"enabled":   req.Enabled,
				"updatedAt": time.Now(),
			},
		}

		// Buscar por el campo _id
		result, err := col.UpdateOne(ctx, bson.M{"_id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "usuario no encontrado")
			return
		}

		log.Printf("✅ AdminUsersToggleEnabled - Usuario %s actualizado a enabled: %v", id, req.Enabled)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"enabled": req.Enabled,
			"message": "Estado del usuario actualizado exitosamente",
		})
	}
}

// ========== Handlers de Proveedores de Publicidad ==========

// AdminProvidersList - Listar proveedores con paginación
func AdminProvidersList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("ad_providers")

		filter := bson.M{}

		// Filtro por enabled si viene en parámetros
		enabledParam := r.URL.Query().Get("enabled")
		if enabledParam != "" {
			enabled := enabledParam == "true"
			filter["enabled"] = enabled
		}

		total, err := col.CountDocuments(ctx, filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		skip := (page - 1) * limit
		opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "order", Value: 1}})

		cur, err := col.Find(ctx, filter, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var providers []domain.AdProvider
		if err := cur.All(ctx, &providers); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		response := domain.PaginatedResponse{
			Items: providers,
			Pagination: domain.PaginationInfo{
				Page:       page,
				Limit:      limit,
				Total:      int(total),
				TotalPages: totalPages,
			},
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// AdminProvidersGetByID - Obtener proveedor por ID
func AdminProvidersGetByID(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("ad_providers")

		var provider domain.AdProvider
		if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&provider); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "proveedor no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, provider)
	}
}

// AdminProvidersCreate - Crear nuevo proveedor
func AdminProvidersCreate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.AdProvider

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.ProviderID = strings.TrimSpace(strings.ToLower(req.ProviderID))

		if req.Name == "" || req.ProviderID == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name y providerId requeridos")
			return
		}

		if len(req.Name) < 3 || len(req.Name) > 100 {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name debe tener entre 3 y 100 caracteres")
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

		col := client.Database(cfg.DBName).Collection("ad_providers")

		// Verificar que no existe un proveedor con el mismo providerId
		var existing domain.AdProvider
		if err := col.FindOne(ctx, bson.M{"providerId": req.ProviderID}).Decode(&existing); err == nil {
			writeError(w, http.StatusConflict, "provider_exists", "ya existe un proveedor con este ID")
			return
		}

		now := time.Now()
		req.ID = uuid.NewString()
		req.CreatedAt = now
		req.UpdatedAt = now

		if _, err := col.InsertOne(ctx, req); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ AdminProvidersCreate - Proveedor %s creado", req.ProviderID)
		writeJSON(w, http.StatusCreated, req)
	}
}

// AdminProvidersUpdate - Actualizar proveedor
func AdminProvidersUpdate(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req domain.AdProvider

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.Name = strings.TrimSpace(req.Name)

		if req.Name != "" && (len(req.Name) < 3 || len(req.Name) > 100) {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "name debe tener entre 3 y 100 caracteres")
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

		col := client.Database(cfg.DBName).Collection("ad_providers")

		update := bson.M{
			"$set": bson.M{
				"name":      req.Name,
				"icon":      req.Icon,
				"color":     req.Color,
				"order":     req.Order,
				"updatedAt": time.Now(),
			},
		}

		var provider domain.AdProvider
		if err := col.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&provider); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "proveedor no encontrado")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		log.Printf("✅ AdminProvidersUpdate - Proveedor %s actualizado", id)
		writeJSON(w, http.StatusOK, provider)
	}
}

// AdminProvidersToggleEnabled - Cambiar estado enabled/disabled
func AdminProvidersToggleEnabled(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req struct {
			Enabled bool `json:"enabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
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

		col := client.Database(cfg.DBName).Collection("ad_providers")

		update := bson.M{
			"$set": bson.M{
				"enabled":   req.Enabled,
				"updatedAt": time.Now(),
			},
		}

		result, err := col.UpdateOne(ctx, bson.M{"_id": id}, update)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.MatchedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "proveedor no encontrado")
			return
		}

		log.Printf("✅ AdminProvidersToggleEnabled - Proveedor %s actualizado a enabled: %v", id, req.Enabled)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"enabled": req.Enabled,
			"message": "Estado del proveedor actualizado exitosamente",
		})
	}
}

// AdminProvidersDelete - Eliminar proveedor
func AdminProvidersDelete(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("ad_providers")

		result, err := col.DeleteOne(ctx, bson.M{"_id": id})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		if result.DeletedCount == 0 {
			writeError(w, http.StatusNotFound, "not_found", "proveedor no encontrado")
			return
		}

		log.Printf("✅ AdminProvidersDelete - Proveedor %s eliminado", id)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Proveedor eliminado exitosamente",
		"deletedId": id,
	})
}

// ========== Handlers de Base de Datos ==========

// AdminDatabaseStats - Obtener estadísticas de la base de datos
func AdminDatabaseStats(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		database := client.Database(cfg.DBName)

		// Obtener lista de colecciones
		collections, err := database.ListCollectionNames(ctx, bson.M{})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "error obteniendo colecciones: "+err.Error())
			return
		}

		var collectionStats []domain.CollectionStats
		var totalDocuments int64
		var totalSize int64

		// Obtener estadísticas de cada colección
		for _, collectionName := range collections {
			collection := database.Collection(collectionName)

			// Contar documentos
			docCount, err := collection.CountDocuments(ctx, bson.M{})
			if err != nil {
				log.Printf("⚠️ Error contando documentos en colección %s: %v", collectionName, err)
				docCount = 0
			}

			// Obtener estadísticas de la colección
			var stats bson.M
			err = database.RunCommand(ctx, bson.M{"collStats": collectionName}).Decode(&stats)
			if err != nil {
				log.Printf("⚠️ Error obteniendo estadísticas de colección %s: %v", collectionName, err)
				stats = bson.M{"size": 0}
			}

			size := int64(0)
			if sizeVal, ok := stats["size"].(int32); ok {
				size = int64(sizeVal)
			} else if sizeVal, ok := stats["size"].(int64); ok {
				size = sizeVal
			}

			collectionStats = append(collectionStats, domain.CollectionStats{
				Name:          collectionName,
				DocumentCount: docCount,
				Size:          size,
			})

			totalDocuments += docCount
			totalSize += size
		}

		response := domain.DatabaseStats{
			DatabaseName:   cfg.DBName,
			TotalSize:      totalSize,
			Collections:    collectionStats,
			TotalDocuments: totalDocuments,
		}

		log.Printf("✅ AdminDatabaseStats - Estadísticas obtenidas: %d colecciones, %d documentos, %d bytes", len(collectionStats), totalDocuments, totalSize)
		writeJSON(w, http.StatusOK, response)
	}
}

// AdminDatabaseDownload - Descargar backup de la base de datos
func AdminDatabaseDownload(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Crear directorio temporal único
		tempDir := fmt.Sprintf("/tmp/mongodb_backup_%d", time.Now().Unix())
		err := os.MkdirAll(tempDir, 0755)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "error creando directorio temporal: "+err.Error())
			return
		}

		// Limpiar directorio temporal al final
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				log.Printf("⚠️ Error limpiando directorio temporal %s: %v", tempDir, err)
			}
		}()

		// Verificar que mongodump esté disponible
		_, err = exec.LookPath("mongodump")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "mongodump no está disponible en el sistema")
			return
		}

		// Construir comando mongodump
		cmd := exec.Command("mongodump", 
			"--uri", cfg.DBURL,
			"--db", cfg.DBName,
			"--out", tempDir,
		)

		// Ejecutar mongodump con timeout
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()

		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("❌ Error ejecutando mongodump: %v, output: %s", err, string(output))
			writeError(w, http.StatusInternalServerError, "server_error", "error ejecutando mongodump: "+err.Error())
			return
		}

		log.Printf("✅ AdminDatabaseDownload - mongodump ejecutado exitosamente")

		// Crear archivo tar.gz
		backupFileName := fmt.Sprintf("mongodb_backup_%s_%d.tar.gz", cfg.DBName, time.Now().Unix())
		backupPath := filepath.Join(tempDir, backupFileName)

		err = createTarGz(backupPath, tempDir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "error creando archivo de backup: "+err.Error())
			return
		}

		// Configurar headers para descarga
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", backupFileName))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", getFileSize(backupPath)))

		// Enviar archivo
		file, err := os.Open(backupPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "error abriendo archivo de backup: "+err.Error())
			return
		}
		defer file.Close()

		_, err = io.Copy(w, file)
		if err != nil {
			log.Printf("❌ Error enviando archivo de backup: %v", err)
			return
		}

		log.Printf("✅ AdminDatabaseDownload - Backup enviado exitosamente: %s", backupFileName)
	}
}

// createTarGz crea un archivo tar.gz con el contenido del directorio
func createTarGz(tarGzPath, sourceDir string) error {
	file, err := os.Create(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Saltar el archivo tar.gz que estamos creando
		if strings.HasSuffix(path, ".tar.gz") {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Ajustar el nombre del archivo en el tar
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}

// getFileSize obtiene el tamaño de un archivo
func getFileSize(filePath string) int64 {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return 0
	}

	return stat.Size()
}
