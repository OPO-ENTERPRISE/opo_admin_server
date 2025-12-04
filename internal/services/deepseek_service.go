package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"opo_admin_server/internal/domain"
)

const (
	deepSeekAPIURL  = "https://api.deepseek.com/v1/chat/completions"
	deepSeekModel   = "deepseek-chat"
	deepSeekTimeout = 90 * time.Second
	maxParagraphs   = 200
)

// DeepSeekClient encapsula las llamadas a la API de DeepSeek
type DeepSeekClient struct {
	apiKey     string
	httpClient *http.Client
}

// DeepSeekParagraphRequest describe la petición de párrafos
type DeepSeekParagraphRequest struct {
	DocumentID  string
	FileName    string
	FileType    string
	Text        string
	Metadata    map[string]string
	Instruction string
}

// DeepSeekParagraphResponse contiene los párrafos procesados
type DeepSeekParagraphResponse struct {
	Paragraphs []domain.DocumentParagraph
	RawContent string
}

// NewDeepSeekClient crea un nuevo cliente
func NewDeepSeekClient(apiKey string) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey: strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: deepSeekTimeout,
		},
	}
}

// GenerateParagraphs envía el texto a DeepSeek para segmentarlo en párrafos
func (c *DeepSeekClient) GenerateParagraphs(ctx context.Context, req DeepSeekParagraphRequest) (*DeepSeekParagraphResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("cliente DeepSeek no inicializado")
	}
	if c.apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY no configurada")
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, fmt.Errorf("texto vacío, no se puede solicitar párrafos")
	}

	payload, err := c.buildRequestPayload(req, text)
	if err != nil {
		return nil, err
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error al serializar request para DeepSeek: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, deepSeekAPIURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("error al crear request para DeepSeek: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error al invocar DeepSeek: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error al leer respuesta de DeepSeek: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("DeepSeek respondió %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp deepSeekChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("error al parsear respuesta de DeepSeek: %w", err)
	}

	content := chatResp.firstMessage()
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("DeepSeek no retornó contenido utilizable")
	}

	paragraphs, err := parseDeepSeekParagraphs(content)
	if err != nil {
		return nil, err
	}

	return &DeepSeekParagraphResponse{
		Paragraphs: paragraphs,
		RawContent: content,
	}, nil
}

func (c *DeepSeekClient) buildRequestPayload(req DeepSeekParagraphRequest, text string) (deepSeekChatRequest, error) {
	metadata := map[string]interface{}{
		"documentId": req.DocumentID,
		"fileName":   req.FileName,
		"fileType":   req.FileType,
	}
	if len(req.Metadata) > 0 {
		metadata["metadata"] = req.Metadata
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return deepSeekChatRequest{}, fmt.Errorf("error al serializar metadata para DeepSeek: %w", err)
	}

	instruction := strings.TrimSpace(req.Instruction)
	if instruction == "" {
		instruction = "Divide el texto proporcionado en párrafos coherentes listos para ser incrustados en una base de datos vectorial. Cada párrafo debe capturar una idea única y tener un máximo aproximado de 600 caracteres. Resume cada párrafo en una oración corta y genera de 2 a 4 etiquetas descriptivas. Devuelve únicamente JSON válido con la forma {\"paragraphs\":[{\"index\":1,\"content\":\"...\",\"summary\":\"...\",\"tags\":[\"tag1\"]}]} sin texto adicional."
	}

	userContent := fmt.Sprintf("Datos estructurados del documento:\n%s\n\nTexto fuente:\n%s", string(metadataJSON), text)

	return deepSeekChatRequest{
		Model: deepSeekModel,
		Messages: []deepSeekChatMessage{
			{
				Role:    "system",
				Content: "Eres un asistente que prepara datos para embeddings. Responde exclusivamente con JSON válido.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("%s\n\n%s", instruction, userContent),
			},
		},
		Temperature: 0.2,
		MaxTokens:   4096,
	}, nil
}

func parseDeepSeekParagraphs(content string) ([]domain.DocumentParagraph, error) {
	var payload struct {
		Paragraphs []domain.DocumentParagraph `json:"paragraphs"`
	}

	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, fmt.Errorf("no se pudo interpretar la respuesta de DeepSeek como JSON: %w", err)
	}

	if len(payload.Paragraphs) == 0 {
		return nil, fmt.Errorf("DeepSeek no generó párrafos")
	}

	if len(payload.Paragraphs) > maxParagraphs {
		payload.Paragraphs = payload.Paragraphs[:maxParagraphs]
	}

	for idx := range payload.Paragraphs {
		if payload.Paragraphs[idx].Index == 0 {
			payload.Paragraphs[idx].Index = idx + 1
		}
		payload.Paragraphs[idx].Content = strings.TrimSpace(payload.Paragraphs[idx].Content)
		payload.Paragraphs[idx].Summary = strings.TrimSpace(payload.Paragraphs[idx].Summary)
		if len(payload.Paragraphs[idx].Tags) > 0 {
			var cleanTags []string
			for _, tag := range payload.Paragraphs[idx].Tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					cleanTags = append(cleanTags, tag)
				}
			}
			payload.Paragraphs[idx].Tags = cleanTags
		}
	}

	return payload.Paragraphs, nil
}

type deepSeekChatRequest struct {
	Model       string                `json:"model"`
	Messages    []deepSeekChatMessage `json:"messages"`
	Temperature float32               `json:"temperature,omitempty"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
}

type deepSeekChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (r deepSeekChatResponse) firstMessage() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}
