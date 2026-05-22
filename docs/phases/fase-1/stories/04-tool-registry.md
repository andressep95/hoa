# Story 04 — Tool Registry + Tools Read-Only

## Como usuario
Quiero que el agente pueda leer archivos, buscar texto y listar archivos de mi proyecto para responder preguntas sobre el código.

## Criterios de Aceptación

- [ ] Interface `Tool` con `Definition()` y `Execute(ctx, input)`
- [ ] `Registry` con `Register()`, `Get()`, `Definitions()`, `Execute()`
- [ ] Auto-registro via `init()` — agregar tool = crear archivo
- [ ] Tool `read_file`: lee archivo por path, soporta offset/limit
- [ ] Tool `grep`: búsqueda regex con ripgrep (o fallback a grep)
- [ ] Tool `glob`: buscar archivos por patrón
- [ ] Tool `bash`: ejecutar comando shell (read-only por ahora, sin gate)
- [ ] El agent loop pasa `tools.Definitions()` al provider
- [ ] El agent loop ejecuta tool calls y agrega results al historial

## Archivos a Crear

```
internal/tool/registry.go    # Tool interface + Registry + Default
internal/tool/readfile.go    # read_file tool
internal/tool/grep.go        # grep tool
internal/tool/glob.go        # glob tool
internal/tool/bash.go        # bash tool
```

## Pattern

```go
// internal/tool/readfile.go
package tool

func init() { Default.Register(&ReadFileTool{}) }

type ReadFileTool struct{}

func (ReadFileTool) Definition() api.ToolDef { ... }
func (ReadFileTool) Execute(ctx context.Context, input string) (string, bool) { ... }
```

## Definición de Done

- `/tools` (o equivalente) lista las 4 tools registradas
- "lee el archivo main.go" → el agente llama read_file y muestra contenido
- "busca TODO en el proyecto" → el agente usa grep
- Tool desconocida → error como tool_result, no crash
