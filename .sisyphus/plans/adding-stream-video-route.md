# Migrate Video Streaming Endpoint

## TL;DR
> **Summary**: Adding an idiomatic Go `GET /stream/{videoId}` endpoint that streams persisted MP4 files with HTTP range support.
> **Deliverables**:
> - Tests-first coverage for stream success, full GET, missing records, missing files, unsafe paths, and invalid ranges.
> - Repository video lookup by ID with explicit not-found semantics.
> - Separate Go stream handler using `http.ServeContent`.
> - Route registration in `cmd/server/main.go`.
> - Validation evidence from `go test ./...` and `make lint`.
> **Effort**: Short
> **Parallel**: YES - 3 waves
> **Critical Path**: Task 1 → Task 2 → Task 3 → Task 4

## Context
### Original Request
User requested migration of GitHub commit `https://github.com/tech-leads-club/enterprise-apps-classes/commit/b40f2febc3ec5f8ef4591ffe3bccdea6e57177f9` into this repo as a Specialist Golang Engineer, using skills in `./.agents/skills`, preserving the existing compose file, not downgrading PostgreSQL, and following Go best practices.

### Interview Summary
- Source access: use authenticated `gh` CLI.
- Migration style: adapt source behavior to existing Fakeflix Go conventions.
- Test strategy: tests-before, then run full validation at the end.
- PostgreSQL: preserve current `postgres:18-alpine` in `compose.yml`.

### Source Commit Summary
Authenticated `gh api repos/tech-leads-club/enterprise-apps-classes/commits/b40f2febc3ec5f8ef4591ffe3bccdea6e57177f9` returned commit message `s3` with two changed files:
- `src/app.controller.ts`: adds NestJS `GET /stream/:videoId`.
- `test/video.e2e-spec.ts`: adds e2e coverage for ranged streaming and 404 missing video.

Source behavior to migrate:
- Lookup video by ID.
- Unknown ID returns 404.
- Stored `video.url` identifies the uploaded MP4 file.
- `Range: bytes=start-end` returns `206 Partial Content`.
- Response headers include `Content-Range`, `Accept-Ranges`, `Content-Length`, and `Content-Type: video/mp4`.
- Full GET without Range should stream the full body in Go; do **not** mirror the source bug where the non-range response writes headers but does not pipe the file.

### Local Repository Summary
- `cmd/server/main.go:20-86`: server entry point, creates `./uploads`, opens DB via `DATABASE_URL`, registers `GET /` and `POST /video`, listens on `:3000`.
- `internal/handler/video.go:20-153`: upload handler with `videoSaver`, `videoCreator`, multipart validation, persisted `model.Video` JSON response.
- `internal/repository/video.go:12-32`: `VideoRepository` currently supports only `Create(ctx, video)`.
- `internal/model/video.go:7-17`: `Video` fields include `ID`, `URL`, `ThumbnailURL`, metadata, timestamps.
- `compose.yml:1-24`: PostgreSQL service uses `postgres:18-alpine`; must remain unchanged.
- `internal/handler/video_test.go:84-421`: table-driven upload handler tests and helper patterns.
- `internal/handler/video_integration_test.go:20-175`: DB-backed integration test pattern gated by `DATABASE_URL`.
- `.golangci.yml:13-70`: strict lint set, including `gosec`, `paralleltest`, `funlen`, `gocyclo`, `errcheck`.

### Metis Review (gaps addressed)
- Missing local file decision: return `404 Not Found`.
- DB not-found decision: explicit `repository.ErrVideoNotFound`, mapped to `404`.
- DB non-not-found errors: map to `500`.
- Path safety: never trust `video.URL`; stream only `filepath.Base(video.URL)` under injected uploads directory.
- Full non-range GET: intentionally supported with `200 OK` and full body.
- Invalid/malformed/unsatisfiable ranges: use stdlib `http.ServeContent` behavior, expected `416`.
- Scope guardrails: no new router/framework, no auth, no transcoding, no CDN, no PostgreSQL downgrade.

## Work Objectives
### Core Objective
Add an idiomatic Go streaming endpoint equivalent to the source commit behavior while preserving local layered architecture and Go testing/lint standards.

### Deliverables
- Failing tests written before implementation changes within each task flow.
- `FindByID(ctx, id)` repository contract and GORM implementation.
- `internal/handler/stream.go` with a dedicated stream handler.
- `GET /stream/{videoId}` route in `cmd/server/main.go`.
- Validation evidence under `.sisyphus/evidence/`.

### Definition of Done (verifiable conditions with commands)
- `rtk go test ./...` passes.
- `rtk make lint` passes.
- `rtk go test -run 'TestStream|TestVideoStream|TestVideoRepository' ./...` passes.
- Existing `POST /video` behavior remains covered and unchanged.
- `compose.yml:3` remains `image: postgres:18-alpine`.

### Must Have
- Use tests-before sequencing inside each implementation task.
- Use standard library `http.ServeContent` for range handling.
- Set `Content-Type: video/mp4` before serving content.
- Return `404` for unknown video IDs.
- Return `404` for missing stored files.
- Constrain file access to the configured uploads directory by using `filepath.Base(video.URL)`.
- Preserve existing upload route and JSON contract.
- Preserve existing repository constructor pattern.

### Must NOT Have
- MUST NOT downgrade or modify PostgreSQL image in `compose.yml`.
- MUST NOT introduce Gin, Chi, Echo, Fiber, or any new router/framework.
- MUST NOT add auth, JWT, billing, CDN, S3, transcoding, thumbnails, queues, GraphQL, or AI integration.
- MUST NOT hand-roll HTTP range parsing unless `http.ServeContent` cannot satisfy tests.
- MUST NOT stream arbitrary filesystem paths from database content.
- MUST NOT change `POST /video` status codes, response fields, MIME validation, or upload storage conventions.
- MUST NOT require browser/manual QA.

## Verification Strategy
> ZERO HUMAN INTERVENTION - all verification is agent-executed.
- Test decision: tests-before + Go `testing` package, `httptest`, GORM integration tests gated by `DATABASE_URL`.
- QA policy: Every task has agent-executed scenarios.
- Evidence: `.sisyphus/evidence/task-{N}-{slug}.{ext}`.

## Execution Strategy
### Parallel Execution Waves
> Target: 5-8 tasks per wave. This migration is small; fewer tasks are intentional to avoid artificial splitting.
> Extract shared dependencies as Wave-1 tasks for max parallelism.

Wave 1: Task 1 repository lookup tests + implementation.
Wave 2: Task 2 stream handler tests + implementation.
Wave 3: Task 3 route registration + integration validation, Task 4 final validation hardening.

### Dependency Matrix (full, all tasks)
| Task | Depends On | Blocks |
|---|---|---|
| 1. Add video lookup repository contract | None | 2 |
| 2. Add stream handler | 1 | 3 |
| 3. Register stream route and integration coverage | 2 | 4 |
| 4. Final validation and evidence | 1, 2, 3 | Final Verification Wave |

### Agent Dispatch Summary (wave → task count → categories)
| Wave | Tasks | Categories |
|---|---:|---|
| 1 | 1 | quick |
| 2 | 1 | deep |
| 3 | 2 | quick, unspecified-high |

## TODOs
> Implementation + Test = ONE task. Never separate.
> EVERY task MUST have: Agent Profile + Parallelization + QA Scenarios.

- [ ] 1. Add video lookup repository contract

  **What to do**: Write failing tests first for repository lookup by ID, then extend the repository contract and GORM implementation.
  1. Add tests in `internal/repository/video_test.go` before implementation.
  2. Test successful lookup: create a `model.Video`, call `FindByID(ctx, id)`, assert returned fields match.
  3. Test missing lookup: call `FindByID(ctx, "missing-id")`, assert `errors.Is(err, repository.ErrVideoNotFound)`.
  4. Add `ErrVideoNotFound` in `internal/repository/video.go` as a package-level sentinel error.
  5. Extend `VideoRepository` with `FindByID(ctx context.Context, id string) (*model.Video, error)`.
  6. Implement with `db.WithContext(ctx).First(&video, "id = ?", id)`.
  7. Map `gorm.ErrRecordNotFound` to `ErrVideoNotFound` using `errors.Is`.
  8. Wrap other DB errors with context: `find video by id %q: %w`.

  **Must NOT do**: Do not change `Create` behavior. Do not add UUID parsing. Do not change model fields. Do not modify `compose.yml`.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: localized repository change with clear tests.
  - Skills: [`golang-testing`, `golang-patterns`, `golang-code-style`] - tests-first and idiomatic error semantics.
  - Omitted: [`playwright-cli`] - no browser/UI involved.

  **Parallelization**: Can Parallel: NO | Wave 1 | Blocks: [2] | Blocked By: []

  **References** (executor has NO interview context - be exhaustive):
  - Pattern: `internal/repository/video.go:12-32` - existing repository interface, constructor, GORM create pattern.
  - API/Type: `internal/model/video.go:7-17` - persisted `Video` fields to assert.
  - Test: `internal/repository/video_test.go` - existing repository test file to extend.
  - Test: `internal/handler/video_integration_test.go:20-39` - DB setup/cleanup pattern gated by `DATABASE_URL`.
  - External: Go `errors.Is` standard library - use sentinel error matching.
  - External: GORM `ErrRecordNotFound` - map DB not-found to domain sentinel.

  **Acceptance Criteria** (agent-executable only):
  - [ ] `rtk go test ./internal/repository -run TestVideoRepository` passes.
  - [ ] `rtk go test ./...` passes after repository changes.
  - [ ] `internal/repository/video.go` exposes `ErrVideoNotFound` and `FindByID(ctx,id)`.
  - [ ] Missing row assertion uses `errors.Is(err, repository.ErrVideoNotFound)`.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Repository finds existing video
    Tool: Bash
    Steps: Run `rtk go test ./internal/repository -run TestVideoRepository` with DATABASE_URL set if repository tests require PostgreSQL; otherwise run unit test directly.
    Expected: Test passes and asserts ID, URL, title, description, thumbnail URL, size, duration match inserted record.
    Evidence: .sisyphus/evidence/task-1-repository-find.txt

  Scenario: Repository reports missing video
    Tool: Bash
    Steps: Run `rtk go test ./internal/repository -run TestVideoRepository`.
    Expected: Missing ID test passes with `errors.Is(err, repository.ErrVideoNotFound)` and no nil video returned.
    Evidence: .sisyphus/evidence/task-1-repository-not-found.txt
  ```

  **Commit**: NO | Message: `feat(streaming): add video lookup repository` | Files: [`internal/repository/video.go`, `internal/repository/video_test.go`]

- [ ] 2. Add dedicated video stream handler

  **What to do**: Write failing handler tests first, then add a separate stream handler in `internal/handler/stream.go` using `http.ServeContent`.
  1. Create `internal/handler/stream_test.go` before implementation.
  2. Define a mock repository implementing `FindByID(ctx,id)`.
  3. Test ranged GET for a temp MP4 file containing known bytes, with `Range: bytes=0-3`.
  4. Test full GET without `Range` returns `200 OK` and full body.
  5. Test unknown video maps `repository.ErrVideoNotFound` to `404`.
  6. Test existing video with missing file returns `404`.
  7. Test unsafe stored URL such as `../../secret.mp4` cannot escape uploads dir; expected `404`.
  8. Test invalid or unsatisfiable range returns `416 Requested Range Not Satisfiable`.
  9. Implement `type VideoStream struct { repo videoFinder; uploadsDir string }` with constructor `NewVideoStream(repo videoFinder, uploadsDir string) *VideoStream`.
  10. In `ServeHTTP`, reject non-GET with `405` for direct handler tests.
  11. Extract `videoId := r.PathValue("videoId")`; blank ID returns `404`.
  12. Lookup repository before opening file.
  13. Resolve file path as `filepath.Join(uploadsDir, filepath.Base(video.URL))`.
  14. Open file read-only; missing file returns `404`, other open/stat errors return `500`.
  15. Set `Content-Type: video/mp4` and call `http.ServeContent(w, r, filepath.Base(video.URL), fileInfo.ModTime(), file)`.
  16. Ensure file close errors are handled/logged according to existing code style.

  **Must NOT do**: Do not modify upload handler except if an interface name conflict requires renaming. Do not parse ranges manually. Do not serve `video.URL` directly. Do not add dependencies.

  **Recommended Agent Profile**:
  - Category: `deep` - Reason: security-sensitive file serving and HTTP range behavior.
  - Skills: [`golang-testing`, `golang-patterns`, `golang-code-style`] - handler tests, `ServeContent`, path safety.
  - Omitted: [`golang-pro`] - no concurrency/microservice complexity needed.

  **Parallelization**: Can Parallel: NO | Wave 2 | Blocks: [3, 4] | Blocked By: [1]

  **References** (executor has NO interview context - be exhaustive):
  - Pattern: `internal/handler/video.go:20-42` - local interface and constructor style.
  - Pattern: `internal/handler/video.go:44-153` - handler method/error response style.
  - Pattern: `internal/handler/video_test.go:84-421` - table-driven tests, mocks, `httptest`, helper conventions.
  - API/Type: `internal/model/video.go:7-17` - use `Video.URL` for stored path.
  - API/Type: `internal/repository/video.go` after Task 1 - use `FindByID` and `ErrVideoNotFound`.
  - Source: `src/app.controller.ts` from commit `b40f2feb...` - original route `/stream/:videoId`, range streaming contract.
  - Source Test: `test/video.e2e-spec.ts` from commit `b40f2feb...` - expected `206`, `Content-Range`, `Accept-Ranges`, `Content-Length`, `Content-Type`.
  - External: Go `net/http.ServeContent` - robust range implementation.

  **Acceptance Criteria** (agent-executable only):
  - [ ] `rtk go test ./internal/handler -run TestVideoStream` passes.
  - [ ] Ranged request `bytes=0-3` returns `206`, `Content-Range: bytes 0-3/{size}`, `Accept-Ranges: bytes`, `Content-Length: 4`, `Content-Type` containing `video/mp4`, and body equals first four bytes.
  - [ ] Full GET returns `200` and full body.
  - [ ] Unknown ID returns `404`.
  - [ ] Existing DB record with missing file returns `404`.
  - [ ] Unsafe stored URL cannot escape uploads dir and returns `404`.
  - [ ] Invalid/unsatisfiable range returns `416`.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Stream partial MP4 bytes
    Tool: Bash
    Steps: Run `rtk go test ./internal/handler -run TestVideoStream`.
    Expected: Range test passes with status 206, correct range headers, and exact byte body.
    Evidence: .sisyphus/evidence/task-2-stream-range.txt

  Scenario: Missing and unsafe stream targets fail safely
    Tool: Bash
    Steps: Run `rtk go test ./internal/handler -run 'TestVideoStream/(unknown|missing|unsafe|invalid)'` or the exact subtest names implemented.
    Expected: Unknown/missing/unsafe cases return 404; invalid range returns 416; no filesystem escape occurs.
    Evidence: .sisyphus/evidence/task-2-stream-failures.txt
  ```

  **Commit**: NO | Message: `feat(streaming): add stream handler` | Files: [`internal/handler/stream.go`, `internal/handler/stream_test.go`]

- [ ] 3. Register `GET /stream/{videoId}` and add integration coverage

  **What to do**: Wire the handler into the server and add DB-backed coverage proving upload + stream works together.
  1. Update `cmd/server/main.go` to construct `streamHandler := handler.NewVideoStream(videoRepo, "./uploads")` after `videoRepo` exists.
  2. Register `mux.Handle("GET /stream/{videoId}", streamHandler)`.
  3. Add or extend integration coverage in `internal/handler/video_integration_test.go` or create `internal/handler/stream_integration_test.go`.
  4. Integration setup must use `DATABASE_URL` gating like `internal/handler/video_integration_test.go:20-25`.
  5. Integration test should upload a video through existing `handler.NewVideo`, capture returned ID and URL, then stream via `NewVideoStream` with a request path `/stream/{id}` and `Range: bytes=0-3`.
  6. Assert DB-backed stream returns `206` and body bytes match uploaded content prefix.
  7. Add integration test for unknown UUID returning `404`, matching source e2e intent.
  8. Keep route registration compile-safe with Go 1.26 `http.ServeMux` path values.

  **Must NOT do**: Do not change `POST /video` route. Do not add Makefile targets unless needed for validation evidence. Do not alter compose PostgreSQL image.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: wiring and focused integration tests.
  - Skills: [`golang-testing`, `golang-patterns`] - integration test and mux route conventions.
  - Omitted: [`playwright-cli`] - no browser/UI involved.

  **Parallelization**: Can Parallel: NO | Wave 3 | Blocks: [4] | Blocked By: [2]

  **References** (executor has NO interview context - be exhaustive):
  - Pattern: `cmd/server/main.go:43-51` - dependency construction and route registration.
  - Pattern: `internal/handler/video_integration_test.go:20-175` - DB setup, upload request construction, cleanup.
  - Pattern: `internal/handler/video_test.go:281-321` - request helper style.
  - Infra: `compose.yml:1-24` - PostgreSQL service, do not downgrade.
  - Source Test: `test/video.e2e-spec.ts` from source commit - upload then stream by returned ID.

  **Acceptance Criteria** (agent-executable only):
  - [ ] `rtk go test ./internal/handler -run 'Test.*Stream.*Integration|TestVideoUploadPersistsToPostgres'` passes with `DATABASE_URL` when Postgres is available.
  - [ ] `rtk go test ./...` passes without `DATABASE_URL`; integration tests skip cleanly.
  - [ ] `cmd/server/main.go` contains `GET /stream/{videoId}` route.
  - [ ] `POST /video` route remains registered and unchanged.
  - [ ] `compose.yml:3` remains `image: postgres:18-alpine`.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Uploaded video streams by ID
    Tool: Bash
    Steps: Start PostgreSQL if needed with `rtk docker compose up -d postgres`; set `DATABASE_URL=postgres://postgres:postgres@localhost:5432/fakeflix?sslmode=disable`; run targeted handler integration stream test.
    Expected: Upload returns ID; stream by ID with range returns 206 and expected body prefix.
    Evidence: .sisyphus/evidence/task-3-upload-stream-integration.txt

  Scenario: Unknown stream ID through integration stack
    Tool: Bash
    Steps: Run integration test for `/stream/45705b56-a47f-4869-b736-8f6626c940f8` equivalent.
    Expected: Response status is 404 without panic or file access.
    Evidence: .sisyphus/evidence/task-3-stream-unknown-integration.txt
  ```

  **Commit**: NO | Message: `feat(streaming): wire stream endpoint` | Files: [`cmd/server/main.go`, `internal/handler/video_integration_test.go` or `internal/handler/stream_integration_test.go`]

- [ ] 4. Run final validation and capture evidence

  **What to do**: Execute validation commands and fix only issues directly caused by Tasks 1-3.
  1. Run `rtk go test ./...`.
  2. Run `rtk make lint`.
  3. Run targeted stream tests: `rtk go test ./internal/handler -run TestVideoStream -v`.
  4. Run targeted repository tests: `rtk go test ./internal/repository -run TestVideoRepository -v`.
  5. Verify `compose.yml:3` still reads `image: postgres:18-alpine`.
  6. If validation fails due to lint style, fix within changed files only.
  7. If validation fails due to existing unrelated issue, capture output and report separately; do not broaden scope.

  **Must NOT do**: Do not run formatters that rewrite unrelated files. Do not change source behavior beyond planned streaming. Do not commit automatically unless explicitly instructed by user outside this plan.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: validation triage may require careful distinction between caused and pre-existing failures.
  - Skills: [`golang-lint`, `golang-testing`, `golang-code-style`] - lint/test interpretation.
  - Omitted: [`frontend-ui-ux`] - no frontend work.

  **Parallelization**: Can Parallel: NO | Wave 3 | Blocks: [Final Verification Wave] | Blocked By: [1, 2, 3]

  **References** (executor has NO interview context - be exhaustive):
  - CI: `.github/workflows/ci.yml` - CI runs tests and golangci-lint.
  - Lint config: `.golangci.yml:13-70` - strict enabled linters.
  - Makefile: `Makefile` - `make lint`, `make lint-fix`, `make fmt` targets.
  - Infra: `compose.yml:3` - PostgreSQL version guardrail.

  **Acceptance Criteria** (agent-executable only):
  - [ ] `rtk go test ./...` output captured and passing.
  - [ ] `rtk make lint` output captured and passing.
  - [ ] Targeted stream handler tests output captured and passing.
  - [ ] Targeted repository tests output captured and passing.
  - [ ] PostgreSQL image remains `postgres:18-alpine`.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Full automated validation
    Tool: Bash
    Steps: Run `rtk go test ./...` and `rtk make lint` from repo root.
    Expected: Both commands exit 0; no new lint/test failures.
    Evidence: .sisyphus/evidence/task-4-full-validation.txt

  Scenario: PostgreSQL version preserved
    Tool: Bash
    Steps: Run `rtk grep 'postgres:18-alpine' compose.yml`.
    Expected: Output includes `image: postgres:18-alpine`; no other postgres image downgrade appears.
    Evidence: .sisyphus/evidence/task-4-postgres-version.txt
  ```

  **Commit**: YES | Message: `feat(streaming): add video range streaming endpoint` | Files: [`cmd/server/main.go`, `internal/handler/stream.go`, `internal/handler/stream_test.go`, `internal/handler/video_integration_test.go` or `internal/handler/stream_integration_test.go`, `internal/repository/video.go`, `internal/repository/video_test.go`]

## Final Verification Wave (MANDATORY — after ALL implementation tasks)
> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**
> **Never mark F1-F4 as checked before getting user's okay.** Rejection or user feedback -> fix -> re-run -> present again -> wait for okay.
> Final verification items are mandatory tasks and include executable QA instructions below.

- [ ] F1. Plan Compliance Audit — oracle

  **Agent-executable checks**:
  - Read `.sisyphus/plans/migrate-stream-video.md` and inspect changed files only.
  - Verify every implementation task was completed as specified.
  - Verify `GET /stream/{videoId}` exists and `POST /video` remains unchanged.
  - Verify `compose.yml:3` still uses `postgres:18-alpine`.
  - Verify source-specified behavior is represented: ranged stream 206, unknown video 404.

  **Evidence**: `.sisyphus/evidence/f1-plan-compliance.md`
  **Approval condition**: APPROVE only if all plan tasks and guardrails are satisfied; otherwise list exact task numbers to fix.

- [ ] F2. Code Quality Review — unspecified-high

  **Agent-executable checks**:
  - Run `rtk go test ./...`.
  - Run `rtk make lint`.
  - Inspect changed Go files for idiomatic error handling, small interfaces, no manual range parsing, no new dependencies, and no path traversal.
  - Verify handler code uses early returns and keeps upload handler complexity stable.

  **Evidence**: `.sisyphus/evidence/f2-code-quality.md`
  **Approval condition**: APPROVE only if tests/lint pass and changed code follows local Go style.

- [ ] F3. Agent-Executed API QA — unspecified-high

  **Agent-executable checks**:
  - Use Go tests or a temporary local server plus Bash HTTP requests; no browser and no human manual QA.
  - Verify `GET /stream/{videoId}` with `Range: bytes=0-3` returns `206`, `Content-Range`, `Accept-Ranges`, `Content-Length: 4`, `Content-Type: video/mp4`, and exact body bytes.
  - Verify `GET /stream/{videoId}` without `Range` returns `200` and full body.
  - Verify unknown ID returns `404`.
  - Verify missing file returns `404`.
  - Verify invalid/unsatisfiable range returns `416`.

  **Evidence**: `.sisyphus/evidence/f3-agent-api-qa.md`
  **Approval condition**: APPROVE only if all API behaviors are verified by command/test output.

- [ ] F4. Scope Fidelity Check — deep

  **Agent-executable checks**:
  - Inspect `rtk git diff` for all changed files.
  - Confirm changed files are limited to planned files or explicitly justified minimal helpers.
  - Confirm no auth, JWT, billing, CDN, S3, transcoding, thumbnails, queues, GraphQL, AI integration, new router, or PostgreSQL downgrade was added.
  - Confirm source commit behavior was adapted to Go conventions without copying the NestJS full-GET bug.
  - Confirm evidence files exist for Tasks 1-4 and F1-F3.

  **Evidence**: `.sisyphus/evidence/f4-scope-fidelity.md`
  **Approval condition**: APPROVE only if implementation remains within scope and all verification evidence exists.

## Commit Strategy
- One atomic commit after all tasks and final validation pass.
- Message: `feat(streaming): add video range streaming endpoint`
- Include only intended files:
  - `cmd/server/main.go`
  - `internal/handler/stream.go`
  - `internal/handler/stream_test.go`
  - `internal/handler/video_integration_test.go` if integration coverage is added there
  - `internal/repository/video.go`
  - `internal/repository/video_test.go`
  - any minimal helper test files required
- Exclude `.sisyphus/evidence/*` from commit unless repository convention requires evidence artifacts.

## Success Criteria
- Source commit behavior is available in Go as `GET /stream/{videoId}`.
- Ranged streaming returns `206` and correct headers/body.
- Full streaming returns `200` and full body.
- Unknown video and missing file return `404`.
- Invalid range returns `416` via stdlib behavior.
- Existing upload tests still pass.
- No PostgreSQL downgrade.
- No new framework dependencies.
