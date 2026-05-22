# Memoria — Vectorial + Amnesia + Progressive Context

[← INDEX](../INDEX.md)

---

## Concepto

La memoria es el sistema que permite al agente "recordar" decisiones pasadas sin arrastrar contexto infinito. Combina búsqueda vectorial semántica con una política de amnesia controlada por commits de git.

---

## Estrategias de Memoria

| Estrategia | Retención | Trigger de limpieza | Datos |
|------------|-----------|---------------------|-------|
| **Memoria de trabajo** | Sesión actual | Fin de sesión | Conversación completa |
| **Cache efímero** | Entre interacciones (intra-commit) | Siguiente commit | Resultados de tools, decisiones parciales |
| **Memoria corto plazo** | Últimos 30 días | Consolidación automática | Diffs + intent + what + why |
| **Memoria largo plazo** | Indefinida | Nunca (solo squash) | Commit messages + file lists + decisiones arquitectónicas |

---

## Amnesia por Commit

```
[interacción 1] → cache
[interacción 2] → cache
[interacción N] → cache
       │
       ▼ git commit
[FLUSH] → indexar en vector store → limpiar cache
```

Post-commit:
1. Se indexa el diff + metadata en el vector store
2. Se invalida el cache efímero
3. La próxima sesión arranca "limpia" — solo accede a memoria via búsqueda semántica

---

## Progressive Context Disclosure

No inyectar toda la memoria al inicio. El flujo es:

1. Usuario envía prompt
2. Harness genera embedding del prompt
3. Busca top-K memorias relevantes (vector search)
4. Inyecta solo esas memorias como contexto
5. El agente trabaja con contexto mínimo y relevante

---

## Temas a Definir

- [ ] ¿Módulo interno o MCP externo? (latencia vs desacoplamiento)
- [ ] Token budget para memoria inyectada (¿máx 20% del context window?)
- [ ] Consolidación temporal (30 días → squash a largo plazo)
- [ ] Re-indexación si cambia el modelo de embeddings
- [ ] Memoria compartida entre sub-agentes (¿sí o no?)
- [ ] Enrichment de commits pobres (LLM infiere intent/what/why)
- [ ] Filtros híbridos: vector + branch + author + date range

---

## Base Existente

El `memory-management-mcp` ya implementa:
- ✅ Oracle 23ai con columnas VECTOR (384 dims)
- ✅ DJL + multilingual-e5-small (embeddings locales)
- ✅ queryMemory / queryCode / queryDocs
- ✅ Ingesta batch de commits
- ✅ Enrichment async con LLM
- ✅ Setup validado con SHA-256
