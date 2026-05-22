# Skills — Marco de Trabajo Reutilizable

[← INDEX](../../INDEX.md)

---

## Concepto

```
Memory = QUÉ existe y POR QUÉ (contexto histórico por commit)
Skill  = CÓMO hacerlo (procedimiento, patrones, reglas)
```

Un skill es un marco de trabajo que el agente aplica cuando enfrenta una tarea específica. Post-amnesia, el agente reconstruye contexto con memory (historia) y skills (procedimiento).

---

## Skill Template (formato canónico)

Todos los skills siguen este formato. El `skill-creator` genera este template.

```yaml
# Skill Definition — v1
id: "auth-jwt-implementation"
name: "JWT Authentication"
version: 1
created: "2026-05-21T23:00:00-04:00"
updated: "2026-05-21T23:00:00-04:00"
author: "andres"

# Cuándo aplica este skill
trigger:
  keywords: ["auth", "authentication", "login", "jwt", "token"]
  file_patterns: ["**/auth/**", "**/middleware/**"]
  intent: "Implementar o modificar autenticación"

# Contexto que el agente necesita saber
context:
  description: |
    Este proyecto usa JWT con RS256 para autenticación stateless.
    Los refresh tokens se almacenan en cookies httpOnly.
  dependencies:
    - "github.com/golang-jwt/jwt/v5"
    - "golang.org/x/crypto/bcrypt"
  related_files:
    - "internal/auth/jwt.go"
    - "internal/middleware/auth.go"
    - "internal/config/keys.go"

# Pasos que el agente debe seguir
procedure:
  - step: "Verificar que las keys RSA existen en config"
    check: "internal/config/keys.go debe exportar PublicKey y PrivateKey"
  - step: "Implementar en internal/auth/"
    rules:
      - "Tokens de acceso: 15 min TTL"
      - "Refresh tokens: 7 días TTL, rotación en cada uso"
      - "Claims mínimos: sub, exp, iat, roles"
  - step: "Middleware en internal/middleware/auth.go"
    rules:
      - "Extraer token de header Authorization: Bearer <token>"
      - "Validar firma con PublicKey"
      - "Inyectar claims en context"
  - step: "Errores con el patrón del proyecto"
    rules:
      - "401 para token inválido/expirado"
      - "403 para permisos insuficientes"
      - "Usar internal/errors/ para wrapping"

# Reglas que el harness valida (deterministas)
invariants:
  - "Nunca almacenar tokens de acceso en base de datos"
  - "Refresh tokens hasheados con bcrypt antes de persistir"
  - "Keys RSA nunca en el código — siempre desde config/env"
  - "Tests obligatorios: token válido, expirado, malformado, sin permisos"

# Ejemplos de código (opcional, para guiar al modelo)
examples:
  - file: "internal/auth/jwt.go"
    snippet: |
      func GenerateAccessToken(userID string, roles []string) (string, error) {
          claims := jwt.MapClaims{
              "sub":   userID,
              "roles": roles,
              "exp":   time.Now().Add(15 * time.Minute).Unix(),
              "iat":   time.Now().Unix(),
          }
          token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
          return token.SignedString(privateKey)
      }

# Tags para búsqueda vectorial
tags: ["auth", "jwt", "security", "middleware", "tokens"]
```

---

## Skill Discovery (Chequeo Inverso Pre-LLM)

El discovery de skills ocurre **antes** de enviar el prompt al modelo. Es una operación de base de datos pura — cero tokens consumidos.

### Principio

```
El prompt del usuario se compara contra los campos semánticos de los skills
ANTES de llamar al LLM. Solo los skills que matchean se inyectan al contexto.
Si no matchea ninguno → el modelo trabaja sin skills (ahorra tokens).
```

### Flujo

```
Usuario escribe: "agrega validación al endpoint de login"
    │
    ▼
┌─────────────────────────────────────────────────────┐
│  SKILL DISCOVERY (pre-LLM, cero tokens)            │
│                                                     │
│  1. Extraer keywords del prompt:                    │
│     ["validación", "endpoint", "login"]             │
│                                                     │
│  2. Query a BD con LIKE sobre campos semánticos:    │
│     - trigger.keywords                              │
│     - trigger.intent                                │
│     - tags                                          │
│                                                     │
│  3. Filtrar por threshold de relevancia             │
│                                                     │
│  4. Resultado: 0..N skills matcheados               │
└─────────────────────────────────────────────────────┘
    │
    ├── 0 skills → prompt va al LLM sin skills (ahorro de tokens)
    │
    ├── 1-2 skills → se inyectan completos al contexto
    │
    └── 3+ skills → se inyectan solo context.description + procedure
        (sin examples, para no explotar token budget)
    │
    ▼
LLM recibe solo lo relevante
```

### Query de Discovery (Oracle 23ai)

```sql
-- Paso 1: LIKE rápido contra keywords y tags (filtro grueso, barato)
SELECT id, name, content,
       SCORE(1) AS keyword_score
FROM skills
WHERE project_id = :project_id
  AND (
    CONTAINS(trigger_keywords, :user_keywords, 1) > 0
    OR LOWER(tags) LIKE '%' || :term1 || '%'
    OR LOWER(tags) LIKE '%' || :term2 || '%'
    OR LOWER(trigger_intent) LIKE '%' || :term1 || '%'
  )
ORDER BY keyword_score DESC
FETCH FIRST 5 ROWS ONLY;
```

```sql
-- Paso 2 (opcional): si el LIKE no da resultados, fallback a vector similarity
-- Solo se usa si el paso 1 devuelve 0 rows — sigue siendo pre-LLM
SELECT id, name, content,
       VECTOR_DISTANCE(embedding,
           DBMS_VECTOR_CHAIN.UTL_TO_EMBEDDING(:user_prompt,
               JSON('{"provider":"database","model":"ALL_MINILM_L12_V2"}')),
           COSINE) AS score
FROM skills
WHERE project_id = :project_id
ORDER BY score ASC
FETCH FIRST 3 ROWS ONLY;
```

### Campos Semánticos Indexados

| Campo | Tipo de Match | Ejemplo |
|-------|---------------|---------|
| `trigger.keywords` | LIKE / CONTAINS | `["auth", "login", "jwt"]` |
| `trigger.intent` | LIKE | `"Implementar o modificar autenticación"` |
| `tags` | LIKE | `["auth", "security", "middleware"]` |
| `embedding` | Vector similarity (fallback) | Embedding del skill completo |

### Estrategia de Dos Pasos

| Paso | Método | Costo | Cuándo |
|------|--------|-------|--------|
| 1 | LIKE / CONTAINS sobre keywords+tags | ~1ms, cero tokens | Siempre |
| 2 | Vector similarity | ~100ms, cero tokens (embedding en BD) | Solo si paso 1 devuelve 0 |

**Nunca se llama al LLM para discovery.** El embedding del paso 2 lo genera Oracle internamente con `DBMS_VECTOR_CHAIN` — no consume tokens del provider.

### Inyección al Contexto

```go
// internal/harness/skills.go
func (s *SkillDiscovery) Discover(ctx context.Context, userPrompt string) []Skill {
    // Paso 1: keyword match (LIKE)
    skills := s.db.MatchByKeywords(ctx, extractKeywords(userPrompt))
    if len(skills) > 0 {
        return skills
    }
    // Paso 2: vector fallback (solo si paso 1 falla)
    return s.db.MatchByVector(ctx, userPrompt, 3)
}

// Se inyecta ANTES del Send() al provider
func (a *Agent) buildContext(userPrompt string) {
    skills := a.skillDiscovery.Discover(ctx, userPrompt)
    if len(skills) == 0 {
        return // nada que inyectar, ahorra tokens
    }
    // Inyectar solo lo necesario según cantidad
    a.injectSkills(skills)
}
```

### Token Budget por Skills

| Skills matcheados | Qué se inyecta | Tokens aprox |
|-------------------|----------------|--------------|
| 0 | Nada | 0 |
| 1 | Skill completo (context + procedure + invariants + examples) | ~500-1000 |
| 2 | Skills completos sin examples | ~400-800 |
| 3+ | Solo context.description + invariants | ~200-400 |

El harness nunca inyecta más de N tokens de skills (configurable en `config.harness.skillTokenBudget`).

---

## Skill Creator

### Comando

```
/skill create
```

```
⚙️  Skill Creator

Nombre del skill: _
  ▸ (escribir nombre)

Trigger keywords (separados por coma): _

Intent (cuándo aplica): _

[Se abre editor con el template YAML pre-llenado]
[El usuario completa procedure, invariants, examples]
[Al guardar → se indexa en BD vectorial]

✅ Skill "auth-jwt-implementation" creado y indexado
```

### Otras formas de crear skills

| Método | Cómo |
|--------|------|
| `/skill create` | Wizard interactivo + editor |
| `/skill create --from-file patterns.md` | Parsea un markdown existente al formato |
| `/skill extract` | El agente analiza commits recientes y propone skills |
| Automático | El harness detecta patrones repetidos → sugiere crear skill |

### Gestión

```
/skill list                    # Lista skills del proyecto
/skill show auth-jwt           # Muestra detalle
/skill edit auth-jwt           # Abre en editor
/skill delete auth-jwt         # Elimina
/skill search "manejo errores" # Búsqueda semántica
```

---

## Almacenamiento

```sql
-- Oracle 23ai
CREATE TABLE skills (
    id            VARCHAR2(128) PRIMARY KEY,
    project_id    VARCHAR2(64) NOT NULL,
    name          VARCHAR2(256) NOT NULL,
    version       NUMBER DEFAULT 1,
    content       CLOB NOT NULL,           -- YAML completo
    embedding     VECTOR(384, FLOAT32),    -- Para búsqueda semántica
    created_at    TIMESTAMP DEFAULT SYSTIMESTAMP,
    updated_at    TIMESTAMP DEFAULT SYSTIMESTAMP
);

-- Embedding generado con DBMS_VECTOR_CHAIN sobre:
-- trigger.keywords + trigger.intent + context.description + tags
```

---

## Relación con Memory

| Aspecto | Memory | Skills |
|---------|--------|--------|
| Qué almacena | Historia de commits (what/why/diff) | Procedimientos y reglas |
| Cuándo se crea | Automático en cada commit | Manual o por detección de patrones |
| Cuándo se usa | Siempre (preamble + recall) | Cuando la tarea matchea un trigger |
| Muta | No (inmutable, histórico) | Sí (se versiona y actualiza) |
| Granularidad | Por commit | Por concepto/dominio |

---

## Context Injection por Task (Plan → Execute con Breadcrumbs)

### Principio

```
El planning model (inteligente) genera el plan con "context hints" por cada task.
El execution model (rápido) usa esos hints para consultar la BD solo lo que necesita
para el paso que está ejecutando. Cero contexto innecesario.
```

### Flujo Completo

```
┌─────────────────────────────────────────────────────────────┐
│  PLANNING MODEL (opus/o3)                                   │
│                                                             │
│  Recibe:                                                    │
│    - Prompt del usuario                                     │
│    - Skill matcheado (procedure + invariants)               │
│    - Memory relevante (contexto histórico)                  │
│                                                             │
│  Genera: tasks.json con context_hints por task              │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│  tasks.json                                                 │
│                                                             │
│  [                                                          │
│    {                                                        │
│      "id": 1,                                               │
│      "description": "Crear middleware de auth",             │
│      "file": "internal/middleware/auth.go",                 │
│      "context_hints": {                                     │
│        "query": "middleware pattern auth validation",       │
│        "skill_section": "procedure[2]",                     │
│        "memory_query": "middleware existing patterns"       │
│      }                                                      │
│    },                                                       │
│    {                                                        │
│      "id": 2,                                               │
│      "description": "Agregar endpoint POST /login",        │
│      "file": "internal/handler/auth.go",                   │
│      "context_hints": {                                     │
│        "query": "handler login endpoint structure",         │
│        "skill_section": "procedure[1]",                     │
│        "memory_query": "handler patterns http responses"    │
│      }                                                      │
│    }                                                        │
│  ]                                                          │
└─────────────────────────────────────────────────────────────┘
         │
         ▼ (por cada task)
┌─────────────────────────────────────────────────────────────┐
│  PRE-EXECUTION LOOKUP (cero tokens LLM)                     │
│                                                             │
│  Para task[1]:                                              │
│    1. skill_section → extraer procedure[2] del skill        │
│    2. memory_query → buscar en BD vectorial                 │
│       "middleware existing patterns"                        │
│       → "el middleware actual usa chi.Router,               │
│          patrón en internal/middleware/logger.go"           │
│    3. Inyectar SOLO eso al execution model                  │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│  EXECUTION MODEL (sonnet/gpt-4o)                            │
│                                                             │
│  Recibe (contexto mínimo por task):                         │
│    - Task description                                       │
│    - Sección específica del skill (no todo el skill)        │
│    - Memory puntual (no toda la historia)                   │
│    - Invariants relevantes                                  │
│                                                             │
│  Ejecuta: escribe código → verify loop                      │
└─────────────────────────────────────────────────────────────┘
```

### context_hints — El Contrato

El planning model genera estos campos por cada task:

```yaml
context_hints:
  # Query para buscar en BD vectorial (memory)
  memory_query: "string de búsqueda semántica"

  # Sección específica del skill a inyectar
  skill_section: "procedure[N]" | "invariants" | "examples[N]"

  # Query adicional para embeddings de código existente
  code_query: "string para buscar en embeddings de archivos"

  # Archivos que el executor debe leer antes de escribir
  read_first: ["internal/middleware/logger.go"]
```

### Token Budget por Modelo

| Modelo | Recibe | Tokens aprox |
|--------|--------|--------------|
| Planning (opus) | Prompt + skill completo + memory amplia | ~3000-5000 |
| Execution (sonnet) | Task + skill_section + memory puntual | ~500-1000 |

El executor trabaja con **contexto quirúrgico**: solo lo que necesita para ese paso específico. El planning model ya hizo el trabajo pesado de entender el todo.

### Implementación

```go
// internal/agent/router.go
func (r *Router) ExecuteTask(ctx context.Context, task Task) error {
    // 1. Resolver context_hints contra BD (cero tokens)
    hints := task.ContextHints
    
    var injectedContext strings.Builder
    
    // Skill section específica
    if hints.SkillSection != "" {
        section := r.skills.ExtractSection(task.SkillID, hints.SkillSection)
        injectedContext.WriteString(section)
    }
    
    // Memory puntual
    if hints.MemoryQuery != "" {
        memories := r.memory.Query(ctx, hints.MemoryQuery, 3)
        for _, m := range memories {
            injectedContext.WriteString(m.Summary())
        }
    }
    
    // Code embeddings
    if hints.CodeQuery != "" {
        snippets := r.memory.QueryCode(ctx, hints.CodeQuery, 2)
        for _, s := range snippets {
            injectedContext.WriteString(s.Snippet)
        }
    }
    
    // 2. Enviar al execution model con contexto mínimo
    return r.executor.Send(ctx, task.Description, injectedContext.String())
}
```

### Resultado

```
Planning model:  1 llamada costosa con contexto completo → genera plan con hints
Execution model: N llamadas baratas con contexto mínimo por task

Costo total = 1×opus + N×sonnet (con contexto reducido)
vs sin hints = 1×opus + N×sonnet (con contexto completo repetido)

Ahorro: ~60-70% de tokens en ejecución
```

### Post-Amnesia Reconstruction

```
┌─────────────────────────────────────────────────────┐
│  Agente post-commit (contexto limpio)               │
│                                                     │
│  Nueva tarea llega                                  │
│       │                                             │
│       ├──▶ Memory: "¿qué existe relacionado?"       │
│       │    → "auth está en internal/auth/,          │
│       │       se migró de sessions a JWT en v2.1"   │
│       │                                             │
│       ├──▶ Skill: "¿cómo se hace esto aquí?"       │
│       │    → procedure + invariants + examples      │
│       │                                             │
│       ▼                                             │
│  Agente tiene contexto completo para actuar         │
└─────────────────────────────────────────────────────┘
```

---

## Roadmap

| Fase | Capacidad |
|------|-----------|
| Fase 1 | No implementado — solo documentado |
| Fase 2 | `/skill create` manual + almacenamiento + búsqueda |
| Fase 3 | Detección automática de patrones → sugerencia de skills |
| Fase 4 | Skills compartidos entre proyectos (skill marketplace) |
