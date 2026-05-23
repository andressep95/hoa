# Memory Management MCP — Harness Engineering Bible

> Documento de referencia para evolucionar el sistema de memoria semántica.
> Organizado por área funcional. Cada mejora tiene prioridad, estado y contexto de implementación.
>
> **Ecuación fundamental:** `Agente = Modelo + Harness`
> El MCP server ES el harness — impone estructura determinista sobre la inteligencia probabilística del modelo.

---

## Principios de Diseño

| Principio | Aplicación en este sistema |
|-----------|---------------------------|
| **Progreso incremental** | 1 commit = 1 feature. Estado limpio al final de cada sesión |
| **Memoria persistente** | Oracle 23ai como Engram — decisiones y contexto sobreviven entre sesiones |
| **Verificación autónoma** | El agente valida su trabajo via `queryMemory` antes de implementar |
| **Progressive Context Disclosure** | Solo inyectar skills/memory relevantes, no todo el catálogo |
| **Eliminación categórica** | Si un error se repite, se arregla el harness (skill/linter/hook), no el código |
| **Código predecible** | Skills estandarizan formato, hooks imponen calidad |

---

## Estado Actual vs. Visión

| Aspecto | Implementado Hoy | Visión Objetivo |
|---------|-------------------|-----------------|
| Búsqueda | Vectorial pura (COSINE) | Híbrida (vector + filtros relacionales) |
| Ingesta | 1 registro por archivo por commit | Chunking inteligente + metadata trimming |
| Pruning | Ninguno — crece infinitamente | Consolidación temporal (corto/largo plazo) |
| Contexto Git | Solo `branch` y `author` | Lineage, divergencia, impacto arquitectónico |
| Setup | 3 capas: blueprint → confirmSetup → hook runtime gate | 3 capas: blueprint → confirmSetup → hook runtime gate ✅ |
| Disciplina | Skills manuales | Linters bespoke + invariantes arquitectónicos |
| Verificación | Ninguna | Agente de verificación + TDD estricto |

---

## Mejoras por Categoría

---

### 🔧 SETUP INICIAL

Mejoras al proceso de `setupProject` y configuración del entorno del agente.

| # | Mejora | Impacto | Esfuerzo | Estado |
|---|--------|---------|----------|--------|
| S1 | **3-Layer Validated Setup** — `setupProject` (blueprint) → `confirmSetup` (server valida) → hook runtime (gate) | 🔴 Crítico | Medio | ✅ Implementado |
| S2 | **Auto-repair Loop** — Hook runtime detecta drift, inyecta fixes, agente repara | 🟠 Alto | Bajo | ✅ Implementado |
| S3 | **Linters Bespoke** — Invariantes arquitectónicos verificados por código (no por deseos) | 🟡 Medio | Medio | ❌ |

#### S1. 3-Layer Validated Setup ✅

**Implementado.** `setupProject` genera blueprint sin marcar completed. `confirmSetup` valida hash SHA-256 del config y marca `setup_completed_at`. Hook `validate-setup.sh` ejecuta en cada sesión como runtime gate — si detecta drift, inyecta fixes al contexto del agente.

#### S2. Auto-repair Loop ✅

**Implementado.** Hook `session-start.sh` ejecuta `validate-setup.sh` en cada inicio de sesión. Si detecta problemas (exit 1), inyecta la lista de fixes como contexto obligatorio al agente. Valida: config, git hooks, permisos de scripts, symlinks, skills, memory state.

#### S3. Linters Bespoke

**Concepto:** Imponer calidad mediante código. Si el agente viola la arquitectura, el linter dispara un error que actúa como prompt de corrección inmediata.

**Ejemplos:**
- Prohibir archivos > 350 líneas (fuerza modularización)
- Verificar que todo handler tenga su test
- Detectar prop-drilling o dependencias circulares
- Validar que commits sigan el formato del skill

---

### 📥 INGESTA DE COMMITS

Mejoras al pipeline de extracción, transformación y carga de commits en la memoria.

| # | Mejora | Impacto | Esfuerzo | Estado |
|---|--------|---------|----------|--------|
| I0 | **Enriquecimiento de Commits Legacy** — LLM infiere intent/what/why | 🔴 Bloqueante | Medio | ✅ Implementado |
| I1 | **Metadata Trimming** — Excluir binarios, lockfiles, archivos generados | 🟠 Alto | Bajo | ⚠️ Parcial |
| I2 | **Chunking Inteligente por Tipo** — IaC por bloques, deps con flag, código por función | 🟠 Alto | Alto | ❌ |
| I3 | **Impacto Arquitectónico** — Tags automáticos para IaC/deps/security changes | 🟡 Medio | Medio | ❌ |
| I4 | **Compactación Head/Tail** — Diffs largos: conservar inicio+final, persistir resto | 🟡 Medio | Bajo | ❌ |

#### I0. Enriquecimiento de Commits Legacy ✅

**Implementado.** Cola persistente `enrichment_queue` + OpenAI gpt-4o-mini. Detecta campos pobres en ingesta, encola, processor async drena y actualiza intent/what/why + re-genera embedding.

#### I1. Metadata Trimming

**Problema:** `go.sum`, `package-lock.json`, binarios generan embeddings inútiles.

**Solución:** Lista de exclusión en `extract_changes.py`:
```python
IGNORE_PATTERNS = {
    "go.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
    "Cargo.lock", "*.min.js", "*.min.css", "*.map",
    "*.pb.go", "*.generated.*", "dist/", "build/", "node_modules/",
}
```

#### I2. Chunking Inteligente por Tipo

| Tipo de Archivo | Estrategia |
|-----------------|-----------|
| IaC (`.tf`, `.cdk.ts`) | 1 chunk por `resource`/`module`/`Construct` |
| Dependencias (`pom.xml`, `go.mod`) | 1 chunk por cambio de versión, flag `ARCH_CHANGE` |
| Código fuente | 1 chunk por función/método modificado (hunks + symbol) |
| Documentación | 1 chunk por sección (heading level 2) |
| Config (`.yaml`, `.env`) | 1 chunk por bloque lógico (top-level key) |

#### I3. Impacto Arquitectónico

Enriquecer metadatos automáticamente:
- Archivos IaC → tag `INFRA_CHANGE` + descripción
- Dependencias → tag `ARCH_CHANGE` + delta de versiones
- Seguridad → tag `SEC_CHANGE`

#### I4. Compactación Head/Tail

Para diffs > 2000 chars: conservar primeros 500 + últimos 500 tokens en el embedding text, persistir el diff completo en `raw_diff` para acceso bajo demanda.

---

### 🔍 BÚSQUEDA Y RECUPERACIÓN

Mejoras a cómo el agente consulta la memoria.

| # | Mejora | Impacto | Esfuerzo | Estado |
|---|--------|---------|----------|--------|
| B1 | **Búsqueda Híbrida** — Vector + filtros relacionales (fecha, autor, rama) | 🔴 Crítico | Medio | ❌ |
| B2 | **Commit Lineage** — N commits antes/después para contexto de flujo | 🟠 Alto | Bajo | ❌ |
| B3 | **Active Branch Context** — Rama actual, upstream, divergencia | 🟡 Medio | Bajo | ❌ |
| B4 | **Grafos de Dependencia** — Vincular commits que tocan archivos relacionados | 🟡 Medio | Alto | ❌ |
| B5 | **Progressive Context Disclosure** — Solo inyectar skills relevantes a la tarea | 🟡 Medio | Medio | ❌ |

#### B1. Búsqueda Híbrida

**SQL propuesto:**
```sql
SELECT id, commit_hash, file_path, what, why,
       VECTOR_DISTANCE(embedding, ?, COSINE) AS score
FROM memory_changes
WHERE project_id = ?
  AND (:branch IS NULL OR branch = :branch)
  AND (:author IS NULL OR author = :author)
  AND (:since IS NULL OR created_at >= :since)
  AND (:fileLike IS NULL OR file_path LIKE :fileLike)
ORDER BY score ASC
FETCH FIRST ? ROWS ONLY
```

#### B2. Commit Lineage

Tool `getCommitLineage(commitHash, projectId, window)` — retorna N commits antes/después para entender el "flujo mental" de una refactorización.

#### B5. Progressive Context Disclosure

No inyectar todas las skills al inicio. Usar el resultado de `querySkills` para inyectar solo las relevantes a la tarea actual. Reduce context rot.

---

### 🧹 MANTENIMIENTO Y PRUNING

Mejoras para mantener la base de datos limpia y las búsquedas relevantes.

| # | Mejora | Impacto | Esfuerzo | Estado |
|---|--------|---------|----------|--------|
| M1 | **Memory Pruning & Consolidación** — Corto/largo plazo + squash rule | 🔴 Crítico | Alto | ❌ |
| M2 | **Garbage Collection Days** — Análisis semanal de fallos sistemáticos | 🟡 Medio | Bajo | ❌ |
| M3 | **Re-indexación** — Migration tool si cambia el modelo de embeddings | 🟢 Bajo | Medio | ❌ |

#### M1. Memory Pruning & Consolidación

| Nivel | Retención | Datos | Embedding |
|-------|-----------|-------|-----------|
| Memoria de Trabajo | Últimos 30 días | Diff completo + hunks | Vector de `intent+what+why+filePath` |
| Memoria a Largo Plazo | > 30 días | Solo mensaje + lista de archivos | Vector de `commitMsg + fileList` |

**Reglas:**
- Commits de ramas `feature/*` ya mergeadas → purgar tras 14 días, conservar solo squash en `main`
- Job diario que ejecute la consolidación

#### M2. Garbage Collection Days

**Concepto:** Política semanal donde se analizan fallos repetitivos. Si un agente comete un error recurrente, no se arregla el código — se arregla el harness (skill, linter, hook) para que ese error nunca vuelva a ocurrir.

---

### 🛡️ DISCIPLINA Y VERIFICACIÓN

Mejoras para garantizar calidad y seguridad en la ejecución del agente.

| # | Mejora | Impacto | Esfuerzo | Estado |
|---|--------|---------|----------|--------|
| D1 | **Strict TDD Pipeline** — Red → Green → Triangulate → Refactor obligatorio | 🟠 Alto | Medio | ❌ |
| D2 | **Agente de Verificación** — Valida implementación contra spec/design | 🟠 Alto | Alto | ❌ |
| D3 | **Clasificación Dinámica de Permisos** — Read-only / Workspace / Full-access | 🟡 Medio | Medio | ❌ |
| D4 | **SDD-IA Phases** — Proposal → Spec → Design → Task → Apply → Verify | 🟡 Medio | Alto | ❌ |
| D5 | **Skills Indexing Digest** — Resúmenes accionables en vez de docs masivas | 🟡 Medio | Bajo | ❌ |

#### D1. Strict TDD Pipeline

Skill que imponga el ciclo:
1. **Red** — Test fallido basado en spec
2. **Green** — Código mínimo para pasar
3. **Triangulate** — Al menos 2 formas de romper el código (casos borde)
4. **Refactor** — Limpiar sin romper tests

#### D2. Agente de Verificación

Sub-agente independiente que:
- Ejecuta tests
- Inspecciona logs
- Compara implementación contra artefactos de Spec/Design
- Detecta "victoria prematura" (el agente dice que terminó pero no)

#### D3. Clasificación Dinámica de Permisos

| Nivel | Alcance | Ejemplos |
|-------|---------|----------|
| Read-only | Lectura de estado | `ls`, `grep`, `cat`, `git log` |
| Workspace-write | Escritura al proyecto | `git commit`, `npm install` |
| Full-access | Operaciones destructivas | `rm -rf`, `sudo`, `git push --force` |

Full-access requiere **Interactive Approval Gate** (pausa para aprobación humana).

#### D4. SDD-IA Phases

Fases obligatorias que impiden al agente saltar directo al código:

```
Proposal → Spec → Design → Task → Apply → Verify
```

Cada fase produce un artefacto. El agente no puede avanzar sin completar la fase anterior.

#### D5. Skills Indexing Digest

En vez de inyectar documentación masiva, el harness indexa y "digiere" capacidades reutilizables, proporcionando resúmenes accionables que no asfixian el contexto.

---

## Roadmap de Implementación

| Fase | Mejoras | Semanas Est. |
|------|---------|--------------|
| **Fase 1 — Quick Wins** | I1 (Trimming), B2 (Lineage), B3 (Branch Context), S1 (Validate Setup) | 1-2 |
| **Fase 2 — Búsqueda** | B1 (Híbrida), I3 (Impacto Arquitectónico) | 2-3 |
| **Fase 3 — Disciplina** | D1 (TDD), D5 (Skills Digest), S3 (Linters) | 2-3 |
| **Fase 4 — Ingesta** | I2 (Chunking), B4 (Grafos) | 3-4 |
| **Fase 5 — Mantenimiento** | M1 (Pruning) — requiere datos históricos para validar | 2-3 |
| **Fase 6 — Autonomía** | D2 (Verificación), D3 (Permisos), D4 (SDD-IA) | 4-6 |

---

## Notas Técnicas

- **Modelo de embeddings:** `multilingual-e5-small` (384 dims) — alineado entre ingesta y búsqueda ✅
- **Índices HNSW:** Target accuracy 95%
- **Batch size:** 25 en Python script, 50 en Java handler (chunks internos)
- **Oracle es el store autoritativo**
- **Limitación:** No hay re-indexación si cambia el modelo de embeddings
- **Enrichment LLM:** Intercambiable via interface (`OpenAI` → `Ollama` → `Claude`)
