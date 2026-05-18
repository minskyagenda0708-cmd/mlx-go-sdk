# mlx-go-sdk

## What This Is

Go SDK для Multilogin X — типизированная клиентская библиотека для управления браузерными профилями, запуска/остановки браузеров, работы с cookies, генерации прокси, импорта/экспорта и проверенных высокоуровневых workflow. Включает референсный CLI.

## Core Value

Надёжный, хорошо задокументированный Go SDK, позволяющий разработчикам уверенно автоматизировать Multilogin X браузерные профили.

## Requirements

### Validated

- ✓ Profiles CRUD (create, search, patch, move, clone, meta reads) — existing
- ✓ Launcher control (start, stop, status, version, health) — existing
- ✓ Transfers (import/export job control) — existing
- ✓ Archive management (export-to-folder file organization) — existing
- ✓ Cookies (metadata, list, import/export, cookie seeding) — existing
- ✓ Resources (templates, extensions, object storage flows) — existing
- ✓ Proxy generation (MLX managed proxies, parsing) — existing
- ✓ High-level workflows (StartProfileByName, ExportProfileByNameToFolder, etc.) — existing
- ✓ Retry with exponential backoff + jitter — existing
- ✓ Error classification (TransportError, ErrorResponse, ErrorClass) — existing
- ✓ Generic polling (pollUntil) — existing
- ✓ Reference CLI scaffold (cmd/mlx) — existing
- ✓ Functional options pattern for client config — existing
- ✓ Context-aware — all service methods accept context.Context — existing

### Active

- [ ] **AUDIT-01**: Все тесты проходят (`go test ./...`)
- [ ] **AUDIT-02**: Документация (README + docs/) соответствует реальному коду
- [ ] **STYLE-01**: Ширина строк кода нормализована до 80-90 символов
- [ ] **STYLE-02**: Единый стиль нейминга (camelCase, консистентные имена методов)
- [ ] **CLEAN-01**: `.gitignore` вычищен — исключены AGENTS.md, CLAUDE.md, .beads/, .firecrawl/, .claude/, .tmp/, Postman collection
- [ ] **CLEAN-02**: Git config исправлен (core.hookspath указывает на актуальный путь)
- [ ] **META-01**: Добавлен MIT LICENSE файл
- [ ] **META-02**: Настроен GitHub Actions CI для авто-прогона тестов
- [ ] **META-03**: Репозиторий готов к push на приватный GitHub

### Out of Scope

- Новые фичи — это milestone чистки и полировки перед публикацией
- Публичный релиз — репозиторий будет приватным
- Интерактивные auth-флоу — остаётся environment-only (`MLX_TOKEN`)
- Мобильные профили — не в скоупе CLI

## Context

- **Репозиторий:** `c:\Users\bath0ry\mlx\mlx-go-sdk`
- **Язык:** Go 1.26, модуль `mlx-go-sdk`
- **Зависимости:** Только стандартная библиотека (go-rod — indirect, для consumer-кода)
- **API:** Multilogin X (REST, Launcher, Cookies, Proxy — 4 эндпоинта)
- **Codebase map:** уже существует в `.planning/codebase/`
- **Git:** инициализирован, user = Marvin, email = minsky.agenda0708@gmail.com
- **Проблема git config:** `core.hookspath` указывает на старый путь `C:\Users\bath0ry\Desktop\mlx-go-sdk\.beads\hooks` — нужно поправить на актуальный
- **Проблема стиля:** массовые нарушения ширины строк (до 589 символов), неконсистентный нейминг
- **Проблема чистоты:** AGENTS.md, CLAUDE.md, .beads/, .firecrawl/ отслеживаются git, но не должны попасть на GitHub

## Constraints

- **Tech stack:** Go 1.26, только стандартная библиотека для SDK
- **Auth:** Environment-only через `MLX_TOKEN`, без конфиг-файлов
- **Стиль:** Ширина строк 80-90 символов, консистентный camelCase
- **Приватность:** Репозиторий приватный, но всё равно должен быть чистым
- **Совместимость:** Не ломать существующие API и тесты

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Плоский пакет — все типы в одном `mlx` package | Простота использования, нет вложенных импортов | ✓ Good |
| Functional options для конфигурации | Идиоматичный Go, расширяемость без breaking changes | ✓ Good |
| Интерфейсы для каждого сервиса | Тестируемость, возможность моков | ✓ Good |
| Retry safety — клонирование body перед повтором | Идемпотентность повторных запросов | ✓ Good |
| MIT лицензия | Максимальная permissive, стандарт для Go-экосистемы | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-05-18 after initialization*
