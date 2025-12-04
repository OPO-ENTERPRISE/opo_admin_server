# IA Works – Flujo de subida y procesado

Este documento resume los pasos necesarios para subir documentos desde el panel de administración, segmentarlos con DeepSeek y dejarlos listos para su indexación vectorial.

## 1. Variables de entorno

- `DEEPSEEK_API_KEY`: clave privada del workspace de DeepSeek. Debe configurarse en cada entorno junto con `PINECONE_API_KEY`.
- `API_BASE_PATH`, `PORT`, `DB_URL`, etc. permanecen igual que en el backend anterior.

> ⚠️ La API de DeepSeek rechaza claves con espacios. Usa `sk-...` o el prefijo que corresponda a la cuenta.

## 2. Petición `POST /api/v1/admin/ia-works/upload`

Solicitud multipart:

| Campo            | Tipo     | Descripción                                                                                 |
|------------------|----------|---------------------------------------------------------------------------------------------|
| `file`           | archivo  | PDF, DOCX o TXT (≤100 MB).                                                                   |
| `metadata`       | texto    | (Opcional) JSON con metadatos del documento. Se mergea con el resto de campos del formulario. |
| `topic`, `area`, `tags`, … | texto | (Opcional) otros campos libres. Se almacenan tal cual en MongoDB.                      |

Ejemplo de `metadata`:

```json
{
  "topicUuid": "abc-123",
  "area": "PN",
  "language": "es",
  "source": "boe-2024"
}
```

El backend normaliza la clave (quita espacios) y guarda todo en `Document.Metadata`.

## 3. Segmentación vía DeepSeek

1. Tras convertir el archivo a texto plano se construye una instrucción estándar para DeepSeek (`deepseek-chat`).
2. El texto completo + metadatos se envía a `https://api.deepseek.com/v1/chat/completions`.
3. DeepSeek debe responder **exclusivamente** con JSON válido:

```json
{
  "paragraphs": [
    {
      "index": 1,
      "content": "…",
      "summary": "…",
      "tags": ["procedimiento", "derecho"]
    }
  ]
}
```

El backend valida el JSON, trimea espacios y limita a 200 párrafos. El payload crudo se guarda en `Document.ParagraphsRaw` para auditoría.

## 4. Persistencia en MongoDB

Se guarda un documento con:

- `Text`: texto completo.
- `Paragraphs`: arreglo limpio para embeddings.
- `ParagraphsRaw`: JSON que devolvió DeepSeek.
- `Metadata`: todos los campos del formulario (incluido `metadata` una vez parseado).

## 5. Procesado a vectores (`POST /api/v1/admin/ia-works/process`)

1. Se recupera el documento por `documentId`.
2. Si hay `Paragraphs`, se usan directamente como segmentos (modo `deepseek_paragraphs`).
3. Si no, se aplica `ChunkText` como fallback (`chunking_fallback`).
4. Cada vector incluye metadata adicional (`sourceParagraphIndex`, `paragraphSummary`, `paragraphTags`).
5. Los vectores se envían a Pinecone con las credenciales configuradas.

## 6. Checklist para el frontend

- Confirmar que se envían los campos requeridos en el multipart (`file` + `metadata` JSON).
- Validar tamaño y formato antes de lanzar la petición para evitar errores 4xx.
- Mostrar al usuario el conteo de párrafos devueltos (`paragraphs.length`) para verificar segmentación.
- Reintentar cuando DeepSeek responda 502/503 (el backend delega ese error como `deepseek_error`).

Con estos pasos el pipeline queda listo para ingerir documentos y generar embeddings consistentes. Cualquier modificación al prompt o al formato del JSON debe actualizarse en este archivo y en `DeepSeekClient`.

