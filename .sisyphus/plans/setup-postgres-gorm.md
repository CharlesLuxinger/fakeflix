# Migrate Upstream S2 Commit to Go

## TL;DR
> **Summary**: Migrate upstream into the Go service by adding PostgreSQL-backed video metadata persistence with GORM while preserving existing upload behavior and PostgreSQL 18.
> **Deliverables**:
> - GORM PostgreSQL dependency and startup connection.
> - `Video` persistence model with AutoMigrate.
> - `POST /video` accepting `title`, `description`, `video`, `thumbnail`, saving files, persisting metadata, and returning JSON record.
> - Tests for happy path, validation failures, DB failure, and cleanup.
> - CI workflow renamed from lint-only to CI with `go test ./...` before lint.
> **Effort**: Medium
> **Parallel**: YES - 3 waves
> **Critical Path**: Task 1 → Task 3 → Task 4 → Task 6 → Final Verification

## Context

### Interview Summary
- Database approach: GORM.
- Migration style: GORM AutoMigrate at startup.
- Duration behavior: keep upstream-compatible constant `100` seconds.
- Response behavior: return persisted video record JSON.
- CI: add `go test ./...` and rename `.github/workflows/lint.yml` to `.github/workflows/ci.yml`.

### Metis Review (gaps addressed)
- Defaulted DB config to `DATABASE_URL`.
- Defaulted startup behavior to fail fast if DB connection or migration fails.
- Defaulted persisted paths to relative paths under `uploads/`.
- Required cleanup on partial failures: remove video if thumbnail fails; remove both files if DB insert fails.
- Guarded against frontend, Prisma, real duration extraction, router/framework, and broad architecture refactors.

## Work Objectives
### Core Objective
Implement the upstream database-backed video upload behavior in idiomatic Go while preserving the current stdlib HTTP stack and PostgreSQL 18 compose setup.

### Deliverables
- GORM PostgreSQL setup using `gorm.io/gorm` and `gorm.io/driver/postgres`.
- `internal/model` or equivalent package containing `Video` model.
- `internal/repository` or equivalent GORM-backed repository for video creation.
- Updated `internal/handler.Video` flow that validates multipart fields, saves files, persists metadata, returns JSON.
- Updated `cmd/server/main.go` that reads `DATABASE_URL` and calls a testable DB bootstrap helper that connects to PostgreSQL, runs `AutoMigrate`, and fails fast on DB errors.
- Updated tests covering upload persistence and failure cleanup.
- `.github/workflows/ci.yml` replacing `.github/workflows/lint.yml`.

### Definition of Done (verifiable conditions with commands)
- `rtk go test ./...` passes.
- `rtk golangci-lint run ./...` passes.
- `rtk git diff -- compose.yml` shows no PostgreSQL downgrade; `postgres:18-alpine` remains.
- `.github/workflows/ci.yml` exists and contains `go test ./...` before golangci-lint.
- `.github/workflows/lint.yml` no longer exists after rename.

### Must Have
- Route remains `POST /video`.
- Multipart fields are exactly `video`, `thumbnail`, `title`, `description`.
- Missing `video`, `thumbnail`, `title`, or `description` returns `400`.
- Invalid video MIME returns `400`; only `video/mp4` accepted.
- Invalid thumbnail MIME returns `400`; only `image/jpeg` accepted.
- Successful upload returns `201 Created` and JSON record.
- JSON response includes `id`, `title`, `description`, `url`, `thumbnailUrl`, `sizeInKb`, `duration`, `createdAt`, `updatedAt`; database table is `videos` with snake_case columns.
- `duration` is always `100`.
- `sizeInKb` is derived from saved video file size rounded/truncated consistently as integer KB; use bytes/1024 integer division.
- Paths stored in DB are relative: `uploads/{filename}`.

### Must NOT Have
- MUST NOT downgrade `compose.yml` from `postgres:18-alpine`.
- MUST NOT add real video duration extraction, ffmpeg, workers, auth, static file serving, OpenAPI, or cloud storage.
- MUST NOT replace stdlib `net/http` with Gin/Echo/Fiber/Chi.
- MUST NOT broaden architecture beyond the minimal model/repository/db packages needed.
- MUST NOT require manual verification.

## Verification Strategy
> ZERO HUMAN INTERVENTION - all verification is agent-executed.
- Test decision: tests-after using existing stdlib `testing`, `httptest`, table-driven patterns.
- QA policy: Every task has agent-executed scenarios.
- Evidence: `.sisyphus/evidence/task-{N}-{slug}.{ext}`.

## Execution Strategy
### Parallel Execution Waves
> Target: 5-8 tasks per wave. <3 per wave (except final) = under-splitting.
> Extract shared dependencies as Wave-1 tasks for max parallelism.

Wave 1: Task 1 foundation dependencies/config; Task 2 domain model/repository; Task 7 CI rename can run after reading current workflow.
Wave 2: Task 3 startup DB wiring; Task 4 upload handler/service persistence; Task 5 tests can begin once interfaces are stable.
Wave 3: Task 6 integration cleanup/hardening; Task 8 full verification and compose guard.

### Dependency Matrix
| Task | Depends On | Blocks |
|---|---|---|
| 1 | none | 2, 3, 4, 5 |
| 2 | 1 | 3, 4, 5 |
| 3 | 1, 2 | 6, 8 |
| 4 | 1, 2 | 5, 6, 8 |
| 5 | 2, 4 | 6, 8 |
| 6 | 3, 4, 5 | 8 |
| 7 | none | 8 |
| 8 | 3, 6, 7 | Final Verification |

### Agent Dispatch Summary
| Wave | Task Count | Categories |
|---|---:|---|
| 1 | 3 | quick, unspecified-low |
| 2 | 3 | unspecified-high, deep |
| 3 | 1 | unspecified-high |
| Final | 4 | oracle, unspecified-high, deep |

## TODOs
> Implementation + Test = ONE task. Never separate.
> EVERY task MUST have: Agent Profile + Parallelization + QA Scenarios.

- [ ] 1. Add GORM Dependencies and DB Config Contract

  **What to do**: Add `gorm.io/gorm` and `gorm.io/driver/postgres` dependencies via `go get`. Define the app config contract as `DATABASE_URL`, defaulting only in tests; production/dev startup must require the environment variable. Document expected local value in code comments or README only if README is already touched by this task; otherwise keep this task code-only.
  **Must NOT do**: Do not add Prisma, sqlc, pgx-only repository code, or change `compose.yml` PostgreSQL image.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: dependency/config contract only.
  - Skills: [`golang-patterns`] - Keep dependency introduction idiomatic.
  - Omitted: [`golang-testing`] - No dedicated tests needed beyond build resolving imports.

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: 2, 3, 4, 5 | Blocked By: none

  **References**:
  - Pattern: `go.mod:1-5` - current module and single dependency style.
  - External: GORM docs `/websites/gorm_io` - PostgreSQL uses `gorm.io/driver/postgres` and `gorm.Open(postgres.Open(dsn), &gorm.Config{})`.

  **Acceptance Criteria**:
  - [ ] `go.mod` includes `gorm.io/gorm` and `gorm.io/driver/postgres`.
  - [ ] `go.sum` is updated by Go tooling.
  - [ ] No file under `src/` is created.
  - [ ] `rtk go test ./...` exits 0 after dependency update.

  **QA Scenarios**:
  ```
  Scenario: Dependencies resolve
    Tool: Bash
    Steps: Run `rtk go test ./...`.
    Expected: Command exits 0; no missing module errors for GORM packages.
    Evidence: .sisyphus/evidence/task-1-gorm-deps.txt

  Scenario: PostgreSQL version preserved
    Tool: Bash
    Steps: Run `rtk git diff -- compose.yml` and inspect image line.
    Expected: No replacement of `postgres:18-alpine` with `postgres:15-alpine`.
    Evidence: .sisyphus/evidence/task-1-compose-guard.txt
  ```

  **Commit**: YES | Message: `chore(db): add gorm dependencies` | Files: `go.mod`, `go.sum`

- [ ] 2. Add Video Model and GORM Repository

  **What to do**: Create a small persistence layer for video metadata. Add `Video` model with fields: `ID string`, `Title string`, `Description string`, `URL string`, `ThumbnailURL string`, `SizeInKB int`, `Duration int`, `CreatedAt time.Time`, `UpdatedAt time.Time`. Use GORM default table `videos` and snake_case database columns (`thumbnail_url`, `size_in_kb`, `created_at`, `updated_at`), with JSON tags for upstream response keys (`thumbnailUrl`, `sizeInKb`, `createdAt`, `updatedAt`); use `gorm:"primaryKey"` for `ID`. Add repository interface and GORM implementation with `Create(ctx context.Context, video *Video) error` using `db.WithContext(ctx).Create(video).Error`.
  **Must NOT do**: Do not place DB logic in HTTP handlers; do not call AutoMigrate here; do not create broad CRUD methods beyond `Create` unless tests require existence checks.

  **Recommended Agent Profile**:
  - Category: `unspecified-low` - Reason: small domain/repository addition.
  - Skills: [`golang-patterns`, `golang-code-style`] - Keep packages small and names idiomatic.
  - Omitted: [`golang-pro`] - No concurrency or performance work.

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: 3, 4, 5 | Blocked By: 1

  **References**:
  - Pattern: `internal/service/video.go:22-35` - domain-specific error/type declarations stay in internal packages.
  - Pattern: `internal/handler/video.go:16-18` - dependency inversion via narrow interfaces.
  - External: GORM docs `/websites/gorm_io` - `primaryKey`, field tags, `CreatedAt`/`UpdatedAt` auto timestamps.
  - Upstream: commit `56b8b6a` `src/schema.prisma` - Video model fields.

  **Acceptance Criteria**:
  - [ ] Model fields and JSON tags produce response keys `id`, `title`, `description`, `url`, `thumbnailUrl`, `sizeInKb`, `duration`, `createdAt`, `updatedAt`.
  - [ ] Repository accepts `context.Context`.
  - [ ] Repository returns wrapped errors using `%w`.
  - [ ] Unit test covers model JSON serialization; repository DB create behavior is verified in Task 6 against PostgreSQL.

  **QA Scenarios**:
  ```
  Scenario: Video JSON schema
    Tool: Bash
    Steps: Run `rtk go test ./internal/... -run TestVideo`.
    Expected: JSON contains exact upstream-compatible keys including `thumbnailUrl` and `sizeInKb`.
    Evidence: .sisyphus/evidence/task-2-video-json.txt

  Scenario: Repository compiles against GORM
    Tool: Bash
    Steps: Run `rtk go test ./internal/repository/...`.
    Expected: Repository package compiles and any non-DB unit tests pass without requiring Docker.
    Evidence: .sisyphus/evidence/task-2-repository-compile.txt
  ```

  **Commit**: YES | Message: `feat(video): add gorm video repository` | Files: `internal/model/**`, `internal/repository/**`, related tests

- [ ] 3. Wire PostgreSQL Connection and AutoMigrate at Startup

  **What to do**: Create a testable DB bootstrap helper, e.g. `internal/database.Open(ctx, dsn string) (*gorm.DB, func() error, error)`, that opens GORM with PostgreSQL, retrieves the underlying `database/sql` handle, pings with context, configures pool values, and runs `AutoMigrate(&model.Video{})`. Update `cmd/server/main.go` to read `DATABASE_URL`, call this helper, fail fast with `log.Fatalf` if env var missing or helper fails, and defer/execute the returned close function during shutdown. Configure pool values: max idle 10, max open 25, max lifetime 1 hour.
  **Must NOT do**: Do not add fallback in-memory/file-only mode; do not silently ignore migration failure; do not add CLI flags.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: startup lifecycle and resource cleanup affect production behavior.
  - Skills: [`golang-patterns`, `golang-code-style`] - Keep lifecycle code readable.
  - Omitted: [`golang-pro`] - No advanced concurrency beyond current graceful shutdown.

  **Parallelization**: Can Parallel: NO | Wave 2 | Blocks: 6, 8 | Blocked By: 1, 2

  **References**:
  - Pattern: `cmd/server/main.go:18-30` - dependency construction before route registration.
  - Pattern: `cmd/server/main.go:31-63` - graceful shutdown and fatal startup errors.
  - Pattern: `compose.yml:6-9` - local DB credentials: `fakeflix`, `postgres`, `postgres`.
  - External: GORM docs `/websites/gorm_io` - PostgreSQL DSN opening and connection pool settings.

  **Acceptance Criteria**:
  - [ ] Missing `DATABASE_URL` exits before listening.
  - [ ] Valid `DATABASE_URL` connects and runs AutoMigrate for Video through `internal/database.Open`.
  - [ ] Server still listens on `:3000`.
  - [ ] Route registrations remain `GET /` and `POST /video`.
  - [ ] DB close is attempted during shutdown.

  **QA Scenarios**:
  ```
  Scenario: Missing DATABASE_URL fails fast
    Tool: Bash
    Steps: Run server command in a subprocess with `DATABASE_URL` unset.
    Expected: Process exits non-zero with message containing `DATABASE_URL` before `server listening on :3000`.
    Evidence: .sisyphus/evidence/task-3-missing-database-url.txt

  Scenario: Compose PostgreSQL connects and migrates
    Tool: Bash
    Steps: Run `rtk docker compose up -d postgres`; set `DATABASE_URL=postgresql://postgres:postgres@localhost:5432/fakeflix?sslmode=disable`; run `rtk go test ./internal/database -run TestOpen_AutoMigratesVideo -count=1 -v`; then run `rtk docker compose exec -T postgres psql -U postgres -d fakeflix -c "\\d videos"`.
    Expected: Go test exits 0 and `psql` output lists columns `id`, `title`, `description`, `url`, `thumbnail_url`, `size_in_kb`, `duration`, `created_at`, `updated_at`.
    Evidence: .sisyphus/evidence/task-3-db-connect.txt
  ```

  **Commit**: YES | Message: `feat(db): connect postgres on startup` | Files: `cmd/server/main.go`, DB/model/repository wiring files

- [ ] 4. Persist Upload Metadata from POST /video

  **What to do**: Extend `internal/handler.Video` dependencies to include a narrow video repository interface. Parse multipart fields `title` and `description`; trim whitespace; return `400` for either missing/blank. Save video first with allowed MIME `video/mp4`; save thumbnail second with allowed MIME `image/jpeg`. Build a `Video` record with generated UUID ID, title, description, `url="uploads/"+videoName`, `thumbnailUrl="uploads/"+thumbnailName`, `sizeInKb` from saved video file size / 1024, `duration=100`; persist through repository; return `201` with `application/json` and the persisted record. Preserve existing cleanup when thumbnail save fails, and add cleanup of both saved files when DB create fails.
  **Must NOT do**: Do not calculate real duration; do not expose absolute filesystem paths; do not keep plain text success response; do not remove existing MIME validation.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: modifies endpoint behavior and failure cleanup.
  - Skills: [`golang-patterns`, `golang-code-style`, `golang-testing`] - Interface design, handler clarity, test coverage.
  - Omitted: [`golang-pro`] - No concurrency or microservices.

  **Parallelization**: Can Parallel: NO | Wave 2 | Blocks: 5, 6, 8 | Blocked By: 1, 2

  **References**:
  - Pattern: `internal/handler/video.go:16-32` - narrow dependency injection.
  - Pattern: `internal/handler/video.go:41-82` - current multipart parsing, save order, and thumbnail cleanup.
  - Pattern: `internal/service/video.go:50-87` - MIME validation and generated filename return.
  - Pattern: `internal/service/video.go:14-19` - current MIME constants.
  - Upstream: commit `56b8b6a` `src/app.controller.ts` - persist title, description, URL, thumbnail URL, size, duration.

  **Acceptance Criteria**:
  - [ ] `POST /video` requires `title`, `description`, `video`, `thumbnail`.
  - [ ] Happy path returns HTTP `201`, JSON content type, persisted record fields, and `duration: 100`.
  - [ ] Missing fields return HTTP `400` with deterministic error body.
  - [ ] Invalid MIME errors remain HTTP `400`.
  - [ ] DB create failure returns HTTP `500` and removes both saved files.
  - [ ] Thumbnail save failure removes already-saved video.

  **QA Scenarios**:
  ```
  Scenario: Happy path persists record
    Tool: Bash
    Steps: Run `rtk go test ./internal/handler/... -run TestVideo_ServeHTTP`.
    Expected: Case with `title=Test Video`, `description=Test Description`, `video.mp4`, `thumbnail.jpg` returns 201 JSON containing non-empty `id`, exact title/description, `duration:100`, relative `url`, relative `thumbnailUrl`.
    Evidence: .sisyphus/evidence/task-4-handler-happy-path.txt

  Scenario: DB failure cleans files
    Tool: Bash
    Steps: Run handler test using fake repository that returns error after both saves.
    Expected: HTTP 500, saved video and thumbnail files do not exist, repository called once.
    Evidence: .sisyphus/evidence/task-4-db-failure-cleanup.txt
  ```

  **Commit**: YES | Message: `feat(video): persist upload metadata` | Files: `internal/handler/video.go`, `internal/service/video.go` if needed, handler tests

- [ ] 5. Expand Tests for Persistence and Validation

  **What to do**: Update existing table-driven tests instead of replacing the style. Extend `internal/handler/video_test.go` mock dependencies to cover saver + repository behavior. Add cases for missing title, missing description, DB failure cleanup, JSON response schema, and preserved invalid MIME behavior. Add model JSON tests. Do not add mandatory DB integration tests here; real GORM/PostgreSQL create behavior belongs to Task 6 and must be skipped unless `DATABASE_URL` is set.
  **Must NOT do**: Do not introduce testify unless strongly justified; do not use brittle sleeps; do not require manual DB cleanup.

  **Recommended Agent Profile**:
  - Category: `deep` - Reason: broad test coverage and edge cases.
  - Skills: [`golang-testing`, `golang-patterns`] - Table-driven tests and clean helpers.
  - Omitted: [`golang-pro`] - No benchmark/concurrency needed.

  **Parallelization**: Can Parallel: PARTIAL | Wave 2 | Blocks: 6, 8 | Blocked By: 2, 4

  **References**:
  - Pattern: `internal/handler/video_test.go:62-191` - table-driven handler tests with `t.Parallel()`.
  - Pattern: `internal/handler/video_test.go:193-280` - multipart request helper pattern.
  - Pattern: `internal/service/video_test.go:16-91` - service table tests with `t.TempDir()`.
  - Pattern: `.golangci.yml:59-64` - testing linters require helper hygiene and parallel tests.
  - Upstream: commit `56b8b6a` `test/video.e2e-spec.ts` - happy path, missing thumbnail, invalid file type coverage.

  **Acceptance Criteria**:
  - [ ] Existing successful upload test now expects JSON, not `video uploaded` text.
  - [ ] Tests cover missing `video`, missing `thumbnail`, missing `title`, missing `description`.
  - [ ] Tests cover invalid video MIME and invalid thumbnail MIME.
  - [ ] Tests cover DB create failure cleanup.
  - [ ] Tests cover `duration == 100` and `sizeInKb` calculation.
  - [ ] `rtk go test ./...` passes.

  **QA Scenarios**:
  ```
  Scenario: Full Go test suite
    Tool: Bash
    Steps: Run `rtk go test ./...`.
    Expected: All packages pass with no race-prone or order-dependent failures.
    Evidence: .sisyphus/evidence/task-5-go-test.txt

  Scenario: Validation matrix
    Tool: Bash
    Steps: Run `rtk go test ./internal/handler/... -run TestVideo_ServeHTTP -v`.
    Expected: Named subtests include missing video, missing thumbnail, missing title, missing description, wrong video MIME, wrong thumbnail MIME, DB failure cleanup.
    Evidence: .sisyphus/evidence/task-5-validation-matrix.txt
  ```

  **Commit**: YES | Message: `test(video): cover persisted upload flow` | Files: `internal/handler/*_test.go`, `internal/repository/*_test.go`, `internal/model/*_test.go` as needed

- [ ] 6. Harden Integration Behavior and Compose Guard

  **What to do**: Add an explicit DB-enabled integration test named `TestVideoUploadPersistsToPostgres` that is skipped when `DATABASE_URL` is unset. The test must call the same DB bootstrap helper from Task 3, use `httptest` with a temp uploads dir, submit multipart `title`, `description`, `video`, and `thumbnail`, assert HTTP 201 JSON, and query GORM/PostgreSQL to confirm one row exists. Run it against local Compose PostgreSQL using `DATABASE_URL=postgresql://postgres:postgres@localhost:5432/fakeflix?sslmode=disable`. Ensure `compose.yml` remains semantically unchanged except optional comments; especially preserve `image: postgres:18-alpine`.
  **Must NOT do**: Do not rename `compose.yml` unless separately requested; do not copy upstream `docker-compose.yml`; do not change volume path to upstream `.data`.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: validates real DB behavior and environment guardrails.
  - Skills: [`golang-testing`, `golang-patterns`] - Integration-safe tests.
  - Omitted: [`golang-pro`] - No distributed system work.

  **Parallelization**: Can Parallel: NO | Wave 3 | Blocks: 8 | Blocked By: 3, 4, 5

  **References**:
  - Pattern: `compose.yml:1-24` - keep PostgreSQL 18, named volume, healthcheck, network.
  - Pattern: `cmd/server/main.go:18-63` - startup/shutdown behavior.
  - Pattern: `internal/handler/video.go:70-79` - cleanup pattern for partial failures.
  - Metis guardrail: fail fast on DB unavailable; no file-only fallback.

  **Acceptance Criteria**:
  - [ ] `compose.yml:3` still reads `image: postgres:18-alpine`.
  - [ ] `docker compose up -d postgres` reaches healthy status.
  - [ ] AutoMigrate creates table for the Video model.
  - [ ] `TestVideoUploadPersistsToPostgres` persists one row and returns JSON when `DATABASE_URL` points to Compose PostgreSQL.
  - [ ] Normal `go test ./...` does not require Docker unless DB env is explicitly set.

  **QA Scenarios**:
  ```
  Scenario: Real PostgreSQL smoke path
    Tool: Bash
    Steps: Run `rtk docker compose up -d postgres`; set `DATABASE_URL=postgresql://postgres:postgres@localhost:5432/fakeflix?sslmode=disable`; run `rtk go test ./internal/handler -run TestVideoUploadPersistsToPostgres -count=1 -v`; then run `rtk docker compose exec -T postgres psql -U postgres -d fakeflix -c "select title, description, duration from videos order by created_at desc limit 1;"`.
    Expected: Go test exits 0; SQL output contains `Test Video`, `Test Description`, and `100`.
    Evidence: .sisyphus/evidence/task-6-postgres-smoke.txt

  Scenario: Compose version guard
    Tool: Bash
    Steps: Run `rtk grep "postgres:" compose.yml` and `rtk git diff -- compose.yml`.
    Expected: Only `postgres:18-alpine` appears; no `postgres:15-alpine` appears.
    Evidence: .sisyphus/evidence/task-6-compose-version.txt
  ```

  **Commit**: YES | Message: `test(db): verify postgres persistence path` | Files: DB smoke tests/helpers; `compose.yml` only if unavoidable and never downgraded

- [ ] 7. Rename Lint Workflow to CI and Add Tests

  **What to do**: Rename `.github/workflows/lint.yml` to `.github/workflows/ci.yml`. Change workflow name from `Lint` to `CI`. Keep triggers and `actions/setup-go@v5` using `go-version-file: go.mod`. Add a test step after Go setup and before golangci-lint: `go test ./...`. Keep golangci-lint action version `v2.12.2`.
  **Must NOT do**: Do not remove lint; do not add PostgreSQL service to CI unless DB tests require it and are guarded.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: small workflow rename and test step.
  - Skills: [] - YAML-only change.
  - Omitted: [`golang-lint`] - Existing lint config remains unchanged.

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: 8 | Blocked By: none

  **References**:
  - Pattern: `.github/workflows/lint.yml:1-26` - existing triggers, setup-go, golangci-lint version.
  - Pattern: `.golangci.yml:1-8` - lint runs tests-aware analysis.

  **Acceptance Criteria**:
  - [ ] `.github/workflows/lint.yml` is deleted/renamed.
  - [ ] `.github/workflows/ci.yml` exists.
  - [ ] Workflow `name:` is `CI`.
  - [ ] `go test ./...` runs before `golangci/golangci-lint-action@v7`.
  - [ ] Golangci-lint version remains `v2.12.2`.

  **QA Scenarios**:
  ```
  Scenario: Workflow file renamed
    Tool: Bash
    Steps: Run `rtk git status --short .github/workflows`.
    Expected: Shows lint workflow removed/renamed and ci workflow added; no duplicate workflows.
    Evidence: .sisyphus/evidence/task-7-workflow-rename.txt

  Scenario: CI contains tests before lint
    Tool: Bash
    Steps: Run `rtk grep "go test ./...\|golangci-lint" .github/workflows/ci.yml`.
    Expected: `go test ./...` appears before `golangci-lint-action@v7`; version remains `v2.12.2`.
    Evidence: .sisyphus/evidence/task-7-ci-steps.txt
  ```

  **Commit**: YES | Message: `ci(go): run tests before lint` | Files: `.github/workflows/lint.yml`, `.github/workflows/ci.yml`

- [ ] 8. Final Local Verification and Evidence Capture

  **What to do**: Run the complete verification set after all implementation tasks. Capture evidence files for tests, lint, compose guard, and workflow guard. Fix any failures by returning to the smallest responsible task area.
  **Must NOT do**: Do not mark final verification complete without all commands passing; do not ignore lint failures; do not use `--fix` unless a separate code cleanup change is intentional and reviewed.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: cross-cutting final verification.
  - Skills: [`golang-lint`, `golang-testing`] - Run and interpret checks.
  - Omitted: [`golang-pro`] - No new implementation logic.

  **Parallelization**: Can Parallel: NO | Wave 3 | Blocks: Final Verification | Blocked By: 3, 6, 7

  **References**:
  - Pattern: `Makefile:3-10` - `make lint` target runs `golangci-lint run ./...`.
  - Pattern: `.golangci.yml:13-70` - strict lint set to satisfy.
  - Pattern: `compose.yml:3` - PostgreSQL image guard.
  - Pattern: `go.mod:1-5` - module and Go version.

  **Acceptance Criteria**:
  - [ ] `rtk go test ./...` passes.
  - [ ] `rtk golangci-lint run ./...` passes.
  - [ ] `rtk git diff -- compose.yml` shows no downgrade.
  - [ ] `rtk git diff -- compose.yml` shows no `postgres:15-alpine` reference.

  **QA Scenarios**:
  ```
  Scenario: Complete verification
    Tool: Bash
    Steps: Run `rtk go test ./...` then `rtk golangci-lint run ./...`.
    Expected: Both commands exit 0.
    Evidence: .sisyphus/evidence/task-8-tests-lint.txt

  Scenario: Scope guard
    Tool: Bash
    Steps: Run `rtk git diff --stat`; inspect changed files.
    Expected: Changes limited to Go source/tests, go.mod/go.sum, CI workflow.
    Evidence: .sisyphus/evidence/task-8-scope-guard.txt
  ```

  **Commit**: YES | Message: `chore: verify migrated upload flow` | Files: only fixes needed from verification


## Final Verification Wave (MANDATORY — after ALL implementation tasks)
> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**
> **Never mark F1-F4 as checked before getting user's okay.** Rejection or user feedback -> fix -> re-run -> present again -> wait for okay.
- [ ] F1. Plan Compliance Audit — oracle
- [ ] F2. Code Quality Review — unspecified-high
- [ ] F3. Agent API QA — unspecified-high
- [ ] F4. Scope Fidelity Check — deep

## Commit Strategy
- Use atomic commits per completed task where files are coherent.
- Suggested final commit: `feat(video): persist uploads with gorm`.
- Do not commit evidence files unless project convention requires them.

## Success Criteria
- Upstream S2 behavior is represented in Go.
- PostgreSQL remains `postgres:18-alpine`.
- Existing upload validation remains covered.
- Metadata is persisted and returned as JSON.
- CI runs tests and lint.
