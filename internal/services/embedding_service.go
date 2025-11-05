package services

import (
	"context"
	"fmt"
	"strings"

	"opo_admin_server/internal/domain"

	"github.com/sashabaranov/go-openai"
)

// ChunkText divide el texto en chunks según la estrategia configurada
func ChunkText(text string, config domain.EmbeddingConfig) ([]string, error) {
	if config.ChunkSize <= 0 {
		config.ChunkSize = 500 // Valor por defecto
	}
	if config.Overlap < 0 {
		config.Overlap = 50 // Valor por defecto
	}
	if config.ChunkingStrategy == "" {
		config.ChunkingStrategy = "characters"
	}

	var chunks []string

	switch config.ChunkingStrategy {
	case "characters":
		chunks = chunkByCharacters(text, config.ChunkSize, config.Overlap)
	case "paragraphs":
		chunks = chunkByParagraphs(text, config.ChunkSize, config.Overlap)
	case "sections":
		chunks = chunkBySections(text, config.ChunkSize, config.Overlap)
	default:
		chunks = chunkByCharacters(text, config.ChunkSize, config.Overlap)
	}

	return chunks, nil
}

// chunkByCharacters divide el texto por caracteres
func chunkByCharacters(text string, chunkSize, overlap int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunk := text[start:end]
		chunks = append(chunks, strings.TrimSpace(chunk))

		if end >= len(text) {
			break
		}

		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// chunkByParagraphs divide el texto por párrafos
func chunkByParagraphs(text string, chunkSize, overlap int) []string {
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var currentChunk strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// Si agregar este párrafo excede el tamaño, guardar chunk actual
		if currentChunk.Len() > 0 && currentChunk.Len()+len(para) > chunkSize {
			chunk := strings.TrimSpace(currentChunk.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			currentChunk.Reset()

			// Aplicar overlap: tomar últimos caracteres del chunk anterior
			if overlap > 0 && len(chunks) > 0 {
				lastChunk := chunks[len(chunks)-1]
				if len(lastChunk) > overlap {
					overlapText := lastChunk[len(lastChunk)-overlap:]
					currentChunk.WriteString(overlapText)
				}
			}
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)
	}

	// Agregar último chunk
	if currentChunk.Len() > 0 {
		chunk := strings.TrimSpace(currentChunk.String())
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// chunkBySections divide el texto por secciones (basado en títulos o líneas en mayúsculas)
func chunkBySections(text string, chunkSize, overlap int) []string {
	lines := strings.Split(text, "\n")
	var sections []string
	var currentSection strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detectar inicio de sección (línea en mayúsculas o que termina en :)
		isSectionHeader := (strings.ToUpper(line) == line && len(line) > 3) || strings.HasSuffix(line, ":")

		if isSectionHeader && currentSection.Len() > 0 {
			sections = append(sections, strings.TrimSpace(currentSection.String()))
			currentSection.Reset()
		}

		if currentSection.Len() > 0 {
			currentSection.WriteString("\n")
		}
		currentSection.WriteString(line)
	}

	if currentSection.Len() > 0 {
		sections = append(sections, strings.TrimSpace(currentSection.String()))
	}

	// Si no se encontraron secciones, usar chunking por caracteres
	if len(sections) == 0 {
		return chunkByCharacters(text, chunkSize, overlap)
	}

	// Combinar secciones en chunks del tamaño adecuado
	var chunks []string
	var currentChunk strings.Builder

	for _, section := range sections {
		if currentChunk.Len() > 0 && currentChunk.Len()+len(section) > chunkSize {
			chunk := strings.TrimSpace(currentChunk.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			currentChunk.Reset()

			// Aplicar overlap
			if overlap > 0 && len(chunks) > 0 {
				lastChunk := chunks[len(chunks)-1]
				if len(lastChunk) > overlap {
					overlapText := lastChunk[len(lastChunk)-overlap:]
					currentChunk.WriteString(overlapText)
				}
			}
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(section)
	}

	if currentChunk.Len() > 0 {
		chunk := strings.TrimSpace(currentChunk.String())
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// GenerateEmbeddings genera embeddings para los chunks de texto
func GenerateEmbeddings(chunks []string, config domain.EmbeddingConfig) ([][]float32, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no hay chunks para generar embeddings")
	}

	switch config.EmbeddingModel {
	case "openai":
		return generateOpenAIEmbeddings(chunks, config.OpenAIAPIKey)
	case "huggingface":
		return generateHuggingFaceEmbeddings(chunks, config.HuggingFaceAPIKey)
	default:
		// Por defecto usar OpenAI
		return generateOpenAIEmbeddings(chunks, config.OpenAIAPIKey)
	}
}

// generateOpenAIEmbeddings genera embeddings usando OpenAI
func generateOpenAIEmbeddings(chunks []string, apiKey string) ([][]float32, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key de OpenAI no configurada")
	}

	client := openai.NewClient(apiKey)
	var embeddings [][]float32
	ctx := context.Background()

	for i, chunk := range chunks {
		// Usar text-embedding-ada-002 como modelo por defecto (más común y económico)
		// O usar text-embedding-3-small/3-large para modelos más nuevos
		req := openai.EmbeddingRequest{
			Input: []string{chunk},
			Model: openai.SmallEmbedding3, // text-embedding-3-small
		}

		resp, err := client.CreateEmbeddings(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("error al generar embedding para chunk %d: %w", i, err)
		}

		if len(resp.Data) == 0 {
			return nil, fmt.Errorf("no se recibió embedding para chunk %d", i)
		}

		// Los embeddings de OpenAI vienen como []float32 directamente
		// El SDK de go-openai devuelve []float32 en Embedding
		embedding := resp.Data[0].Embedding
		embeddings = append(embeddings, embedding)
	}

	return embeddings, nil
}

// generateHuggingFaceEmbeddings genera embeddings usando Hugging Face
// Por ahora retorna error, ya que requiere implementación específica
func generateHuggingFaceEmbeddings(chunks []string, apiKey string) ([][]float32, error) {
	// TODO: Implementar integración con Hugging Face API
	return nil, fmt.Errorf("Hugging Face embeddings no implementado aún")
}

