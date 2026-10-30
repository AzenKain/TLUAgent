# Agent Mandatory Guidelines & Security Policies

> **CRITICAL**: All AI agents, contributors, and subagents working in this repository MUST strictly follow these mandatory directives. Any violation will result in task rejection.

---

## 1. Zero Tolerance for Hardcoded Secrets & Credentials

- **ABSOLUTE BAN ON HARDCODING SECRETS**: Never hardcode API keys, secret tokens, bearer tokens, passwords, private keys, or client credentials anywhere in the repository:
  - NO secrets in source code (`.go`, `.ts`, `.tsx`, `.js`, `.py`, `.sql`).
  - NO secrets in unit tests, integration tests, or mock files.
  - NO secrets in documentation or comments.
- **NO REALISTIC-LOOKING MOCK KEYS**: Never write fake keys that resemble real credential patterns (e.g., `sk-*`, `sk-or-v1-*`, `AIzaSy*`, `ghp_*`, `gho_*`, `AKIA*`, `xoxb-*`).
  - *Reason*: Even fake keys trigger security scanning tools (GitGuardian, Trufflehog, GitHub Secret Scanning), producing false alarms and security audit failures.
- **NEVER EMBED FALLBACK CREDENTIALS**: Patterns like `if key == "" { key = "my-secret-key" }` are strictly forbidden. If an environment variable is absent, fail fast, return an error, or skip the live test.

---

## 2. Dynamic Secret Management & Testing Standards

- **READ SECRETS FROM ENVIRONMENT ONLY**:
  - All real API keys and secrets must be retrieved via `os.Getenv(...)` or loaded through configuration managers (`config.LoadEnv()`).
- **TEST CREDENTIAL HANDLING**:
  - **Live API Tests**: Read the secret from `os.Getenv(...)`. If the variable is unset, cleanly skip the test using `t.Skip("Skipping live test: missing ENV_VAR")`.
  - **Mock / Cryptographic Tests**: Always generate temporary dummy tokens dynamically at runtime using cryptographically secure random generators (e.g., `crypto.GenerateRandomHex(32)` or `rand.Read(...)`).
- **ENCRYPTION AT REST**:
  - Any API keys, webhooks, or third-party credentials stored in the database MUST be encrypted using AES-256-GCM.
  - Encryption keys must be read from environment variables (`LLM_ENCRYPTION_KEY`) or derived securely from persistent key files.
- **CREDENTIAL MASKING**:
  - Never log plaintext secrets.
  - Never return plaintext secrets in API responses. Responses must always mask credentials (e.g., `sk-...abcd` or `******`).

---

## 3. Ban on Heuristic Guessing Logic

- **DATA-DRIVEN CAPABILITIES ONLY**: Never write hardcoded string-matching heuristic functions to guess capabilities, permissions, or system behaviors (e.g., guessing model capabilities based on model name substrings like `ling-*`, `gemini-*`, or `*-vl`).
- **DATABASE & API METADATA**: All capabilities (such as vision support, allowed file formats, and upload limits) must be explicitly managed through database configuration by administrators or retrieved from official provider metadata APIs.

---

## 4. Mandatory Repository & Database Architecture (`sqlc` + Singleflight + Cache IDs)

- **ABSOLUTE BAN ON RAW SQL IN REPOSITORIES**:
  - Never write inline raw SQL queries (`db.Query`, `db.Exec`, `db.QueryRow` with SQL string literals) inside repository implementations.
  - ALL queries MUST be written in `.sql` files inside `web/db/query/` (e.g. `users.sql`, `roles.sql`, `rag.sql`) and compiled to Go code using `sqlc generate`.
  - Repositories MUST interact with the database exclusively through `q *sqlc.Queries` (initialized via `sqlc.New(db)`).
- **MANDATORY SINGLEFLIGHT TO PREVENT THUNDERING HERDS**:
  - Every repository query method that retrieves data MUST use `singleflight.Group` (`sfg.Do(...)`) from `golang.org/x/sync/singleflight` to collapse concurrent identical database/cache reads into a single execution.
- **MANDATORY RAM CACHE & "CACHE IDS" PATTERN**:
  - Repositories must integrate with `cache.Cache` (`theine-go` / `RamCache`).
  - Search and list operations MUST implement the **Cache IDs pattern** (reference: `web/internal/repositories/userRepository.go`):
    - The search query in `sqlc` filters and returns only entity IDs (`[]string`).
    - The repository resolves entities via `GetByIDs(ctx, ids)` which leverages per-entity RAM cache.
  - Every mutation (Create, Update, Delete) MUST promptly invalidate affected cache keys via an explicit `Invalidate...Cache(ctx, ...)` method.
- **ABSOLUTE BAN ON HARDCODED KNOWLEDGE/DATA JSON FILES**:
  - Never embed or hardcode rules, regulations, decisions, or advisory policies in static `.json` files inside repositories (such as `advisory_rules.json`).
  - All rules, regulations, and knowledge MUST be stored dynamically in database tables (`documents`, `document_rules`, `document_chunks`, `kg_nodes`, `kg_edges`) and queried via `sqlc` and the RAG Knowledge Graph engine.

---

## 5. Code Style & Comment Guidelines

- **NO INLINE COMMENTS**: Absolutely DO NOT put inline `// ...` comments inside function bodies or loops. The code must be self-documenting.
- **CONCISE ENGLISH DOC COMMENTS ONLY**: If a comment is necessary, provide only a brief, 1-line English doc comment above exported/important functions, interfaces, and structs.
- **NO VIETNAMESE COMMENTS IN CODE**: Comments must be in concise English.

---

## 6. Strict Internationalization (i18n) & Zero Hardcoded UI Text

- **ABSOLUTE BAN ON HARDCODED UI TEXT**: Never hardcode user-facing text, button labels, placeholders, tooltips, toasts, modal descriptions, or system replies inside `.tsx`, `.ts`, or `.jsx` source files.
- **MANDATORY i18n USAGE**: All user-facing strings must use `t('key')` via `react-i18next`.
- **100% BILINGUAL PARITY**: Every new key added to `public/locales/vi.json` MUST simultaneously exist in `public/locales/en.json` with accurate translations.
- **ZERO HARDCODED VIETNAMESE IN SOURCE CODE**: Source code files (`.ts`, `.tsx`, `.go`) must never contain literal Vietnamese text; all localization belongs exclusively inside locale translation JSONs.

---

## 7. Mandatory Verification Checklist

Before reporting completion of any task, you MUST run:

1. **SQLC Query Compilation**: `sqlc generate` (Must generate cleanly with zero errors).
2. **Repository Architecture Compliance**: Ensure `grep -rn 'db\.Query\|db\.Exec' internal/repositories/` returns 0 results and all repositories use `*sqlc.Queries` + `singleflight.Group` + RAM cache.
3. **No Hardcoded Knowledge JSON**: Ensure no static rule/knowledge `.json` files exist in `internal/repositories/`.
4. **Full Backend Test Suite**: `go test -count=1 ./...` (All tests must pass).
5. **Server Compilation Check**: `go build -o /dev/null ./cmd/server` (Must compile cleanly).
6. **Frontend Typecheck & Linter**: `npm run check` or `tsc -b && oxlint --deny-warnings` (0 errors, 0 warnings).
7. **Frontend Production Build**: `npm run build` (Must build cleanly).
8. **Secret Scan Verification**: Ensure `grep -rn "sk-" --include="*.go"` and similar pattern checks return 0 results.
9. **i18n Parity & Vietnamese Hardcode Scan**: Ensure `grep -rn -P '[àáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđÀÁẠẢÃÂẦẤẬẨẪĂẰẮẶẲẴÈÉẸẺẼÊỀẾỆỂỄÌÍỊỈĨÒÓỌỎÕÔỒỐỘỔỖƠỜỚỢỞỠÙÚỤỦŨƯỪỨỰỬỮỲÝỴỶỸĐ]' web/frontend/src/` returns 0 results (except the language endonym in LanguageSwitcher).
