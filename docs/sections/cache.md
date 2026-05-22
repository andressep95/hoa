# Cache — Estado Efímero entre Commits

[← INDEX](../INDEX.md)

---

## Concepto

El cache almacena estado transitorio que el agente necesita **entre interacciones pero dentro del mismo commit**. Se invalida completamente al hacer commit (amnesia).

---

## Qué se cachea

| Dato | Ejemplo | TTL |
|------|---------|-----|
| Resultados de tools costosas | Output de `grep` sobre 10k archivos | Hasta cambio en archivos afectados |
| Decisiones parciales del agente | "Elegí usar Strategy pattern para X" | Hasta commit |
| File state | Hash + contenido de archivos leídos | Hasta modificación |
| Embeddings de prompts recientes | Vector del último query | Sesión |
| Tool outputs truncados | Referencia al output completo | Sesión |

---

## Invalidación

| Evento | Acción |
|--------|--------|
| `git commit` | Flush completo — todo el cache se destruye |
| File modificado | Invalidar entries que referencian ese archivo |
| Sesión nueva | Cache arranca vacío |
| Token budget excedido | Evict LRU (least recently used) |

---

## Temas a Definir

- [ ] Storage: in-memory (ConcurrentHashMap) vs Redis vs SQLite
- [ ] Tamaño máximo del cache (en tokens o en bytes)
- [ ] Política de eviction: LRU vs LFU vs TTL-based
- [ ] ¿Persistir cache a disco entre sesiones? (rompe amnesia pura)
- [ ] Cache warming: pre-cargar archivos del último diff al iniciar sesión
- [ ] Cache compartido entre sub-agentes (¿read-only para hijos?)
- [ ] Métricas: hit rate, miss rate, eviction rate
