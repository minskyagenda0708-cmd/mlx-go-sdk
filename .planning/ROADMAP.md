# Roadmap: mlx-go-sdk Pre-Publish Audit

**Created:** 2026-05-18
**Granularity:** Standard (5 phases)

## Phase 1: Code Audit — Tests & Vet

**Goal:** Убедиться, что код компилируется и тесты проходят.

**Requirements covered:** AUDIT-01, AUDIT-02

**Plans:**
1. Run `go build ./...` — verify compilation
2. Run `go test ./...` (excluding e2e) — verify tests pass
3. Run `go vet ./...` — verify no warnings
4. Fix any failures found

**Deliverables:**
- All tests green
- `go vet` clean

---

## Phase 2: Doc Audit — README & Docs vs Code

**Goal:** Проверить, что документация не врёт.

**Requirements covered:** AUDIT-03, AUDIT-04

**Plans:**
1. Verify README.md — все описанные API, примеры, команды CLI существуют
2. Verify docs/ — cli-reference, verified-workflows, batch-helpers, etc. соответствуют коду
3. Fix discrepancies found

**Deliverables:**
- README.md актуален
- docs/ актуальны

---

## Phase 3: Code Style — Line Width & Naming

**Goal:** Привести код к единому стилю: строки 80-90 символов, консистентный camelCase.

**Requirements covered:** STYLE-01, STYLE-02, STYLE-03

**Plans:**
1. Normalize line widths across all .go files (wrap lines > 90 chars)
2. Audit and fix naming inconsistencies (camelCase, method names)
3. Run `gofmt -w` on all files
4. Verify tests still pass after style changes

**Deliverables:**
- Все строки ≤ 90 символов
- Консистентный нейминг
- `gofmt` clean

---

## Phase 4: Repo Cleanup — .gitignore & Git Config

**Goal:** Вычистить репозиторий от файлов, которые не должны попасть на GitHub.

**Requirements covered:** CLEAN-01, CLEAN-02, CLEAN-03, CLEAN-04

**Plans:**
1. Update `.gitignore` — добавить AGENTS.md, CLAUDE.md, .beads/, .firecrawl/, *.postman_collection.json
2. Remove tracked junk: `git rm --cached` для AGENTS.md, CLAUDE.md, .beads/, .firecrawl/, Postman collection
3. Fix `core.hookspath` в git config
4. Verify `git status` — только нужные файлы

**Deliverables:**
- Чистый `.gitignore`
- Только нужные файлы tracked
- Git config исправлен

---

## Phase 5: Meta — LICENSE & CI

**Goal:** Добавить инфраструктурные файлы и подготовить к push.

**Requirements covered:** META-01, META-02, META-03, META-04

**Plans:**
1. Create `LICENSE` (MIT)
2. Create `.github/workflows/test.yml` (GitHub Actions CI)
3. Final commit
4. Verify `git status` clean

**Deliverables:**
- MIT LICENSE
- GitHub Actions CI
- Готово к `git push`

---
*Roadmap created: 2026-05-18*
