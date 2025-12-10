package http

import (
	"context"
	"encoding/json"
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

type supportSummary struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	UserEmail     string    `json:"userEmail,omitempty"`
	UserID        string    `json:"userId,omitempty"`
	Status        string    `json:"status"`
	LastUpdated   time.Time `json:"lastUpdated"`
	UnreadByAdmin bool      `json:"unreadByAdmin"`
	UnreadByUser  bool      `json:"unreadByUser"`
	MessageCount  int       `json:"messageCount"`
	Area          int       `json:"area,omitempty"`
}

// AdminSupportList devuelve las conversaciones de soporte para el panel
func AdminSupportList(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		client, err := getMongoClient(ctx, cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer client.Disconnect(context.Background())

		col := client.Database(cfg.DBName).Collection("support_conversations")

		q := r.URL.Query()
		status := strings.TrimSpace(q.Get("status"))
		search := strings.TrimSpace(q.Get("search"))
		areaStr := strings.TrimSpace(q.Get("area"))

		filter := bson.M{}
		if status != "" {
			filter["status"] = status
		}
		if areaStr != "" {
			if areaInt, err := strconv.Atoi(areaStr); err == nil {
				filter["area"] = areaInt
			}
		}
		if search != "" {
			filter["$or"] = []bson.M{
				{"title": bson.M{"$regex": search, "$options": "i"}},
				{"userEmail": bson.M{"$regex": search, "$options": "i"}},
			}
		}

		opts := options.Find().
			SetSort(bson.M{"lastUpdated": -1}).
			SetLimit(200)

		cur, err := col.Find(ctx, filter, opts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		defer cur.Close(ctx)

		var conversations []domain.SupportConversation
		if err := cur.All(ctx, &conversations); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		items := make([]supportSummary, 0, len(conversations))
		for _, c := range conversations {
			items = append(items, supportSummary{
				ID:            c.ID,
				Title:         c.Title,
				UserEmail:     c.UserEmail,
				UserID:        c.UserID,
				Status:        c.Status,
				LastUpdated:   c.LastUpdated,
				UnreadByAdmin: c.UnreadByAdmin,
				UnreadByUser:  c.UnreadByUser,
				MessageCount:  len(c.Messages),
				Area:          c.Area,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"items": items,
			"total": len(items),
		})
	}
}

// AdminSupportGetByID devuelve el detalle de una conversación
func AdminSupportGetByID(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		convID := strings.TrimSpace(chi.URLParam(r, "id"))
		if convID == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "conversation id requerido")
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

		col := client.Database(cfg.DBName).Collection("support_conversations")

		var conv domain.SupportConversation
		if err := col.FindOne(ctx, bson.M{"_id": convID}).Decode(&conv); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "conversación no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"conversation": conv,
		})
	}
}

// AdminSupportReply añade una respuesta del admin
func AdminSupportReply(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		convID := strings.TrimSpace(chi.URLParam(r, "id"))
		if convID == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "conversation id requerido")
			return
		}

		adminEmail, _ := r.Context().Value("user_email").(string)
		adminID, _ := r.Context().Value("user_id").(string)

		var req struct {
			Message string `json:"message"`
			Status  string `json:"status"` // opcional: open | closed
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid json")
			return
		}

		req.Message = strings.TrimSpace(req.Message)
		if req.Message == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "el mensaje es obligatorio")
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

		col := client.Database(cfg.DBName).Collection("support_conversations")

		now := time.Now()
		msg := domain.SupportMessage{
			ID:         uuid.NewString(),
			Sender:     "admin",
			SenderID:   adminID,
			SenderName: adminEmail,
			Message:    req.Message,
			CreatedAt:  now,
		}

		setFields := bson.M{
			"lastUpdated":   now,
			"unreadByUser":  true,
			"unreadByAdmin": false,
		}
		if req.Status != "" {
			setFields["status"] = req.Status
		}

		update := bson.M{
			"$push": bson.M{"messages": msg},
			"$set":  setFields,
		}

		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		var conv domain.SupportConversation
		if err := col.FindOneAndUpdate(ctx, bson.M{"_id": convID}, update, opts).Decode(&conv); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "conversación no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"conversation": conv,
			"messageId":    msg.ID,
		})
	}
}

// AdminSupportMarkSeen marca como visto por el admin
func AdminSupportMarkSeen(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		convID := strings.TrimSpace(chi.URLParam(r, "id"))
		if convID == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "conversation id requerido")
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

		col := client.Database(cfg.DBName).Collection("support_conversations")

		update := bson.M{
			"$set": bson.M{
				"unreadByAdmin": false,
			},
		}

		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		var conv domain.SupportConversation
		if err := col.FindOneAndUpdate(ctx, bson.M{"_id": convID}, update, opts).Decode(&conv); err != nil {
			if err == mongo.ErrNoDocuments {
				writeError(w, http.StatusNotFound, "not_found", "conversación no encontrada")
				return
			}
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"conversation": conv,
		})
	}
}

