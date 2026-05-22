# Harnesses — Catálogo Completo

[← INDEX](../INDEX.md)

---

## Principio

```
Harness = Regla determinista que se ejecuta sobre el comportamiento probabilístico del modelo.
```

El modelo genera. El harness valida, controla, y corrige. Si un error se repite, no se arregla el prompt — se agrega un harness que lo prevenga.

---

## Arquitectura

```
┌─────────────────────────────────────────────────────────────────┐
│                         AGENT LOOP                               │
│                                                                 │
│  [user input]                                                   │
│       │                                                         │
│       ▼                                                         │
│  ┌──────────────┐                                              │
│  │ ON_SESSION   │ → Memory load, skill discovery, setup check  │
│  └──────┬───────┘                                              │
│         ▼                                                       │
│  ┌──────────────┐                                              │
│  │ PRE_LLM_CALL │ → Context trim, budget check, disclosure    │
│  └──────┬───────┘                                              │
│         ▼                                                       │
│  ┌──────────────┐                                              │
│  │   LLM CALL   │                                              │
│  └──────┬───────┘                                              │
│         ▼                                                       │
│  ┌──────────────┐                                              │
│  │POST_LLM_CALL │ → Cost track, response validation           │
│  └──────┬───────┘                                              │
│         ▼                                                       │
│  ┌──────────────┐                                              │
│  │PRE_TOOL_EXEC │ → Permission gate, SDD gate, skill inject   │
│  └──────┬───────┘                                              │
│         ▼                                                       │
│  ┌──────────────┐                                              │
│  │  TOOL EXEC   │                                              │
│  └──────┬───────┘                                              │
│         ▼                                                       │
│  ┌──────────────┐                                              │
│  │POST_TOOL_EXEC│ → Verify loop, cache invalidation           │
│  └──────┬───────┘                                              │
│         ▼                                                       │
│  ┌──────────────┐                                              │
│  │  PRE_COMMIT  │ → Compile, lint, test, diff review          │
│  └──────┬───────┘                                              │
│         ▼                                                       │
│  ┌──────────────┐                                              │
│  │ POST_COMMIT  │ → Memory index, cache flush, compact        │
│  └──────┬───────┘                                              │
│         ▼                                                       │
│  [response to user]                                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## Interface

```go
type HookPoint int

const (
    OnSessionStart HookPoint = iota
    PreLLMCall
    PostLLMCall
    PreToolExec
    PostToolExec
    PreCommit
    PostCommit
    OnError
    OnSessionEnd
)

type Harness interface {
    Name() string
    Hooks() []HookPoint
    Execute(ctx context.Context, event *Event) error
}

type Event struct {
    Point      HookPoint
    ToolName   string
    Input      any
    Output     any
    Session    *Session
    Abort      bool       // Cancela la operación
    Feedback   string     // Inyecta mensaje al agente
    Metadata   map[string]any
}
```

---

## Harnesses Core (Fase 1)

---

### H01 — SDD State Machine

**Hook:** `PreToolExec`

El agente no puede escribir código sin haber pasado por las fases de planeamiento. Impone disciplina estructural.

| Estado | Tools permitidas | Transición |
|--------|-----------------|------------|
| `idle` | Todas read-only | → `proposal` cuando usuario pide feature |
| `proposal` | read_file, grep, query_memory | → `spec` cuando usuario aprueba |
| `spec` | read_file, grep, query_memory | → `design` cuando harness valida completitud |
| `design` | read_file, grep, query_memory | → `task` cuando harness valida coherencia |
| `task` | read_file, grep | → `apply` cuando tareas generadas |
| `apply` | TODAS (write, edit, bash) | → `verify` cuando tareas completadas |
| `verify` | bash (tests), read_file | → `idle` cuando spec cumplida |

**Bypass:** `hoa --no-sdd` para tareas triviales (hotfix, typo). Se registra en memoria que se saltó.

---

### H02 — Progressive Context Disclosure

**Hooks:** `OnSessionStart`, `PreLLMCall`

No inyecta todo el contexto al inicio. Busca en memoria vectorial solo lo relevante a la tarea actual. Reduce context rot.

**Comportamiento:**

```
OnSessionStart:
  - Cargar solo: project metadata + últimos 3 commits + active branch info
  - NO cargar: historial completo, skills, decisiones antiguas

PreLLMCall:
  - Analizar intent del usuario
  - query_memory(intent, limit=5) → inyectar resultados relevantes
  - Si hay skill relevante → cargar solo esa skill (ver H12)
  - Calcular tokens usados vs budget → trim si necesario
```

**Reglas de inyección:**

| Dato | Cuándo se inyecta | Cuándo NO |
|------|-------------------|-----------|
| Commits recientes (3) | Siempre | — |
| Commits relacionados | Si query_memory los encuentra | Si score < 0.7 |
| Skills | Si la tarea matchea | Nunca al inicio |
| Decisiones arquitectónicas | Si toca archivos de infra | Para cambios triviales |
| Errores previos similares | Si el patrón matchea | Si ya se resolvió |

---

### H03 — Write-Verify Loop

**Hook:** `PostToolExec` (solo para write_file, edit_file)

Cada escritura se valida automáticamente. El agente no declara victoria.

**Niveles:**

| Nivel | Check | Cómo | Obligatorio |
|-------|-------|------|-------------|
| L0 | Escritura correcta | Verificar que str_replace matcheó / archivo existe | Siempre |
| L1 | Syntax válida | `tree-sitter parse` o compilador del lenguaje | Siempre |
| L2 | Compila | Detectar build tool → ejecutar build | Si existe build tool |
| L3 | Lint | Detectar linter → ejecutar | Configurable |
| L4 | Tests | Ejecutar tests afectados | Si existen |
| L5 | Spec check | Comparar contra spec.md | Solo en fase Verify |

**Retry con feedback:**

```
Intento 1: write_file → L1 falla (syntax error en línea 42)
  → Inyectar al agente: "Error de syntax en línea 42: unexpected token '}'"
Intento 2: edit_file (fix) → L2 falla (import faltante)
  → Inyectar: "Compilation error: cannot find symbol 'HttpClient'"
Intento 3: edit_file (fix) → ✅ Todos pasan
```

**Si falla 3 veces:** rollback + registrar en memoria + escalar al usuario.

---

## Harnesses de Memoria

---

### H04 — Amnesia Post-Commit

**Hook:** `PostCommit`

Después de cada commit, el contexto se limpia. Solo sobrevive lo que está en la memoria vectorial.

**Acciones:**
1. Indexar diff + metadata (what/why/intent) en vector store
2. Flush cache de archivos commiteados
3. Si contexto > 70% del window → compactar conversación (resumir turnos pre-commit)
4. Archivar artefactos SDD en `history/`

**Resultado:** El agente empieza "fresco" después de cada commit. Si necesita contexto anterior, lo busca en memoria.

---

### H05 — Eliminación Categórica

**Hook:** `OnError`

Si un error se repite 2+ veces (detectado via memoria vectorial), se crea una regla automática.

**Flujo:**

```
1. Error ocurre
2. query_memory("error: <tipo>", project) → ¿ya ocurrió antes?
3. Si count >= 2:
   a. Generar regla con el planning model:
      "Antes de escribir archivos Java, verificar que todos los imports existen"
   b. Persistir regla en ~/.hoa/rules/<project>.json
   c. Registrar como harness dinámico que se ejecuta en PreToolExec
4. Próxima vez: la regla se ejecuta ANTES de que el agente actúe
```

**Ejemplo de regla generada:**

```json
{
  "id": "rule_001",
  "trigger": "pre_tool:write_file",
  "condition": "file.extension == '.java'",
  "action": "inject_feedback",
  "message": "Antes de escribir: verifica que todos los imports referenciados existen en el classpath. Usa grep para confirmar.",
  "created_from_error": "CompilationError: cannot find symbol",
  "occurrences": 3
}
```

---

### H06 — Context Compactor

**Hook:** `PreLLMCall`

Cuando el contexto se acerca al límite del window, compacta automáticamente.

**Estrategia:**

| % del window | Acción |
|--------------|--------|
| < 50% | Nada |
| 50-70% | Comprimir tool outputs antiguos (solo mantener resumen) |
| 70-85% | Resumir turnos anteriores al último commit |
| > 85% | Compactación agresiva: solo mantener último commit + tarea actual |

**Compactación:**
- Usa el modelo base para generar un resumen de los turnos eliminados
- El resumen se inyecta como "contexto previo" en el siguiente prompt
- Los datos completos siguen en memoria vectorial (recuperables via query)

---

## Harnesses de Calidad

---

### H07 — TDD Enforcer

**Hook:** `PreToolExec` (write_file, edit_file)

Obliga el ciclo Red → Green → Refactor. No acepta código de producción sin test.

**State machine:**

```
[idle] → usuario pide feature
  → [red] — Agente DEBE escribir test primero (que falle)
    → [green] — Agente escribe código mínimo para pasar el test
      → [refactor] — Agente puede limpiar sin romper tests
        → [idle]
```

**Enforcement:**
- En estado `red`: solo permite escribir en `*_test.*` / `*Test.*` / `test_*.*`
- En estado `green`: permite escribir en src, pero ejecuta test después
- En estado `refactor`: permite editar src, pero tests deben seguir pasando

**Bypass:** Configurable por proyecto. Algunos proyectos no tienen tests (legacy).

---

### H08 — Architecture Linter

**Hook:** `PreCommit`

Valida que el código generado respete las reglas arquitectónicas del proyecto.

**Reglas configurables (por proyecto):**

```json
{
  "rules": [
    {"max_file_lines": 350, "action": "block"},
    {"no_circular_deps": true, "action": "warn"},
    {"layer_deps": {
      "controller": ["service"],
      "service": ["repository", "client"],
      "repository": []
    }},
    {"no_business_logic_in": ["controller", "config"]},
    {"every_public_method_has_test": true, "action": "warn"}
  ]
}
```

**Detección:**
- Parsear imports/dependencias del archivo modificado
- Verificar contra reglas del proyecto
- Si viola → feedback al agente con la regla específica

---

### H09 — Diff Reviewer

**Hook:** `PreCommit`

Sub-agente (modelo base) que revisa el diff completo antes de aceptar el commit.

**Prompt del reviewer:**

```
Revisa este diff. Busca:
1. Código muerto o no usado
2. Errores lógicos obvios
3. Violaciones de naming conventions
4. TODOs sin resolver
5. Secrets hardcodeados
6. Inconsistencias con el design.md

Si encuentras problemas, lista cada uno con archivo:línea.
Si está limpio, responde "LGTM".
```

**Si encuentra problemas:** inyecta feedback al agente para que corrija antes del commit.

---

### H10 — Security Scanner

**Hook:** `PostToolExec` (write_file, edit_file)

Detecta vulnerabilidades comunes en código generado.

**Checks:**

| Categoría | Qué detecta |
|-----------|-------------|
| Secrets | API keys, passwords, tokens hardcodeados |
| Injection | SQL sin parametrizar, command injection |
| Auth | Endpoints sin autenticación |
| Crypto | Algoritmos débiles (MD5, SHA1 para passwords) |
| SSRF | URLs construidas con input del usuario |
| Path traversal | File paths sin sanitizar |

**Implementación:** Regex patterns + heurísticas simples. No es un SAST completo — es un first-pass rápido.

---

## Harnesses de Recursos

---

### H11 — Cost Guardian

**Hook:** `PostLLMCall`

Trackea el costo acumulado de la sesión y corta si supera el budget.

**Config:**

```json
{
  "budget": {
    "session_max_usd": 5.00,
    "daily_max_usd": 20.00,
    "warn_at_percent": 80
  }
}
```

**Comportamiento:**
- Cada respuesta del LLM → calcular costo (input tokens × price + output tokens × price)
- Acumular en sesión y en día
- Al 80%: warning al usuario
- Al 100%: pausar y pedir confirmación para continuar

**Pricing table (actualizable):**

```go
var pricing = map[string]TokenPrice{
    "claude-sonnet-4-20250514":  {Input: 3.0, Output: 15.0},  // per 1M tokens
    "claude-opus-4-20250414":    {Input: 15.0, Output: 75.0},
    "gpt-4o":                    {Input: 2.5, Output: 10.0},
}
```

---

### H12 — Dynamic Skill Loader

**Hook:** `PreLLMCall`

Skills almacenadas en memoria vectorial. Se cargan SOLO cuando la tarea las necesita. Nunca contaminan el contexto base.

**Concepto:**

```
Skills en vector store:
  - "skill:docker-compose" → Cómo generar docker-compose para este proyecto
  - "skill:spring-security" → Patrones de auth para este proyecto
  - "skill:terraform-modules" → Convenciones de IaC del equipo
  - "skill:api-design" → Estándares REST del proyecto
  - "skill:error-handling" → Patrón de manejo de errores
```

**Flujo:**

```
1. Usuario pide: "agrega autenticación JWT al endpoint"
2. PreLLMCall → analizar intent
3. query_memory("skill:*", intent="autenticación JWT", limit=3)
4. Resultados: ["skill:spring-security", "skill:api-design"]
5. Inyectar SOLO esas skills al contexto del prompt
6. El agente ejecuta con el conocimiento específico
7. Post-ejecución: las skills se descartan del contexto
```

**Estructura de una skill en BD:**

```sql
INSERT INTO memory_skills (id, project_id, name, content, embedding)
VALUES (
  'skill_001',
  'hoa',
  'skill:error-handling',
  'En este proyecto, todos los errores se manejan con: 1) Custom error types en pkg/errors. 2) Wrap con contexto: fmt.Errorf("operation: %w", err). 3) Logging en el boundary (handler/controller), no en capas internas. 4) Nunca panic en código de producción.',
  DBMS_VECTOR_CHAIN.UTL_TO_EMBEDDING(...)
);
```

**Gestión de skills:**

```
$ hoa skill add "error-handling" --from-file ./docs/error-patterns.md
$ hoa skill list
$ hoa skill remove "old-pattern"
$ hoa skill search "cómo manejar errores"
```

---

### H13 — Session Resumption

**Hook:** `OnSessionStart`, `OnSessionEnd`

Si la sesión se interrumpe (ctrl+c, crash, timeout), puede restaurar estado.

**OnSessionEnd:**
- Persistir en `~/.hoa/sessions/<id>.json`:
  - SDD state actual
  - Tareas pendientes
  - Archivos modificados sin commit
  - Último prompt del usuario

**OnSessionStart:**
- Detectar sesión interrumpida
- Preguntar: "Sesión anterior interrumpida. ¿Continuar? [Y/n]"
- Si sí → restaurar estado + inyectar resumen de lo que se estaba haciendo

---

## Harnesses de Workflow

---

### H14 — Multi-Commit Tracker

**Hook:** `PostToolExec` (write_file, edit_file)

Trackea qué archivos fueron modificados y sugiere separación en commits lógicos.

**Comportamiento:**

```
session_changes = {
  "task_1_auth": ["src/auth/jwt.go", "src/auth/middleware.go"],
  "task_2_tests": ["src/auth/jwt_test.go"],
  "task_3_config": ["config/auth.yaml"]
}
```

**En PreCommit:**
- Si el usuario hace `commit_all` pero hay cambios no relacionados → sugerir split
- Si hay archivos modificados que no pertenecen a ninguna tarea → warning

---

### H15 — Dependency Auditor

**Hook:** `PostToolExec` (write_file en go.mod, pom.xml, package.json)

Cuando el agente agrega una dependencia, valida que sea segura.

**Checks:**
- ¿Existe en el registry oficial? (no typosquatting)
- ¿Tiene mantenimiento activo? (último commit < 1 año)
- ¿Tiene vulnerabilidades conocidas? (advisory databases)
- ¿Es la versión pinneada? (no `latest` ni ranges abiertos)

**Si falla:** feedback al agente con alternativa sugerida.

---

### H16 — Git Hygiene

**Hook:** `PreCommit`

Valida que el commit siga las convenciones del proyecto.

**Checks:**
- Mensaje sigue conventional commits (`feat:`, `fix:`, `refactor:`)
- No hay archivos de debug (`.DS_Store`, `debug.log`, `*.swp`)
- No hay secrets en el diff (regex patterns)
- Branch naming correcto
- No se commitea directo a main/master

---

### H17 — Idle Timeout

**Hook:** `PreLLMCall` (timer-based)

Si el agente lleva mucho tiempo sin producir resultado útil (loop infinito, indecisión), interviene.

**Detección:**
- Más de 5 tool calls sin progreso (mismos archivos, mismos errores)
- Más de 3 minutos sin output al usuario
- Token usage alto sin commits

**Acción:** Pausar, mostrar resumen de lo que intentó, preguntar al usuario si quiere redirigir.

---

### H18 — Parallel Task Orchestrator

**Hook:** `PreToolExec`

Cuando hay tareas independientes en el task list, las ejecuta en paralelo con goroutines.

**Detección de independencia:**
- Tareas que tocan archivos distintos
- Tareas sin dependencia explícita en tasks.json

**Ejecución:**

```go
// Tareas independientes → paralelo
g, ctx := errgroup.WithContext(ctx)
for _, task := range independentTasks {
    g.Go(func() error {
        return agent.ExecuteTask(ctx, task)
    })
}
g.Wait()

// Tareas dependientes → secuencial
for _, task := range dependentTasks {
    agent.ExecuteTask(ctx, task)
}
```

---

## Harnesses Especializados (Backend)

---

### H19 — API Contract Validator

**Hook:** `PostToolExec` (write_file en controllers/handlers)

Valida que los endpoints generados cumplan con el contrato definido (OpenAPI spec si existe).

**Checks:**
- Response codes documentados
- Request/response schemas match
- Headers requeridos presentes
- Pagination patterns consistentes

---

### H20 — Database Migration Guard

**Hook:** `PreCommit` (si hay archivos de migración)

Valida migraciones de BD antes de commitear.

**Checks:**
- Migración es reversible (tiene down/rollback)
- No hay `DROP TABLE` sin confirmación explícita
- Índices para foreign keys
- No altera columnas con datos en producción sin strategy (expand/contract)

---

### H21 — Test Coverage Gate

**Hook:** `PreCommit`

Si el proyecto tiene coverage configurado, valida que no baje.

**Comportamiento:**
- Ejecutar tests con coverage
- Comparar contra baseline (almacenado en memoria)
- Si baja > 5% → bloquear commit + feedback al agente
- Si sube → registrar nuevo baseline

---

### H22 — Performance Sentinel

**Hook:** `PostToolExec` (write_file)

Detecta patrones de código que pueden causar problemas de performance.

**Patterns (configurable por lenguaje):**

| Lenguaje | Pattern | Problema |
|----------|---------|----------|
| Go | `SELECT *` en queries | Over-fetching |
| Go | Loop con query dentro | N+1 |
| Java | `@Transactional` en controller | Transaction scope demasiado amplio |
| Any | Regex sin timeout | ReDoS |
| Any | Unbounded list/pagination | Memory explosion |

---

## Prioridad de Implementación

| Prioridad | Harnesses | Razón |
|-----------|-----------|-------|
| **P0 — Fase 1** | H01 (SDD), H02 (Context), H03 (Verify) | Core del harness |
| **P1 — Fase 1.5** | H04 (Amnesia), H05 (Eliminación), H06 (Compactor), H11 (Cost) | Memoria + control |
| **P2 — Fase 2** | H12 (Skills), H07 (TDD), H09 (Diff Review), H13 (Resume) | Calidad + UX |
| **P3 — Fase 3** | H08 (Arch Lint), H10 (Security), H14 (Multi-commit), H16 (Git) | Disciplina |
| **P4 — Futuro** | H15, H17, H18, H19, H20, H21, H22 | Especializados |

---

## Configuración por Proyecto

Cada proyecto puede activar/desactivar harnesses en `.hoa/project.json`:

```json
{
  "project": "memory-management-mcp",
  "harnesses": {
    "sdd": { "enabled": true, "bypass_allowed": true },
    "verify": { "enabled": true, "levels": ["L0", "L1", "L2", "L4"] },
    "tdd": { "enabled": true },
    "arch_linter": {
      "enabled": true,
      "max_file_lines": 350,
      "layer_rules": true
    },
    "cost_guardian": { "session_max_usd": 5.0 },
    "skills": { "enabled": true, "auto_discover": true },
    "security": { "enabled": true },
    "coverage_gate": { "enabled": false }
  }
}
```

---

## Skills Dinámicas — Detalle

Las skills son el mecanismo para que el harness "aprenda" sin contaminar el contexto base.

### Ciclo de vida

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   CREAR     │────▶│  ALMACENAR   │────▶│  DESCUBRIR  │
│ (usuario o  │     │ (vector DB)  │     │ (por tarea) │
│  agente)    │     │              │     │             │
└─────────────┘     └──────────────┘     └──────┬──────┘
                                                 │
                                                 ▼
                    ┌──────────────┐     ┌─────────────┐
                    │   DESCARTAR  │◀────│  INYECTAR   │
                    │ (post-uso)   │     │ (al prompt) │
                    └──────────────┘     └─────────────┘
```

### Tipos de skills

| Tipo | Ejemplo | Quién la crea |
|------|---------|---------------|
| **Project** | Convenciones de código del proyecto | Usuario (manual) |
| **Pattern** | "Así se hace auth en este proyecto" | Usuario o agente |
| **Anti-pattern** | "Nunca usar X porque causa Y" | Harness H05 (eliminación categórica) |
| **Procedure** | "Para deployar: 1) build 2) test 3) push" | Usuario |
| **Decision** | "Elegimos Oracle sobre Postgres porque..." | Agente (post-commit) |

### Auto-generación de skills

El agente puede proponer skills después de resolver un problema complejo:

```
[agent] Resolví el problema de conexión a Oracle con connection pooling.
        ¿Quieres que guarde esto como skill para futuras sesiones?
        
        Skill propuesta: "oracle-connection-pooling"
        Contenido: "Para conexiones Oracle en este proyecto, usar HikariCP con
                    pool size = cores * 2 + 1. Configurar validationTimeout=5000
                    y connectionTimeout=30000..."

[user] > sí

[harness] Skill guardada en vector store. Se inyectará automáticamente
          cuando futuras tareas involucren conexiones a Oracle.
```

---

## Resumen Visual

```
HARNESSES POR FASE DEL AGENT LOOP:

Session Start ─── H02 (Context), H12 (Skills), H13 (Resume)
       │
Pre-LLM ───────── H02 (Disclosure), H06 (Compact), H11 (Cost), H17 (Idle)
       │
Post-LLM ──────── H11 (Cost track)
       │
Pre-Tool ──────── H01 (SDD Gate), H07 (TDD), H12 (Skill inject)
       │
Post-Tool ─────── H03 (Verify), H05 (Eliminación), H10 (Security),
                   H14 (Tracker), H15 (Deps), H19 (API), H22 (Perf)
       │
Pre-Commit ────── H08 (Arch), H09 (Diff Review), H16 (Git),
                   H20 (Migration), H21 (Coverage)
       │
Post-Commit ───── H04 (Amnesia), H14 (Reset tracker)
       │
On Error ──────── H05 (Eliminación categórica), H03 (Retry/Rollback)
       │
Session End ───── H13 (Persist state)
```
