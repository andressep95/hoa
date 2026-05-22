# Code Writing — Estrategias de Escritura de Código

[← INDEX](../INDEX.md)

---

## Concepto

Cómo el agente escribe, edita y valida código. Incluye las estrategias de aplicación de cambios, validación post-escritura, y rollback en caso de error.

---

## Modos de Escritura

| Modo | Descripción | Uso |
|------|-------------|-----|
| **create** | Archivo nuevo completo | Scaffolding, nuevos módulos |
| **str_replace** | Reemplazo exacto de string | Ediciones quirúrgicas |
| **patch/diff** | Aplicar unified diff | Cambios multi-línea complejos |
| **insert** | Insertar en línea específica | Agregar imports, métodos |
| **full_rewrite** | Reescribir archivo completo | Refactors mayores |

---

## Pipeline de Escritura

```
1. Agente genera cambio
2. Harness valida:
   - ¿El archivo está dentro del workspace? (sandbox check)
   - ¿El str_replace matchea exactamente? (evitar corrupción)
   - ¿El archivo resultante es parseable? (syntax check)
3. Aplicar cambio
4. Post-validación:
   - Lint / compile check
   - Tests afectados (si están configurados)
5. Si falla → rollback automático + feedback al agente
```

---

## Temas a Definir

- [ ] ¿Qué estrategia por defecto? (str_replace es la más segura)
- [ ] Backup automático antes de cada escritura (file snapshots)
- [ ] Detección de conflictos (archivo modificado externamente)
- [ ] Límite de tamaño de archivo para full_rewrite vs edición parcial
- [ ] Integración con LSP para validación semántica post-escritura
- [ ] Diff preview antes de aplicar (modo interactivo)
- [ ] Batch writes: agrupar cambios relacionados en una operación atómica
- [ ] Encoding detection (UTF-8, line endings)

---

## Validación Post-Escritura

| Check | Herramienta | Obligatorio |
|-------|-------------|-------------|
| Syntax válida | Tree-sitter / compilador | Sí |
| Lint pass | ESLint / Checkstyle / etc. | Configurable |
| Tests pasan | JUnit / Jest / pytest | Configurable |
| Build compila | Maven / Gradle / npm | Configurable |
