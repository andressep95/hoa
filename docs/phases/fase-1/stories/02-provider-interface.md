# Story 02 — Provider Interface + Anthropic

## Como usuario
Quiero poder enviar un mensaje al modelo y recibir una respuesta de texto, usando Anthropic como primer provider.

## Criterios de Aceptación

- [ ] Interface `Provider` definida con `Send()`, `Model()`, `SetModel()`
- [ ] Tipos genéricos en `internal/api/`: Message, Block, ToolDef, Response, Usage
- [ ] `AnthropicProvider` implementa la interface usando `anthropic-sdk-go`
- [ ] El provider lee la API key desde config (o env var como fallback)
- [ ] Acumula usage (input/output tokens) por sesión
- [ ] Errores de API se propagan limpiamente (no panic)

## Archivos a Crear

```
internal/api/types.go              # Message, Block, ToolDef, Response, Usage, StopReason
internal/provider/provider.go      # Interface Provider
internal/provider/anthropic.go     # AnthropicProvider
```

## Interface

```go
type Provider interface {
    Send(ctx context.Context, messages []api.Message, tools []api.ToolDef) (api.Response, error)
    Model() string
    SetModel(name string)
}
```

## Definición de Done

- `go build ./...` compila
- Se puede instanciar `AnthropicProvider` y hacer un `Send()` con un mensaje simple
- La respuesta contiene texto y usage correcto
- Sin API key → error claro, no crash
