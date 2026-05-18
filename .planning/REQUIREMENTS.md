# Requirements: mlx-go-sdk Pre-Publish Audit

**Defined:** 2026-05-18
**Core Value:** Надёжный, хорошо задокументированный Go SDK, позволяющий разработчикам уверенно автоматизировать Multilogin X браузерные профили.

## v1 Requirements

Requirements for initial private GitHub release.

### Audit — Code Quality

- [ ] **AUDIT-01**: `go test ./...` проходит без ошибок (исключая e2e тесты, требующие `MLX_RUN_E2E=1`)
- [ ] **AUDIT-02**: `go vet ./...` не выдаёт предупреждений
- [ ] **AUDIT-03**: README.md корректен — все описанные API существуют и работают
- [ ] **AUDIT-04**: Документация в `docs/` соответствует реальному коду (CLI reference, workflows, batch helpers, etc.)

### Style — Code Consistency

- [ ] **STYLE-01**: Все строки кода укладываются в 80-90 символов (перенос длинных строк)
- [ ] **STYLE-02**: Единый camelCase нейминг во всех файлах (методы, переменные, параметры)
- [ ] **STYLE-03**: `gofmt` проходит чисто на всех .go файлах

### Clean — Repository Hygiene

- [ ] **CLEAN-01**: `.gitignore` исключает: `AGENTS.md`, `CLAUDE.md`, `.beads/`, `.firecrawl/`, `.claude/`, `.tmp/`, `*.postman_collection.json`
- [ ] **CLEAN-02**: Удалены из git tracking: `AGENTS.md`, `CLAUDE.md`, `.beads/`, `.firecrawl/`, Postman collection
- [ ] **CLEAN-03**: `core.hookspath` в git config указывает на актуальный путь (не Desktop)
- [ ] **CLEAN-04**: Нет мусорных файлов в корне репозитория

### Meta — Project Infrastructure

- [ ] **META-01**: `LICENSE` файл с MIT лицензией в корне репозитория
- [ ] **META-02**: `.github/workflows/test.yml` — GitHub Actions CI, прогоняющий `go test ./...` и `go vet ./...` на push
- [ ] **META-03**: `git status` чистый, всё закоммичено
- [ ] **META-04**: Готово к `git push` на приватный GitHub

## Out of Scope

| Feature | Reason |
|---------|--------|
| Новые фичи SDK | Это milestone чистки и полировки |
| Публичный релиз | Репозиторий приватный |
| Интерактивные auth-флоу | Остаётся environment-only |
| E2E тесты с реальным API | Требуют `MLX_RUN_E2E=1` и живых сервисов Multilogin X |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| AUDIT-01 | Phase 1 | Pending |
| AUDIT-02 | Phase 1 | Pending |
| AUDIT-03 | Phase 2 | Pending |
| AUDIT-04 | Phase 2 | Pending |
| STYLE-01 | Phase 3 | Pending |
| STYLE-02 | Phase 3 | Pending |
| STYLE-03 | Phase 3 | Pending |
| CLEAN-01 | Phase 4 | Pending |
| CLEAN-02 | Phase 4 | Pending |
| CLEAN-03 | Phase 4 | Pending |
| CLEAN-04 | Phase 4 | Pending |
| META-01 | Phase 5 | Pending |
| META-02 | Phase 5 | Pending |
| META-03 | Phase 5 | Pending |
| META-04 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 15 total
- Mapped to phases: 15
- Unmapped: 0 ✓

---
*Requirements defined: 2026-05-18*
*Last updated: 2026-05-18 after initial definition*
