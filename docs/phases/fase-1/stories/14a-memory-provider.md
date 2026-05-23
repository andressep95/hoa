# Story 14A — Memory Provider (Commit Sync con Oracle)

## Como usuario
Quiero que HOA persista mis commits en una base de datos vectorial Oracle 23ai para tener memoria semántica entre sesiones, sin consumir tokens del LLM para la extracción.

## Flujo

### Step 1: Setup (en wizard o /memory)
- El sistema pregunta si desea configurar memoria persistente
- Si acepta: pide DSN, usuario, password
- Si ya tiene API KEY de un proyecto existente, la proporciona
- Si no tiene proyecto, el sistema crea uno en la BD y genera la API KEY

### Step 2: Conexión
- Ping a la BD para confirmar acceso
- Si falla → muestra error al usuario
- Si OK → guarda credenciales en config.json (encriptadas)

### Step 3: Sincronización
- Compara hashes de commits locales (`git log --format=%H`) vs commits en BD
- Si hay commits locales no indexados → ejecuta sync automático
- Extracción es programática (Go, sin LLM) — parsea git log/diff directamente
- Configura rama principal de trabajo para saber de dónde extraer

### Step 4: Post-commit push
- Después de cada `/commit` exitoso, extrae los cambios y los inserta en la BD
- Feedback en TUI: `⎿  Memoria: 3 archivos indexados en Oracle`
- Si what/why están vacíos (commits legacy), el sistema los enriquece con el LLM (enrichment_queue)

### Step 5: Detección de inconsistencias
- Si hay una rama secundaria con cambios no registrados en la principal → avisa al usuario
- Esto es lógica determinista (git), no LLM

## Criterios de Aceptación

- [ ] `internal/memory/client.go` — cliente Oracle con godror (connect, ping, insert, query hashes)
- [ ] `internal/memory/extractor.go` — extrae commit data de git (sin LLM)
- [ ] `internal/memory/sync.go` — compara local vs BD, sincroniza faltantes
- [ ] `/memory` command funcional (status, enable, disable, sync)
- [ ] `/commit` post-commit push a Oracle con feedback
- [ ] Enrichment: si what/why vacíos, encola para LLM enrichment
- [ ] Config persiste DSN + API KEY encriptados
- [ ] Rama principal configurable

## Tablas Oracle involucradas

```
HOA.projects           — Proyecto (api_key identifica)
HOA.memory_changes     — 1 row por archivo por commit
HOA.memory_change_hunks — Hunks individuales del diff
HOA.enrichment_queue   — Cola para enriquecer commits sin what/why
```

## Extracción (sin LLM)

Basado en `extract_changes.py` del proyecto MCP, portado a Go:
- `git log -1 --format=%h,%an,%cI,%s,%b <ref>` → metadata
- `git diff-tree --no-commit-id -r --name-only <ref>` → archivos
- `git diff <parent> <ref> -- <file>` → diff por archivo
- Parse hunks (@@ blocks)
- Detectar language, kind (code/doc/config), change_type

## Arquitectura

```
/commit → git commit OK
    │
    ├─ Si memory enabled:
    │   ├─ extractor.Extract(commitHash) → []MemoryEntry
    │   ├─ client.BatchInsert(entries)
    │   ├─ Si what/why vacíos → client.EnqueueEnrichment(id)
    │   └─ Feedback: "⎿  Memoria: N archivos indexados"
    │
    └─ Si memory disabled: nada
```

## Definición de Done

- `docker compose up` levanta Oracle con schema HOA
- `/memory enable` conecta y sincroniza
- `/commit` inserta en BD y muestra feedback
- Commits legacy sin what/why se encolan para enrichment
- `/memory status` muestra: conectado, N commits indexados, N pendientes enrichment
