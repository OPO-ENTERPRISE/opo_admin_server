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

// CreateTopicRequest representa los datos para crear un nuevo topic
type CreateTopicRequest struct {
	Title       string `json:"title"`
	Area        int    `json:"area"`
	Type        string `json:"type"`
	Order       int    `json:"order"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
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

// AdProvider representa un proveedor de publicidad
type AdProvider struct {
	ID         string    `bson:"_id" json:"_id"`
	ProviderID string    `bson:"providerId" json:"providerId"` // Slug único: "admob", "facebook"
	Name       string    `bson:"name" json:"name"`             // Nombre visible: "AdMob"
	Icon       string    `bson:"icon,omitempty" json:"icon,omitempty"`
	Color      string    `bson:"color,omitempty" json:"color,omitempty"`
	Enabled    bool      `bson:"enabled" json:"enabled"`
	Order      int       `bson:"order" json:"order"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt" json:"updatedAt"`
}

// DatabaseStats representa estadísticas de la base de datos
type DatabaseStats struct {
	DatabaseName   string            `json:"databaseName"`
	TotalSize      int64             `json:"totalSize"`
	Collections    []CollectionStats `json:"collections"`
	TotalDocuments int64             `json:"totalDocuments"`
}

// CollectionStats representa estadísticas de una colección
type CollectionStats struct {
	Name          string `json:"name"`
	DocumentCount int64  `json:"documentCount"`
	Size          int64  `json:"size"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// SourceTopicInfo para mostrar temas disponibles de otras áreas
type SourceTopicInfo struct {
	TopicID       int    `json:"topicId"`
	UUID          string `json:"uuid"`
	Title         string `json:"title"`
	Area          int    `json:"area"`
	IsMain        bool   `json:"isMain"`
	SubtopicCount int    `json:"subtopicCount"`
	QuestionCount int    `json:"questionCount"`
}

// CopyQuestionsRequest para copiar preguntas desde temas origen
type CopyQuestionsRequest struct {
	SourceTopicUuids []string `json:"sourceTopicUuids"` // UUIDs de temas principales origen
}

// CopyQuestionsResponse respuesta de la operación
type CopyQuestionsResponse struct {
	Message         string `json:"message"`
	QuestionsCopied int    `json:"questionsCopied"`
	TopicsProcessed int    `json:"topicsProcessed"`
}

// QuestionAnswer representa una respuesta de una pregunta
type QuestionAnswer struct {
	ID      int    `bson:"id" json:"id"`
	Text    string `bson:"text" json:"text"`
	Correct bool   `bson:"correct" json:"correct"`
}

// Question representa una pregunta en la colección questions
type Question struct {
	MongoID     string           `bson:"_id,omitempty" json:"_id,omitempty"`
	QuestionID  int              `bson:"questionId" json:"questionId"`
	Question    string           `bson:"question" json:"question"`
	Provider    string           `bson:"provider,omitempty" json:"provider,omitempty"`
	Created     string           `bson:"created,omitempty" json:"created,omitempty"`
	Enabled     bool             `bson:"enabled" json:"enabled"`
	Explanation string           `bson:"explanation,omitempty" json:"explanation,omitempty"`
	Answers     []QuestionAnswer `bson:"answers" json:"answers"`
}

// QuestionUnit representa la relación entre un topic y una pregunta
type QuestionUnit struct {
	MongoID       string `bson:"_id,omitempty" json:"_id,omitempty"`
	TopicID       int    `bson:"topicId" json:"topicId"`
	TopicUuid     string `bson:"topicUuid" json:"topicUuid"`
	RootTopicID   int    `bson:"rootTopicId" json:"rootTopicId"`
	RootTopicUuid string `bson:"rootTopicUuid" json:"rootTopicUuid"`
	Area          int    `bson:"area" json:"area"`
	QuestionID    int    `bson:"questionId" json:"questionId"`
}

// QuestionFromJSON representa la estructura JSON para subir preguntas
type QuestionFromJSON struct {
	Statement string                   `json:"statement"`
	Options   []QuestionOptionFromJSON `json:"options"`
	Multi     bool                     `json:"multi"`
}

// QuestionOptionFromJSON representa una opción desde el JSON
type QuestionOptionFromJSON struct {
	Text    string `json:"text"`
	Correct bool   `json:"correct"`
}
