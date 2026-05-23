# Story 14A — Memory Provider (Commit Sync con Oracle) ✅

## Como usuario
Quiero que HOA persista mis commits en una base de datos vectorial Oracle 23ai para tener memoria semántica entre sesiones, sin consumir tokens del LLM para la extracción.

## Estado: COMPLETADA

## Criterios de Aceptación

- [x] `internal/memory/client.go` — cliente Oracle con go-ora (connect, ping, insert, query hashes)
- [x] `internal/memory/extractor.go` — extrae commit data de git (sin LLM)
- [x] `internal/memory/sync.go` — compara local vs BD, sincroniza faltantes (concurrente)
- [x] `internal/memory/enrichment.go` — processor async que drena cola con LLM
- [x] `internal/memory/search.go` — búsqueda semántica con VECTOR_DISTANCE
- [x] `internal/memory/working.go` — working context desde git diff (memoria de sesión)
- [x] `internal/memory/feedback.go` — feedback rules (correcciones del usuario)
- [x] `/memory` command funcional (status, enable, disable, sync) con wizard interactivo
- [x] `/commit` post-commit push a Oracle con feedback
- [x] `/feedback` command para guardar reglas de feedback
- [x] Enrichment: si what/why vacíos, encola y procesa con LLM en paralelo
- [x] Config persiste DSN + API KEY (wizard interactivo)
- [x] Metadata trimming (ignora binarios, lockfiles, generados)
- [x] Auto-creación de proyecto si no tiene API key
- [x] Embeddings generados automáticamente por Oracle (ONNX trigger)
- [x] Prompt caching habilitado (system + automatic caching)
- [x] Contexto inyectado al LLM: working_changes + feedback_rules + project_memory

## Tablas Oracle implementadas

```
HOA.projects           — Proyecto (api_key identifica)
HOA.memory_changes     — 1 row por archivo por commit (con embedding auto)
HOA.memory_change_hunks — Hunks individuales del diff
HOA.enrichment_queue   — Cola para enriquecer commits sin what/why
HOA.feedback_rules     — Correcciones del usuario (con embedding auto)
```

## Archivos implementados

```
internal/memory/
├── client.go       — conexión Oracle, BatchInsert, CreateProject, CountIndexed
├── extractor.go    — Extract() porta extract_changes.py a Go
├── sync.go         — Sync() + SyncOne() con enrichment concurrente
├── enrichment.go   — EnrichmentProcessor (goroutine async, drena cola)
├── search.go       — Search() con VECTOR_DISTANCE + FormatContext()
├── working.go      — WorkingContext() desde git diff
└── feedback.go     — SaveFeedback, SearchFeedback, FormatFeedback

internal/command/
├── memory.go       — /memory (status, enable, disable, sync) con wizard
└── feedback.go     — /feedback (save, list)

docker/
├── oracle/init/01-schema.sql      — todas las tablas + feedback_rules
├── oracle/init/02-embedding-model.sql — ONNX model + triggers auto-embedding
├── oracle/models/                 — all_MiniLM_L12_v2.onnx (Oracle pre-converted)
├── setup-model.sh                 — descarga modelo ONNX
└── docker-compose.yml             — monta /opt/oracle/models

internal/provider/anthropic.go     — prompt caching (system + automatic)
internal/agent/agent.go            — inyección de WorkingContext + MemorySearch
cmd/hoa/main.go                    — wiring de memoria + feedback + working context
```

## Arquitectura final

```
prompt usuario
    │
    ├─ 1. WorkingContext() → git diff (archivos en progreso, auto-limpia post-commit)
    ├─ 2. SearchFeedback() → reglas relevantes (VECTOR_DISTANCE, max 3)
    ├─ 3. Search() → commits relevantes (VECTOR_DISTANCE, max 5, score < 0.7)
    ├─ 4. Prompt del usuario
    │
    └─ Todo enviado al LLM con prompt caching (10% en turns 2+)

/commit exitoso
    │
    ├─ Extract(HEAD) → entries por archivo
    ├─ BatchInsert → Oracle (trigger genera embedding automáticamente)
    ├─ NeedsEnrichment? → EnrichmentProcessor.Trigger() (goroutine async + LLM)
    └─ Feedback: "⎿  Memoria: N archivo(s) indexados en Oracle"
```

## Definición de Done ✅

- [x] `docker compose up` levanta Oracle con schema HOA + modelo ONNX
- [x] `/memory enable` → wizard interactivo (DSN + API key o crear proyecto)
- [x] `/memory sync` → sincroniza historial con spinner (async)
- [x] `/commit` inserta en BD y muestra feedback
- [x] Commits legacy sin what/why se encolan y enriquecen con LLM
- [x] `/memory status` muestra: conectado, N commits indexados, N pendientes
- [x] Embeddings generados automáticamente (Oracle ONNX, 384 dims)
- [x] Búsqueda semántica inyecta contexto relevante al LLM
- [x] Prompt caching reduce costo 90% en tokens repetidos
- [x] Feedback rules persisten y se inyectan cuando son relevantes
