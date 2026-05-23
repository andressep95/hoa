# Fase 1 — MVP: CLI Multi-Provider + Harness de Escritura Verificada

[← INDEX](../INDEX.md)

---

## Ecuación Fundamental

```
Agente = Modelo + Harness
```

El modelo es intercambiable. El harness es lo que impone calidad determinista sobre la inteligencia probabilística. HOA ES el harness.

---

## Alcance

Un CLI funcional que:
1. Se configura en primer uso (provider + modelo base + modelo de planeamiento)
2. Permite cambiar de provider/modelo en runtime via CLI
3. Implementa SDD (Spec-Driven Development) como flujo obligatorio
4. Ejecuta bucles write-then-verify en cada paso de implementación
5. Conecta a BD vectorial externa (ThinkStation)
6. Implementa commit inteligente con amnesia post-commit
7. Valida a nivel de harness (no confía en el modelo)

---

## Bootstrap del Proyecto

```bash
mkdir hoa && cd hoa
go mod init github.com/cloudcentinel/hoa
```

### Dependencias (go.mod)

```go
module github.com/cloudcentinel/hoa

go 1.23

require (
    // SDKs de providers (API key directa)
    github.com/anthropics/anthropic-sdk-go v1.2.0
    github.com/openai/openai-go v0.1.0

    // TUI
    github.com/charmbracelet/bubbletea v1.2.4
    github.com/charmbracelet/lipgloss v1.0.0

    // Oracle (memoria vectorial)
    github.com/godror/godror v0.44.8

    // Concurrencia
    golang.org/x/sync v0.10.0
)
```

> Todo lo dinámico (provider, API keys, modelos, DB) se persiste en `~/.hoa/config.json`.
> Se lee al arranque con `config.Load()`. No hay framework, no hay inyección de dependencias.

---

## Flujo de Arranque

```
$ hoa
     │
     ▼
┌─────────────────────────────────────────┐
│ ¿Existe ~/.hoa/config.json?      │
└────────────┬───────────────┬────────────┘
             │               │
          NO ▼            SÍ ▼
┌────────────────────┐  ┌──────────────────────────────┐
│ WIZARD PRIMER USO  │  │ config.Load()                │
│                    │  │ Construir:                    │
│ 1. Provider        │  │   - Provider (base)          │
│ 2. API Key         │  │   - Provider (planning)      │
│ 3. Modelo base     │  │   - OracleStore (memory)     │
│ 4. Modelo planning │  │   - ToolRegistry             │
│ 5. DB URL/user/pwd │  │   - Harness (verify + SDD)   │
│                    │  │   - Agent (loop principal)    │
│ → config.Save()   │  │                              │
│ → Continuar ──────►│  │                              │
└────────────────────┘  └──────────────────────────────┘
                                    │
                                    ▼
                        ┌───────────────────────┐
                        │   SESIÓN NORMAL        │
                        │   (Agent Loop activo)  │
                        └───────────────────────┘
```

```go
func main() {
    cfg, err := config.Load()
    if err != nil {
        cfg = config.RunWizard()
        cfg.Save()
    }

    llm := provider.New(cfg.Provider, cfg.Models)
    mem := memory.NewOracleStore(cfg.Database)
    tools := tool.NewRegistry()
    harness := harness.New(cfg.Harness)
    agent := agent.New(llm, systemPrompt, tools)
    agent.Compactor = compact.NoCompaction{}

    program := ui.NewProgram(runner, usageFunc)
    program.Run()
}
```

Las sesiones subsiguientes arrancan directo: `config.Load()` → construir structs → sesión. Startup <100ms.

---

## Configuración Multi-Provider

### Primer Uso

```
$ hoa

🔥 HOA — Primera configuración

1. Provider principal:
   [1] Anthropic (Claude)
   [2] OpenAI (GPT-4o / o3)
   [3] Ollama (local)
   [4] AWS Bedrock
   [5] Google (Gemini)

2. API Key: sk-ant-••••••••••••

3. Modelo base (ejecución): claude-sonnet-4-20250514
4. Modelo de planeamiento: claude-opus-4-20250414

5. Base de datos vectorial:
   [1] Conectar a remota (URL)
   [2] Docker local (auto-provision)

✅ Configuración guardada en ~/.hoa/config.json
```

### Cambio en Runtime (dentro de la sesión)

```
> /provider

🔌 Provider activo: anthropic

    Anthropic (Claude)    ← activo
  ▸ OpenAI (GPT)
    Ollama (local)
    Google (Gemini)

  ↑↓ navegar · enter seleccionar · esc cancelar

✅ Provider cambiado a OpenAI
```

```
> /model

🧠 Modelos activos:
   Base:       claude-sonnet-4-20250514
   Planning:   claude-opus-4-20250414

  ▸ Base (ejecución)
    Planning (razonamiento)

  ↑↓ navegar · enter seleccionar

Modelos disponibles (anthropic):

    claude-sonnet-4-20250514    ← activo
  ▸ claude-opus-4-20250414
    claude-haiku-4-5

  ↑↓ navegar · enter seleccionar · esc cancelar

✅ Modelo base cambiado a claude-opus-4-20250414
```

Sin reiniciar. El historial de conversación se mantiene al cambiar. Implementado con Bubble Tea list/selector components.

### Config Persistida (`~/.hoa/config.json`)

```json
{
  "activeProvider": "anthropic",
  "providers": {
    "anthropic": {
      "apiKey": "enc:v1:aes256gcm:base64encodedciphertext...",
      "models": {
        "base": "claude-sonnet-4-20250514",
        "planning": "claude-opus-4-20250414"
      }
    },
    "openai": {
      "apiKey": "enc:v1:aes256gcm:base64encodedciphertext...",
      "models": {
        "base": "gpt-4o",
        "planning": "o3"
      }
    }
  },
  "database": {
    "url": "oracle://thinkstation:1521/FREEPDB1",
    "user": "memory_user",
    "password": "enc:v1:aes256gcm:base64encodedciphertext..."
  },
  "harness": {
    "verifyAfterWrite": true,
    "sddEnforced": true,
    "maxRetries": 3,
    "compactThreshold": 0.7
  }
}
```

### Encriptación de Secrets

Las API keys y passwords se encriptan en disco con AES-256-GCM. La clave de encriptación se deriva de una master key almacenada en `~/.hoa/keyring`:

```
~/.hoa/
├── config.json     # Config con secrets encriptados (prefijo "enc:v1:")
└── keyring         # Master key (256 bits, permisos 0600)
```

**Flujo:**
1. Primer uso → se genera master key aleatoria en `~/.hoa/keyring` (permisos `0600`)
2. Al guardar API key → `encrypt(masterKey, plaintext)` → se persiste como `enc:v1:aes256gcm:<base64>`
3. Al leer → detecta prefijo `enc:v1:` → `decrypt(masterKey, ciphertext)` → plaintext en memoria
4. Variables de entorno (`ANTHROPIC_API_KEY`, etc.) se usan como fallback sin encriptar

```go
// internal/config/crypto.go
func Encrypt(masterKey, plaintext []byte) (string, error) {
    block, _ := aes.NewCipher(masterKey)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    sealed := gcm.Seal(nonce, nonce, plaintext, nil)
    return "enc:v1:aes256gcm:" + base64.StdEncoding.EncodeToString(sealed), nil
}

func Decrypt(masterKey []byte, encoded string) ([]byte, error) {
    // strip prefix, base64 decode, extract nonce, gcm.Open
}
```

**Seguridad:**
- `~/.hoa/keyring` con permisos `0600` (solo el usuario puede leer)
- Si el keyring no existe o se pierde → las API keys se piden de nuevo
- En memoria las keys están en plaintext (inevitable para usarlas)
- Nunca se loguean ni se muestran en `/debug`

Cada provider persiste su propia config (API key + modelos base/planning). Al hacer `/provider`:

- **Provider ya configurado** → se activa directo con sus modelos guardados.
- **Provider sin config** → pide API key + selección de modelos antes de activar. Se persiste para la próxima vez.

```
> /provider

🔌 Provider activo: anthropic

    Anthropic (Claude)    ← activo
  ▸ OpenAI (GPT)              ← configurado
    Ollama (local)            ← no configurado
    Google (Gemini)           ← no configurado

  ↑↓ navegar · enter seleccionar · esc cancelar

✅ Provider cambiado a OpenAI (base: gpt-4o, planning: o3)
```

Si seleccionas uno no configurado:

```
  ▸ Google (Gemini)           ← no configurado

⚙️  Configurando Google por primera vez...

API Key: AIza••••••••

Modelo base (ejecución):
  ▸ gemini-2.5-pro
    gemini-2.5-flash

Modelo planning (razonamiento):
  ▸ gemini-2.5-pro
    gemini-2.5-flash

✅ Google configurado y activado (base: gemini-2.5-pro, planning: gemini-2.5-pro)
```

La config se persiste automáticamente. La próxima vez → arranca directo.

### Selección de Modelo por Fase

| Fase | Modelo Usado | Razón |
|------|-------------|-------|
| Planeamiento (SDD Proposal/Spec/Design) | `models.planning` | Razonamiento profundo, no necesita velocidad |
| Ejecución (Apply/Verify) | `models.base` | Velocidad + costo, tareas bien definidas |
| Verificación (loops) | `models.base` | Validación rápida contra spec |

---

## SDD — Spec-Driven Development

El agente NO puede saltar directo al código. El harness impone fases obligatorias:

```
┌─────────────────────────────────────────────────────────────┐
│  PROPOSAL → SPEC → DESIGN → TASK → APPLY → VERIFY          │
│                                                             │
│  [planning model]──────────────┐  [base model]─────────────│
│  Fases 1-3: razonamiento      │  Fases 4-6: ejecución     │
└─────────────────────────────────────────────────────────────┘
```

### Fases

| # | Fase | Artefacto | Gate de Avance |
|---|------|-----------|----------------|
| 1 | **Proposal** | `proposal.md` — Qué se quiere lograr, por qué, constraints | Usuario aprueba |
| 2 | **Spec** | `spec.md` — Comportamiento esperado, inputs/outputs, edge cases | Harness valida completitud |
| 3 | **Design** | `design.md` — Componentes, interfaces, dependencias, decisiones | Harness valida coherencia con spec |
| 4 | **Task** | Lista de tareas atómicas derivadas del design | Cada tarea es verificable |
| 5 | **Apply** | Código escrito (con write-then-verify loop) | Cada escritura pasa verificación |
| 6 | **Verify** | Validación final contra spec original | Tests pasan + spec cumplida |

### Artefactos SDD (almacenados en `.hoa/sdd/`)

```
.hoa/sdd/
├── current/
│   ├── proposal.md
│   ├── spec.md
│   ├── design.md
│   └── tasks.json       # Lista de tareas con estado
└── history/
    └── 2026-05-21_add-provider-router/
        └── ...           # Artefactos archivados post-commit
```

---

## Write-Then-Verify Loop

Cada escritura de código pasa por un bucle de validación. El agente NO declara victoria — el harness la confirma.

```
┌──────────────────────────────────────────────────────┐
│                  WRITE-THEN-VERIFY                    │
│                                                      │
│  ┌─────────┐    ┌──────────┐    ┌─────────────┐    │
│  │  WRITE  │───▶│  VERIFY  │───▶│  PASS/FAIL  │    │
│  └─────────┘    └──────────┘    └──────┬──────┘    │
│       ▲                                 │           │
│       │         ┌──────────┐            │           │
│       └─────────│  RETRY   │◀───────────┘ (fail)   │
│                 └──────────┘                        │
│                                                      │
│  Max retries: 3. Si falla → rollback + escalate.    │
└──────────────────────────────────────────────────────┘
```

### Niveles de Verificación

| Nivel | Check | Herramienta | Obligatorio |
|-------|-------|-------------|-------------|
| L0 | Archivo escrito correctamente | Diff check (str_replace matcheó) | Siempre |
| L1 | Syntax válida | Tree-sitter / compilador | Siempre |
| L2 | Compila | `go build ./...` | Si hay build tool |
| L3 | Lint pasa | `golangci-lint` / ESLint | Configurable |
| L4 | Tests pasan | `go test` / Jest / pytest | Si hay tests afectados |
| L5 | Spec cumplida | Comparar output vs spec.md | En fase Verify final |

### Comportamiento del Loop

```go
// harness/verify.go
func (h *Harness) RunVerifyLoop(ctx context.Context, agent *agent.Agent, tasks []Task) error {
    for _, task := range tasks {
        var lastErr error
        for retry := 0; retry < h.MaxRetries; retry++ {
            output, err := agent.Execute(ctx, task, task.Context)
            if err != nil {
                lastErr = err
                continue
            }
            result := h.Verify(output, task.Spec)
            if result.Passed {
                h.Accept(output)
                lastErr = nil
                break
            }
            // Feedback al agente con el error específico
            task.Context.InjectFeedback(result.Errors)
            lastErr = fmt.Errorf("verification failed: %s", result.Errors)
        }
        if lastErr != nil {
            h.Rollback(task)
            h.Escalate(task, "max retries reached")
        }
    }
    return nil
}
```

### Rollback Automático

Si la verificación falla 3 veces:
1. `git checkout -- <archivos_afectados>` (revert cambios)
2. Registrar el fallo en memoria vectorial (para no repetir)
3. Notificar al usuario con contexto del error
4. Opcionalmente: re-planear la tarea con el modelo de planeamiento

---

## Validación a Nivel de Harness

Inspirado en el principio de memory-management-mcp: **"Si un error se repite, se arregla el harness, no el código."**

### Invariantes del Harness

| Invariante | Enforcement |
|------------|-------------|
| No se escribe código sin spec | Gate en fase Apply |
| Cada escritura se verifica | Loop obligatorio |
| No se commitea código que no compila | Pre-commit hook |
| El agente no puede saltarse fases SDD | State machine en el harness |
| Archivos > 350 líneas → forzar split | Linter bespoke |
| Cada commit tiene what/why/intent | CommitTool valida campos |

### Eliminación Categórica

Cuando un error se repite:
1. Detectar patrón (via memoria vectorial: "este error ya ocurrió 2+ veces")
2. Crear regla en el harness que lo prevenga
3. La regla se ejecuta ANTES de que el agente actúe (no después)

---

## Tools de Fase 1

| Tool | Nivel | Descripción |
|------|-------|-------------|
| `read_file` | read-only | Leer archivo (con rango de líneas) |
| `write_file` | workspace | Crear archivo nuevo → trigger verify loop |
| `edit_file` | workspace | str_replace → trigger verify loop |
| `bash` | workspace | Ejecutar comando (cwd = proyecto) |
| `grep` | read-only | Búsqueda regex (ripgrep) |
| `glob` | read-only | Buscar archivos por patrón |
| `commit` | workspace | Commit inteligente con amnesia |
| `query_memory` | read-only | Buscar en memoria vectorial |
| `plan` | planning | Iniciar/avanzar flujo SDD (usa planning model) |
| `verify` | read-only | Ejecutar verificación manual contra spec |

---

## Commit Inteligente con Amnesia

### Flujo

```
commit(
  files: ["internal/provider/router.go", "internal/config/config.go"],
  message: "feat: add provider router with Anthropic support",
  what: "Router que selecciona Provider según config",
  why: "Necesario para multi-proveedor sin cambiar código"
)

Post-commit:
  1. Verificar que archivos compilan (L2 check)
  2. git add <files> && git commit -m <message>
  3. Indexar diff en memoria vectorial (intent/what/why + embedding)
  4. Archivar artefactos SDD en history/
  5. Flush cache de archivos commiteados
  6. Compactar contexto si > 70% del window
```

---

## Agent Loop con Modelo Dual

```
┌─────────────────────────────────────────────────────────────┐
│                      AGENT LOOP                              │
│                                                             │
│  [User Input]                                               │
│       │                                                     │
│       ▼                                                     │
│  ┌─────────────────┐                                       │
│  │ CLASSIFY INTENT │ → ¿Es planning o ejecución?           │
│  └────────┬────────┘                                       │
│           │                                                 │
│     ┌─────┴─────┐                                          │
│     ▼           ▼                                          │
│  [PLANNING]  [EXECUTION]                                   │
│  opus/o3     sonnet/gpt-4o                                 │
│     │           │                                          │
│     ▼           ▼                                          │
│  SDD Phases  Write-Verify Loop                             │
│     │           │                                          │
│     └─────┬─────┘                                          │
│           ▼                                                 │
│  [Tool Calls] → [Verify] → [Accept/Retry]                 │
│           │                                                 │
│           ▼                                                 │
│  [Response to User]                                        │
└─────────────────────────────────────────────────────────────┘
```

---

## Componentes a Implementar

| Componente | Responsabilidad |
|------------|-----------------|
| `ConfigManager` | Primer uso, multi-provider, persistir config, CLI flags |
| `ProviderRouter` | Instanciar Provider según config + seleccionar modelo por fase |
| `SDDEngine` | State machine de fases, gates de avance, artefactos |
| `WriteVerifyLoop` | Bucle write → verify → retry/rollback |
| `HarnessValidator` | Invariantes, linters bespoke, eliminación categórica |
| `ToolRegistry` | Registrar tools, resolver por nombre, validar permisos |
| `ToolExecutor` | Ejecutar tool, capturar output, trigger verify si es escritura |
| `CommitTool` | git + verify + indexar + flush + archivar SDD |
| `SessionCache` | `sync.Map` con invalidación por archivo |
| `MemoryClient` | Cliente hacia vector store |
| `AgentLoop` | Dual-model routing + tool calls + verify loops |
| `ContextManager` | Token budget, compactación post-commit, progressive disclosure |

---

## Orden de Implementación

```
1. ConfigManager (multi-provider, CLI flags, primer uso)
2. ProviderRouter (dual-model: base + planning)
3. AgentLoop básico (prompt → modelo → respuesta)
4. ToolRegistry + ToolExecutor + tools read-only
5. SDDEngine (state machine de fases, artefactos)
6. WriteVerifyLoop (L0-L2 obligatorios)
7. Tools de escritura (write_file, edit_file) integrados con verify loop
8. HarnessValidator (invariantes, rollback)
9. SessionCache (invalidación por archivo)
10. MemoryClient (conexión a BD vectorial)
11. CommitTool (git + indexar + flush + archivar)
12. ContextManager (token budget + compactación + progressive disclosure)
```

---

## Diferencias con Claude Code

| Aspecto | Claude Code | HOA |
|---------|-------------|------------|
| Provider | Solo Anthropic (+ Bedrock) | Multi-provider real |
| Planeamiento | Plan mode opcional | SDD obligatorio con gates |
| Verificación | El modelo se auto-valida | El harness valida (no confía en el modelo) |
| Escritura | Write → hope it works | Write → verify → retry/rollback |
| Memoria | Session memory + CLAUDE.md | Vectorial persistente + amnesia por commit |
| Errores repetidos | El usuario los detecta | El harness los detecta y crea reglas |
| Modelo dual | Un solo modelo | Planning model + execution model |

---

## Decisión Final: Harness Agent Personalizado en Go

### Contexto

La decisión es **Go** con un harness personalizado basado en la arquitectura de [`byo-coding-agent`](https://github.com/betta-tech/byo-coding-agent). Este proyecto sirve como referencia de implementación — un agente de código funcional (~1000 líneas bajo `internal/`) que demuestra exactamente los patrones que HOA necesita.

La diferencia clave: **byo-coding-agent usa API keys directas** (variables de entorno `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`). No hay intermediarios, no hay SDKs pesados de cloud providers. HOA adoptará el mismo modelo de acceso directo a las APIs de los modelos.

### Arquitectura Base (extraída de byo-coding-agent)

```
┌─────────────────────────────────────────────────────────────┐
│  main.go          wiring · REPL · agent loop · subagents    │
│  commands.go      /help · /model · /compact · /provider …   │
└─────────────────────────────────────────────────────────────┘
        │
        ├── internal/api/          tipos genéricos (Message, Block, ToolDef, Response)
        │
        ├── internal/provider/     interfaz Provider + impl por proveedor
        │     Un archivo por backend. Solo ese archivo importa el SDK.
        │
        ├── internal/tool/         interfaz Tool + Registry auto-registrante
        │     Un archivo por herramienta. init() registra. Drop-in.
        │
        ├── internal/agent/        Agent struct + loop + diff approval
        │     Subagents son Agents con su propio scope.
        │
        ├── internal/compact/      CompactionStrategy + estrategias
        │     SlidingWindow, Summarize, NoCompaction, SafeSplitPoint.
        │
        ├── internal/memory/       Persistencia entre sesiones
        │     Session files + preamble + auto-summary al shutdown.
        │
        ├── internal/subagent/     Delegación de tareas read-only
        │     Research subagent con tool subset limitado.
        │
        ├── internal/mcp/          Soporte MCP (servidores externos)
        │     Conexión async en background, no bloquea startup.
        │
        └── internal/ui/           TUI Bubble Tea completa
              Input, spinner, banner, diff approval modal, debug panel.
```

### Interfaz Provider — El Contrato Central

```go
// El harness SOLO habla con esta interfaz. Cambiar de modelo = cambiar una línea.
type Provider interface {
    Send(ctx context.Context, messages []api.Message, tools []api.ToolDef) (api.Response, error)
    Model() string
    SetModel(name string)
}
```

Cada provider implementa esta interfaz y es el **único archivo** que importa el SDK del vendor:

| Provider | SDK | Env Var | Default Model |
|----------|-----|---------|---------------|
| Anthropic | `anthropic-sdk-go` | `ANTHROPIC_API_KEY` | `claude-opus-4-7` |
| OpenAI | `openai-go` | `OPENAI_API_KEY` | `gpt-5-codex` |
| Ollama | HTTP directo | `OLLAMA_BASE_URL` | configurable |
| Google | HTTP directo | `GOOGLE_API_KEY` | `gemini-2.5-pro` |

**Swap en runtime sin reiniciar:**

```
/provider openai gpt-4o       # cambia provider + modelo
/provider anthropic            # vuelve a Anthropic con default
/model claude-sonnet-4-20250514         # cambia solo el modelo
```

### Tipos Genéricos (Provider-Agnostic)

```go
// internal/api/types.go — El vocabulario universal del harness
type Message struct {
    Role    Role    // "user" | "assistant"
    Content []Block
}

type Block struct {
    Type      BlockType // "text" | "tool_use" | "tool_result"
    Text      string
    ToolUseID string
    ToolName  string
    ToolInput string    // raw JSON — pass-through al provider
    ToolResult string
    IsError    bool
}

type Response struct {
    Content    []Block
    StopReason StopReason // end_turn | tool_use | other
    Usage      Usage
}
```

Los providers traducen de/hacia estos tipos. El resto del harness (tools, compaction, agent loop) **nunca** toca tipos de SDK.

### Agent Loop — El Corazón

```go
func (a *Agent) loop(ctx context.Context) (string, error) {
    for turn := 0; turn < a.MaxTurns; turn++ {
        // 1. Compactar si la estrategia lo decide
        a.messages = a.Compactor.Compact(ctx, a.messages)

        // 2. Llamar al modelo
        resp := a.Provider.Send(ctx, a.messages, a.Tools.Definitions())

        // 3. Procesar respuesta
        a.messages = append(a.messages, Message{Role: Assistant, Content: resp.Content})

        // 4. Si hay tool_use → ejecutar → agregar results → volver a 2
        var toolResults []Block
        for _, b := range resp.Content {
            if b.Type == BlockToolUse {
                result, isErr := a.executeTool(ctx, b.ToolName, b.ToolInput)
                toolResults = append(toolResults, Block{
                    Type: BlockToolResult, ToolUseID: b.ToolUseID,
                    ToolResult: result, IsError: isErr,
                })
            }
        }

        // 5. Si no hay tool calls → terminó
        if resp.StopReason != StopToolUse {
            return finalText, nil
        }
        a.messages = append(a.messages, Message{Role: User, Content: toolResults})
    }
    return "", fmt.Errorf("max turns (%d) reached", a.MaxTurns)
}
```

### Tool Registry — Drop-in Pattern

```go
// Interfaz que cada tool implementa
type Tool interface {
    Definition() api.ToolDef
    Execute(ctx context.Context, input string) (result string, isError bool)
}

// Registry con auto-registro via init()
var Default = NewRegistry()

// Agregar una tool = crear un archivo. Ejemplo:
// internal/tool/grep.go
func init() { Default.Register(&GrepTool{}) }
```

**Tools de HOA (extendidas sobre byo-coding-agent):**

| Tool | Categoría | Descripción |
|------|-----------|-------------|
| `bash` | workspace | Ejecutar comando shell |
| `read_file` | read-only | Leer archivo |
| `write_file` | workspace | Escribir archivo (con diff approval) |
| `grep` | read-only | Búsqueda regex |
| `glob` | read-only | Buscar archivos por patrón |
| `remember` | memory | Persistir dato entre sesiones |
| `recall` | memory | Buscar en memoria persistente |
| `delegate_research` | subagent | Delegar investigación read-only |
| `commit` | workspace | **HOA:** git + verify + indexar + amnesia |
| `plan` | planning | **HOA:** iniciar/avanzar flujo SDD |
| `verify` | harness | **HOA:** ejecutar verificación contra spec |

### Compaction — Manejo de Contexto Largo

```go
type CompactionStrategy interface {
    Compact(ctx context.Context, messages []api.Message) ([]api.Message, error)
}
```

| Estrategia | Comportamiento |
|------------|----------------|
| `NoCompaction` | Default — no toca mensajes |
| `SlidingWindow{KeepLast: N}` | Descarta mensajes antiguos, conserva últimos N |
| `Summarize{Threshold, KeepRecent}` | Pide al modelo resumir turnos antiguos |
| `WithLogging(inner, file)` | Decorador que graba antes/después |

**SafeSplitPoint:** Garantiza que nunca se corta entre un `tool_use` y su `tool_result`. Camina hacia atrás hasta encontrar un límite limpio.

```go
func SafeSplitPoint(messages []Message, desired int) int {
    for i := desired; i > 0; i-- {
        if messages[i].Role == RoleUser && !messages[i].HasToolResult() {
            return i
        }
    }
    return 0
}
```

### Subagents — Delegación con Contexto Aislado

```go
// Un subagent es otro Agent con su propio scope
type Research struct {
    Provider provider.Provider
    Tools    *tool.Registry  // subset limitado (solo read_file)
}
```

El root agent delega tareas read-only a subagents que tienen su propio context window. Esto evita contaminar el contexto principal con lecturas exploratorias.

**Patrón de uso:**
- Root agent recibe pregunta que requiere investigación
- Llama `delegate_research` con la pregunta
- Subagent lee archivos, busca, y devuelve resumen
- Root agent usa el resumen sin haber gastado su context window

### Memory — Persistencia Entre Sesiones

```go
// Al shutdown: auto-summarize la sesión
func summarizeSession(ctx context.Context, p provider.Provider, history []api.Message) (string, []string) {
    // Pide al modelo un párrafo + tags
    // Se persiste en .harness/sessions/
}

// Al startup: carga preamble con sesiones recientes
func (m *SessionFiles) Preamble(ctx context.Context) (string, error) {
    // Inyecta resúmenes de sesiones anteriores al system prompt
}
```

**Tools de memoria:**
- `remember(content, kind, tags)` — Guardar dato persistente
- `recall(query)` — Buscar en sesiones anteriores

### Diff Approval — Permission Gate para Escrituras

```go
// El agent pide confirmación antes de escribir
rootAgent.Confirm = func(prompt, detail string) bool {
    reply := make(chan bool, 1)
    program.Send(ui.ApprovalRequest{Prompt: prompt, Detail: detail, Reply: reply})
    return <-reply
}
```

Para `write_file`, el harness computa un diff unificado contra el contenido actual y lo muestra en un modal. El usuario aprueba o rechaza cada escritura.

### MCP — Servidores de Herramientas Externos

```json
// mcp.json — configuración de servidores MCP
{
  "servers": {
    "memory-management": {
      "command": "./memory-management-mcp",
      "args": ["--db-url", "oracle://thinkstation:1521/FREEPDB1"],
      "env": { "DB_USER": "memory_user" }
    }
  }
}
```

- Conexión async en background (no bloquea startup)
- Cada servidor registra sus tools en el Registry global
- El usuario puede trabajar mientras los servidores conectan

### Debug Panel — Observabilidad en Tiempo Real

```
/debug on     # Activa panel lateral con:
              # - Llamadas al provider (request/response)
              # - Dispatch de herramientas (input/output/timing)
              # - Eventos de compactación
              # - Costos acumulados
```

---

## HOA = byo-coding-agent + Capas Propias

### Lo que heredamos de byo-coding-agent

| Componente | Estado | Adaptación |
|------------|--------|------------|
| Agent Loop | ✅ Funcional | Extender con dual-model routing |
| Provider Interface | ✅ Anthropic + OpenAI | Agregar Ollama, Google, Bedrock |
| Tool Registry (drop-in) | ✅ Funcional | Agregar tools de HOA |
| Compaction Strategies | ✅ 3 estrategias | Agregar post-commit compaction |
| Subagents | ✅ Research | Agregar planning subagent |
| Memory (session files) | ✅ Funcional | Migrar a Oracle 23ai vectorial |
| MCP Support | ✅ Funcional | Conectar memory-management-mcp |
| TUI (Bubble Tea) | ✅ Completa | Adaptar branding + SDD status |
| Diff Approval | ✅ Funcional | Integrar con verify loop |
| Debug Panel | ✅ Funcional | Agregar harness events |

### Lo que HOA agrega encima

| Componente | Descripción | Prioridad |
|------------|-------------|-----------|
| **Dual-Model Router** | Planning model (opus/o3) vs execution model (sonnet/gpt-4o) | P0 |
| **SDD Engine** | State machine: Proposal → Spec → Design → Task → Apply → Verify | P0 |
| **Write-Verify Loop** | L0-L5 verification levels con retry/rollback | P0 |
| **Harness Hooks** | Pre/Post hooks en cada punto del agent loop | P0 |
| **Commit Tool** | git + verify + indexar en vectorial + amnesia | P1 |
| **Oracle 23ai Memory** | Reemplazar session files por vector store real | P1 |
| **Eliminación Categórica** | Detectar errores repetidos → crear regla preventiva | P1 |
| **Progressive Disclosure** | Solo inyectar contexto relevante a la tarea | P2 |
| **Skill Discovery** | Detectar herramientas del proyecto (go, npm, make, etc.) | P2 |

---

## Estructura Final del Proyecto

```
hoa/
├── cmd/
│   └── hoa/
│       └── main.go                 # Wiring + REPL + startup
├── commands.go                     # Slash commands (/help, /model, /sdd, ...)
├── internal/
│   ├── api/
│   │   └── types.go               # Message, Block, ToolDef, Response, Usage
│   ├── provider/
│   │   ├── provider.go            # Interface Provider
│   │   ├── anthropic.go           # Anthropic (anthropic-sdk-go)
│   │   ├── openai.go              # OpenAI (openai-go)
│   │   ├── ollama.go              # Ollama (HTTP directo)
│   │   ├── google.go              # Google Gemini (HTTP directo)
│   │   └── router.go              # Dual-model: planning vs execution
│   ├── agent/
│   │   ├── agent.go               # Agent struct + loop
│   │   ├── diff.go                # Diff computation para approval
│   │   └── router.go              # Intent classification → model selection
│   ├── tool/
│   │   ├── registry.go            # Registry + auto-register via init()
│   │   ├── bash.go
│   │   ├── readfile.go
│   │   ├── writefile.go
│   │   ├── grep.go
│   │   ├── glob.go
│   │   ├── commit.go              # git + verify + index + amnesia
│   │   ├── plan.go                # SDD flow control
│   │   ├── verify.go              # Manual verification trigger
│   │   ├── remember.go            # Persist to memory
│   │   └── recall.go              # Search memory
│   ├── harness/
│   │   ├── hooks.go               # HookPoint enum + Harness interface
│   │   ├── sdd.go                 # SDD state machine + gates
│   │   ├── verify.go              # Write-then-verify loop (L0-L5)
│   │   ├── invariants.go          # Reglas deterministas
│   │   └── categorical.go         # Eliminación categórica de errores
│   ├── compact/
│   │   ├── strategy.go            # Interface + SafeSplitPoint
│   │   ├── nocompaction.go
│   │   ├── slidingwindow.go
│   │   ├── summarize.go
│   │   └── postcommit.go          # Compactación post-commit (amnesia)
│   ├── memory/
│   │   ├── store.go               # Interface VectorStore
│   │   ├── sessionfiles.go        # File-based (dev/fallback)
│   │   └── oracle.go              # Oracle 23ai + DBMS_VECTOR_CHAIN
│   ├── subagent/
│   │   ├── registry.go            # Subagent registry
│   │   ├── research.go            # Read-only investigation
│   │   └── planning.go            # SDD planning subagent
│   ├── mcp/
│   │   ├── client.go              # MCP client (stdio/HTTP)
│   │   └── register.go            # Auto-register MCP tools
│   ├── config/
│   │   └── config.go              # Load/Save/Wizard ~/.hoa/config.json
│   ├── debug/
│   │   └── debug.go               # Event recording + sink
│   └── ui/
│       ├── program.go             # Bubble Tea main program
│       ├── input.go               # Input box con historial
│       ├── banner.go              # Startup banner
│       ├── spinner.go             # Loading indicator
│       └── styles.go              # Lipgloss styles
├── AGENTS.md                       # Project context (inyectado al system prompt)
├── mcp.json                        # MCP server config
├── go.mod
└── go.sum
```

---

## Orden de Implementación Revisado

Basado en la arquitectura de byo-coding-agent como punto de partida:

```
FASE 1A — Core Agent (fork/adapt de byo-coding-agent)
═══════════════════════════════════════════════════════
1. ✅ Scaffold: go mod init + internal/api, internal/provider, internal/tool
2. ✅ Config: ~/.hoa/config.json con wizard de primer uso
3. ✅ Provider Router: dual-model (planning vs execution)
4. ✅ Agent Loop: agent.go con MaxTurns + tool dispatch
5. ✅ Tools básicas: bash, read_file, grep, glob
6. ✅ TUI: Bubble Tea program con branding HOA
7. ✅ Slash commands: /help, /model, /provider, /tokens, /mode, /memory, /commit, /feedback

FASE 1B — Harness Layer (lo que diferencia a HOA)
═══════════════════════════════════════════════════════════
8. ❌ Harness Hooks: interface + hook points en el agent loop
9. ❌ SDD Engine: state machine (Proposal → Spec → Design → Task → Apply → Verify)
10. ❌ Write-Verify Loop: L0-L2 obligatorios, L3-L5 configurables
11. ✅ Commit Tool: /commit con LLM + validación + post-commit memory push
12. ❌ Compaction post-commit: flush context de archivos commiteados

FASE 1C — Memory & Intelligence ← COMPLETADA
═════════════════════════════════════════════
13. ✅ Memory Provider: Oracle 23ai con go-ora (conexión directa, sin Instant Client)
14. ✅ Extracción determinista: git log/diff → memory_changes + hunks (sin LLM)
15. ✅ Enrichment concurrente: cola async + LLM para commits legacy
16. ✅ Búsqueda semántica: VECTOR_DISTANCE + embeddings ONNX (Oracle nativo)
17. ✅ Working Context: git diff como memoria de sesión (auto-limpia post-commit)
18. ✅ Feedback Rules: correcciones del usuario con evolución (superseded_by)
19. ✅ Prompt Caching: system + automatic caching (90% ahorro en turns 2+)
20. ✅ Context Injection: working_changes + feedback_rules + project_memory → LLM
21. ✅ Metadata Trimming: ignora binarios, lockfiles, generados

FASE 1D — Polish
═════════════════
22. ❌ Subagent Research: delegación read-only
23. ❌ Progressive Disclosure: inyectar solo skills relevantes
24. ❌ Debug panel: harness events + cost tracking
```

---

## Dependencias (go.mod)

```go
module github.com/cloudcentinel/hoa

go 1.23

require (
    // SDKs de providers (API key directa)
    github.com/anthropics/anthropic-sdk-go v1.2.0
    github.com/openai/openai-go v0.1.0

    // TUI
    github.com/charmbracelet/bubbletea v1.2.4
    github.com/charmbracelet/lipgloss v1.0.0

    // Oracle (memoria vectorial)
    github.com/godror/godror v0.44.8

    // Concurrencia
    golang.org/x/sync v0.10.0
)
```

**Sin cobra.** El REPL es el modo principal de interacción. Los slash commands (`/help`, `/model`, etc.) reemplazan subcommands CLI. El binario se ejecuta con `hoa` y entra directo al agent loop.

---

## Modelo de Acceso a APIs

```
┌─────────────────────────────────────────────────────────┐
│                    HOA                             │
│                                                         │
│  ~/.hoa/config.json                              │
│  ┌───────────────────────────────────────────────────┐  │
│  │ providers:                                        │  │
│  │   anthropic: { apiKey: "sk-ant-..." }             │  │
│  │   openai:    { apiKey: "sk-..." }                 │  │
│  │   google:    { apiKey: "AIza..." }                │  │
│  │   ollama:    { baseUrl: "http://localhost:11434" } │  │
│  └───────────────────────────────────────────────────┘  │
│                         │                               │
│                         ▼                               │
│  ┌─────────────────────────────────────────────────┐   │
│  │         Provider Interface (Send)                │   │
│  │                                                  │   │
│  │  ┌──────────┐ ┌────────┐ ┌────────┐ ┌───────┐  │   │
│  │  │Anthropic │ │ OpenAI │ │ Google │ │Ollama │  │   │
│  │  │  SDK     │ │  SDK   │ │  HTTP  │ │ HTTP  │  │   │
│  │  └────┬─────┘ └───┬────┘ └───┬────┘ └───┬───┘  │   │
│  └───────┼────────────┼──────────┼──────────┼──────┘   │
│          │            │          │          │           │
└──────────┼────────────┼──────────┼──────────┼───────────┘
           ▼            ▼          ▼          ▼
    api.anthropic  api.openai  api.google  localhost
      .com           .com       .com        :11434
```

**No hay intermediarios.** No hay Bedrock wrapping Anthropic. No hay Azure wrapping OpenAI. API keys directas al vendor. Esto simplifica:
- Debugging (un solo hop)
- Pricing (rates del vendor directo)
- Features (acceso inmediato a nuevas capabilities)
- Latencia (sin proxy adicional)

---

## Resumen Ejecutivo

| Aspecto | Decisión |
|---------|----------|
| Lenguaje | **Go** |
| Base de código | Fork/adapt de **byo-coding-agent** |
| Acceso a modelos | **API keys directas** (no cloud wrappers) |
| Providers iniciales | Anthropic + OpenAI |
| TUI | **Bubble Tea** (heredada de byo-coding-agent) |
| Memoria | **Session files** → migrar a **Oracle 23ai** |
| Diferenciador | **Harness layer** (SDD + verify loop + hooks + eliminación categórica) |
| Binario final | ~25 MB, startup <100ms, zero dependencies en runtime |
