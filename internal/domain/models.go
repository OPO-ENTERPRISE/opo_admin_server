package domain

import "time"

// User representa un usuario del sistema (admin o usuario de app)
type User struct {
	ID        string    `bson:"_id" json:"id"` // Usar _id como identificador principal
	Name      string    `bson:"name" json:"name"`
	Email     string    `bson:"email" json:"email"`
	Password  string    `bson:"password,omitempty" json:"-"`
	AppID     string    `bson:"appId,omitempty" json:"appId,omitempty"` // Para usuario admin
	Area      int       `bson:"area,omitempty" json:"area,omitempty"`   // Para usuarios de app: 1=PN, 2=PS
	Enabled   bool      `bson:"enabled" json:"enabled"`                 // Para habilitar/deshabilitar usuarios
	LastLogin time.Time `bson:"lastLogin,omitempty" json:"lastLogin,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// App representa una aplicación/área (PN=1, PS=2)
type App struct {
	MongoID     string    `bson:"_id,omitempty" json:"-"`
	ID          string    `bson:"id" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description" json:"description"`
	Enabled     bool      `bson:"enabled" json:"enabled"`
	Order       int       `bson:"order" json:"order"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Topic representa un topic en la colección topics_uuid_map
type Topic struct {
	ID          string    `bson:"_id" json:"_id"`
	TopicID     int       `bson:"id" json:"id"` // Cambiado a int
	UUID        string    `bson:"uuid" json:"uuid"`
	RootID      int       `bson:"rootId" json:"rootId"` // Cambiado a int
	RootUUID    string    `bson:"rootUuid" json:"rootUuid"`
	Area        int       `bson:"area" json:"area"` // Cambiado a int: 1=PN, 2=PS
	Title       string    `bson:"title" json:"title"`
	Description string    `bson:"description,omitempty" json:"description,omitempty"`
	ImageURL    string    `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
	Enabled     bool      `bson:"enabled" json:"enabled"`
	Premium     bool      `bson:"premium" json:"premium"` // Nuevo campo premium
	Type        string    `bson:"type" json:"type"`       // Tipo: "topic", "exam", "misc"
	Order       int       `bson:"order" json:"order"`
	ParentUUID  string    `bson:"parentUuid,omitempty" json:"parentUuid,omitempty"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt" json:"updatedAt"`
}

// TopicResponse representa la respuesta del endpoint público de topics
type TopicResponse struct {
	Title    string          `json:"title"`
	UUID     string          `json:"uuid"`
	RootUUID string          `json:"rootUuid,omitempty"`
	ID       int             `json:"id,omitempty"` // Cambiado a int
	Children []TopicResponse `json:"children,omitempty"`
}

// PaginationInfo representa información de paginación
type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// PaginatedResponse representa una respuesta paginada
type PaginatedResponse struct {
	Items      interface{}    `json:"items"`
	Pagination PaginationInfo `json:"pagination"`
}

// IsMainTopic determina si un topic es principal (id === rootId)
func (t *Topic) IsMainTopic() bool {
	return t.TopicID == t.RootID
}

// IsSubTopic determina si un topic es subtopic (id !== rootId)
func (t *Topic) IsSubTopic() bool {
	return t.TopicID != t.RootID
}

// UserStats representa estadísticas del usuario administrador
type UserStats struct {
	User       User       `json:"user"`
	SystemInfo SystemInfo `json:"systemInfo"`
}

// SystemInfo representa información del sistema
type SystemInfo struct {
	TotalTopics    int `json:"totalTopics"`
	EnabledTopics  int `json:"enabledTopics"`
	DisabledTopics int `json:"disabledTopics"`
}

// TopicStats representa estadísticas de topics
type TopicStats struct {
	TotalTopics    int            `json:"totalTopics"`
	TopicsByArea   map[string]int `json:"topicsByArea"`
	EnabledTopics  int            `json:"enabledTopics"`
	DisabledTopics int            `json:"disabledTopics"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}
