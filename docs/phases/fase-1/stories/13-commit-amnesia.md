# Story 13 — Commit Tool con Amnesia

## Como usuario
Quiero que el agente haga commits inteligentes que incluyan intent/what/why, y que post-commit limpie el contexto de los archivos ya commiteados.

## Criterios de Aceptación

- [ ] Tool `commit` recibe: files, message, what, why
- [ ] Pre-commit: verifica que archivos compilan (L2)
- [ ] Ejecuta `git add <files> && git commit -m <message>`
- [ ] Post-commit: indexa diff en memoria (intent + what + why)
- [ ] Post-commit: archiva artefactos SDD en `history/`
- [ ] Post-commit: compacta contexto (flush mensajes sobre archivos commiteados)
- [ ] El commit message sigue conventional commits

## Archivos a Crear

```
internal/tool/commit.go              # commit tool
internal/compact/postcommit.go       # PostCommitCompaction strategy
```

## Flujo

```
commit(files, message, what, why)
  → L2 verify (compila?)
  → git add + git commit
  → memory.Store(diff, intent, what, why)
  → sdd.Archive(current → history)
  → compact.PostCommit(messages, committedFiles)
```

## Definición de Done

- "commitea los cambios" → el agente genera commit con what/why
- Post-commit el contexto se reduce (mensajes sobre esos archivos se resumen)
- `git log` muestra el commit con mensaje correcto
