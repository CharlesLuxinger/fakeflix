# Video Upload Service

## TL;DR
> **Summary**: Minimal application (GET / + POST /video file upload) to idiomatic Go using `net/http` stdlib only, preserving exact HTTP semantics.
> **Deliverables**: `cmd/server/main.go`, `internal/handler/health.go`, `internal/handler/video.go`, `internal/service/video.go`, corresponding `_test.go` files, updated `go.mod`, `.gitkeep` in `uploads/`
> **Effort**: Short
> **Parallel**: YES — 2 waves
> **Critical Path**: Task 1 (service) → Task 2 (handlers) → Task 3 (main) → Task 4 (tests) → Final Verification

---

## Context

### Interview Summary
- **Endpoints**: `GET /` → 200 "Hello World!" | `POST /video` → 201 "video uploaded"
- **Upload logic**: Multipart form, 2 required fields: `video` (video/mp4) + `thumbnail` (image/jpeg)
- **Filename**: `{unix_ms_timestamp}-{uuid}{original_ext}`
- **Storage**: `./uploads/` on disk
- **Module**: `github.com/CharlesLuxinger/fakeflix`, Go 1.26.4
- **Tooling**: golangci-lint v2 (strict `.golangci.yml`), Makefile with `lint`/`fmt` targets

### Decisions Made (from Metis review)
| Decision | Choice | Rationale |
|----------|--------|-----------|
| MIME validation source | `Content-Type` header from multipart part | Matches Multer behavior |
| Both files required | YES — missing either → 400 | Original behavior |
| Partial failure cleanup | YES — delete video file if thumbnail save fails | Data consistency |
| Upload dir | Auto-create on startup with `os.MkdirAll` | Matches `dest: './uploads'` Multer config |
| Max multipart size | 32 MB (`r.ParseMultipartForm(32 << 20)`) | Go stdlib default, reasonable for MVP |
| Port | Hardcoded `:3000` | Scope match to original commit |
| Router | `net/http` stdlib only | No framework added — minimal migration |
| Ext source | `filepath.Ext(fh.Filename)` from original filename | Matches Multer behavior |
| UUID dep | `github.com/google/uuid` | Smallest correct dep; crypto-safe |

### Metis Guardrails Incorporated
- MUST NOT use PostgreSQL (Docker Compose has it but this commit doesn't)
- MUST NOT add auth, metadata, video processing, or non-stdlib router
- MUST NOT write outside `./uploads/`
- MUST NOT silently succeed when a required file is missing

---

## Work Objectives

### Deliverables
- `cmd/server/main.go` — entry point, graceful shutdown
- `internal/service/video.go` — SaveFile business logic (testable, no HTTP coupling)
- `internal/handler/health.go` — GET / handler
- `internal/handler/video.go` — POST /video handler
- `internal/service/video_test.go` — unit tests for SaveFile
- `internal/handler/health_test.go` — httptest-based handler tests
- `internal/handler/video_test.go` — httptest-based handler tests
- `go.mod` updated with `github.com/google/uuid`
- `uploads/.gitkeep` — tracks uploads dir in git

### Definition of Done
```
go build ./...                          # exits 0
go test -race ./...                     # all pass, no data races
make lint                               # golangci-lint exits 0
curl http://localhost:3000/             # 200 "Hello World!"
curl -X POST /video (valid mp4+jpeg)   # 201 "video uploaded"
curl -X POST /video (wrong mime)       # 400
curl -X POST /video (missing field)    # 400
```

### Must Have
- Exact HTTP status codes: 200 (GET /), 201 (POST /video success), 400 (validation failure)
- Exact response bodies: `"Hello World!"` and `"video uploaded"`
- Filename pattern: `^\d{13}-[0-9a-f-]{36}\.(mp4|jpg|jpeg)$`
- Saved file bytes must match uploaded bytes
- Partial failure cleanup: if thumbnail save fails after video saved, delete video file
- Graceful shutdown on SIGINT/SIGTERM with 30s timeout

### Must NOT Have (Guardrails)
- No PostgreSQL / database usage
- No authentication or authorization
- No non-stdlib HTTP router (no gin/echo/fiber/chi)
- No video processing or transcoding
- No metadata persistence
- No generic upload abstraction beyond what's needed
- No `context.Context` stored in structs
- No `panic` for error handling
- No global mutable state

---

## Verification Strategy
> ZERO HUMAN INTERVENTION — all verification is agent-executed.

- **Test decision**: tests-after, `testing` stdlib + `net/http/httptest`
- **QA policy**: Every task has agent-executed scenarios via `go test` and `curl`
- **Evidence**: `.sisyphus/evidence/task-{N}-{slug}.txt`

---

## Execution Strategy

### Parallel Execution Waves

**Wave 1** (foundation — no deps on each other):
- Task 1: `internal/service/video.go` + `internal/service/video_test.go` (category: `unspecified-high`)
- Task 2: `uploads/.gitkeep` + update `go.mod` (category: `quick`)

**Wave 2** (depends on Wave 1):
- Task 3: `internal/handler/health.go` + `internal/handler/health_test.go` (category: `quick`)
- Task 4: `internal/handler/video.go` + `internal/handler/video_test.go` (category: `unspecified-high`)

**Wave 3** (depends on Wave 2):
- Task 5: `cmd/server/main.go` — wires everything together (category: `unspecified-high`)

**Wave 4** (final):
- F1–F4: Parallel verification (oracle, code quality, QA, scope fidelity)

### Dependency Matrix
| Task | Depends On | Blocks |
|------|-----------|--------|
| T1 service | — | T4 handler/video |
| T2 go.mod | — | T1, T3, T4, T5 |
| T3 health handler | — | T5 |
| T4 video handler | T1, T2 | T5 |
| T5 main | T3, T4 | F1–F4 |

### Agent Dispatch Summary
| Wave | Tasks | Categories |
|------|-------|-----------|
| 1 | T1 + T2 | unspecified-high + quick |
| 2 | T3 + T4 | quick + unspecified-high |
| 3 | T5 | unspecified-high |
| 4 | F1–F4 | oracle + unspecified-high + unspecified-high + deep |

---

## TODOs

- [ ] 1. Create `internal/service/video.go` + `internal/service/video_test.go`

  **What to do**:
  1. Create file `internal/service/video.go` with package `service`
  2. Define `ErrInvalidMIMEType` struct error type (implements `error` interface) with field `Got string`. Error message: `invalid file type %q: only video/mp4 and image/jpeg are accepted`
  3. Define sentinel constants: `MIMEVideoMP4 = "video/mp4"`, `MIMEImageJPEG = "image/jpeg"`, `uploadsDir = "./uploads"` (unexported)
  4. Define `VideoService` struct with unexported field `uploadsDir string`
  5. `NewVideoService(dir string) *VideoService` constructor
  6. `DefaultVideoService() *VideoService` — calls `NewVideoService("./uploads")`
  7. `(s *VideoService) SaveFile(fh *multipart.FileHeader, allowedMIME string) (string, error)`:
     - Check `fh.Header.Get("Content-Type")` against `allowedMIME`; if mismatch return `&ErrInvalidMIMEType{Got: contentType}`
     - `fh.Open()` → defer close
     - `ext := filepath.Ext(fh.Filename)`
     - `filename := fmt.Sprintf("%d-%s%s", time.Now().UnixMilli(), uuid.New().String(), ext)`
     - `destPath := filepath.Join(s.uploadsDir, filename)`
     - `os.MkdirAll(s.uploadsDir, 0o750)` — return wrapped error on failure
     - `os.Create(destPath)` — return wrapped error on failure; add `//nolint:gosec // path constructed from controlled inputs` comment
     - `dst.ReadFrom(src)` — return wrapped error on failure
     - Return `filename, nil`
  8. Create `internal/service/video_test.go` with package `service_test`:
     - Use `os.TempDir()` for test uploads dir
     - Table-driven tests for `SaveFile`:
       - valid mp4 header → file created, filename matches regex `^\d{13}-[0-9a-f-]{36}\.mp4$`, bytes match
       - valid jpeg header → file created, filename matches regex, bytes match
       - wrong MIME → `ErrInvalidMIMEType` returned, `errors.As` check
     - Use `multipart.FileHeader` constructed via `mime/multipart` writer
     - `t.Parallel()` on test and subtests; `t.TempDir()` for isolated dirs
     - Verify saved file content equals input bytes
  
  **Must NOT do**: No HTTP imports in service package. No global state. No `panic`.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` — Reason: file I/O, multipart, test construction
  - Skills: [`golang-pro`, `golang-patterns`, `golang-code-style`, `golang-testing`]
  - Omitted: `golang-lint` — run separately in final verification

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: T4 | Blocked By: T2 (go.mod needs uuid)

  **References**:
  - Pattern: `.agents/skills/golang-patterns/SKILL.md` — "Accept interfaces, return structs"; error wrapping with `%w`
  - Pattern: `.agents/skills/golang-code-style/SKILL.md` — early returns, `var` for zero values, composite literals with field names
  - Pattern: `.agents/skills/golang-pro/references/testing.md` — table-driven tests, `t.Parallel()`
  - API: `mime/multipart.FileHeader` — https://pkg.go.dev/mime/multipart#FileHeader
  - API: `github.com/google/uuid` — `uuid.New().String()` returns RFC 4122 UUID string
  - Linter config: `.golangci.yml` — `gosec` enabled, hence `#nolint:gosec` with explanation required

  **Acceptance Criteria**:
  - [ ] `go build ./internal/service/...` exits 0
  - [ ] `go test -race ./internal/service/...` all subtests pass
  - [ ] Valid mp4: saved file exists under temp dir, content matches, filename regex `^\d{13}-[0-9a-f-]{36}\.mp4$`
  - [ ] Valid jpeg: saved file exists, filename regex `^\d{13}-[0-9a-f-]{36}\.(jpg|jpeg)$`
   - [ ] Wrong MIME: `var mimeErr *ErrInvalidMIMEType; errors.As(err, &mimeErr)` is true, no file created

  **QA Scenarios**:
  ```
  Scenario: Valid MP4 save
    Tool: go test
    Steps: go test -race -run TestVideoService_SaveFile/valid_mp4 ./internal/service/...
    Expected: PASS; file at tempdir matching regex; bytes equal input
    Evidence: .sisyphus/evidence/task-1-service-tests.txt

  Scenario: Invalid MIME rejection
    Tool: go test
    Steps: go test -race -run TestVideoService_SaveFile/wrong_mime ./internal/service/...
    Expected: PASS; error is *ErrInvalidMIMEType; no file written
    Evidence: .sisyphus/evidence/task-1-service-tests.txt
  ```

  **Commit**: YES | Message: `feat(service): add VideoService with SaveFile and MIME validation` | Files: `internal/service/video.go`, `internal/service/video_test.go`

---

- [ ] 2. Update `go.mod` and create `uploads/.gitkeep`

  **What to do**:
  1. Run `go get github.com/google/uuid@latest` in the project root — this adds the dependency to `go.mod` and creates/updates `go.sum`
  2. Run `go mod tidy` to clean up indirect deps
  3. Create file `uploads/.gitkeep` (empty file) — ensures the uploads directory is tracked by git

  **Must NOT do**: Do not add any other dependencies. Do not modify `.golangci.yml` or `Makefile`.

  **Recommended Agent Profile**:
  - Category: `quick` — Reason: two shell commands + one empty file
  - Skills: [`golang-pro`]
  - Omitted: all others — not needed for dependency management

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: T1, T3, T4, T5 | Blocked By: —

  **References**:
  - Module: `go.mod` — current module is `github.com/CharlesLuxinger/fakeflix`, go `1.26.4`
  - External: https://pkg.go.dev/github.com/google/uuid — `uuid.New().String()` API

  **Acceptance Criteria**:
  - [ ] `go.mod` contains `require github.com/google/uuid v1.x.x`
  - [ ] `go.sum` exists and contains uuid entries
  - [ ] `go mod verify` exits 0
  - [ ] `uploads/.gitkeep` exists (empty file)

  **QA Scenarios**:
  ```
  Scenario: Dependencies resolve
    Tool: Bash
    Steps: cd C:\Users\charl\Projetos\fakeflix && go mod verify
    Expected: "all modules verified" printed, exit 0
    Evidence: .sisyphus/evidence/task-2-gomod.txt
  ```

  **Commit**: YES | Message: `chore(deps): add github.com/google/uuid dependency` | Files: `go.mod`, `go.sum`, `uploads/.gitkeep`

---

- [ ] 3. Create `internal/handler/health.go` + `internal/handler/health_test.go`

  **What to do**:
  1. Create `internal/handler/health.go` with package `handler`
  2. Define `Health` struct (no fields needed)
  3. `NewHealth() *Health` constructor
  4. `(h *Health) ServeHTTP(w http.ResponseWriter, r *http.Request)`:
     - Method guard: if `r.Method != http.MethodGet` → `http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)` and return
     - `w.Header().Set("Content-Type", "text/plain; charset=utf-8")`
     - `w.WriteHeader(http.StatusOK)`
     - `fmt.Fprint(w, "Hello World!")`
  5. Create `internal/handler/health_test.go` with package `handler_test`:
     - Table-driven test `TestHealth_ServeHTTP`
     - Cases: GET / → 200 "Hello World!" | POST / → 405
     - Use `httptest.NewRecorder()` and `httptest.NewRequest()`
     - `t.Parallel()` on test and each subtest

  **Must NOT do**: No business logic in handler. No file I/O. No imports of `internal/service` in this file.

  **Recommended Agent Profile**:
  - Category: `quick` — Reason: trivial handler + test
  - Skills: [`golang-pro`, `golang-code-style`, `golang-testing`]
  - Omitted: `golang-patterns` — not needed for simple handler

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: T5 | Blocked By: T2

  **References**:
  - Pattern: `.agents/skills/golang-code-style/SKILL.md` — early return for method guard, no else after return
  - API: `net/http` — `http.ResponseWriter`, `http.Request`, `http.StatusOK`, `http.StatusMethodNotAllowed`
  - API: `net/http/httptest` — `httptest.NewRecorder()`, `httptest.NewRequest()`

  **Acceptance Criteria**:
  - [ ] `go build ./internal/handler/...` exits 0
  - [ ] `go test -race ./internal/handler/... -run TestHealth` all subtests pass
  - [ ] GET / → status 200, body exactly `"Hello World!"`
  - [ ] POST / → status 405

  **QA Scenarios**:
  ```
  Scenario: GET / returns Hello World
    Tool: go test
    Steps: go test -race -run TestHealth_ServeHTTP/GET_returns_200 ./internal/handler/...
    Expected: PASS; recorder.Code == 200; recorder.Body.String() == "Hello World!"
    Evidence: .sisyphus/evidence/task-3-health-tests.txt

  Scenario: Non-GET returns 405
    Tool: go test
    Steps: go test -race -run TestHealth_ServeHTTP/POST_returns_405 ./internal/handler/...
    Expected: PASS; recorder.Code == 405
    Evidence: .sisyphus/evidence/task-3-health-tests.txt
  ```

  **Commit**: YES | Message: `feat(handler): add Health handler for GET /` | Files: `internal/handler/health.go`, `internal/handler/health_test.go`

---

- [ ] 4. Create `internal/handler/video.go` + `internal/handler/video_test.go`

  **What to do**:
  1. Create `internal/handler/video.go` with package `handler`
  2. Define interface in this file (consumer-side, per Go idiom):
     ```go
     type videoSaver interface {
         SaveFile(fh *multipart.FileHeader, allowedMIME string) (string, error)
     }
     ```
  3. Define `Video` struct with two unexported fields: `saver videoSaver` and `uploadsDir string`
  4. `NewVideo(saver videoSaver, uploadsDir string) *Video` constructor — stores both fields
  5. `(v *Video) ServeHTTP(w http.ResponseWriter, r *http.Request)`:
     - Method guard: if `r.Method != http.MethodPost` → `http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)` and return
     - Parse multipart: `if err := r.ParseMultipartForm(32 << 20); err != nil` → `http.Error(w, "invalid multipart body", http.StatusBadRequest)` and return
     - Get `video` field: `videoFiles := r.MultipartForm.File["video"]`; if `len(videoFiles) == 0` → `http.Error(w, "missing video file", http.StatusBadRequest)` and return
     - Get `thumbnail` field: `thumbFiles := r.MultipartForm.File["thumbnail"]`; if `len(thumbFiles) == 0` → `http.Error(w, "missing thumbnail file", http.StatusBadRequest)` and return
     - Save video: `videoName, err := v.saver.SaveFile(videoFiles[0], "video/mp4")`; on error check with `var mimeErr *service.ErrInvalidMIMEType; if errors.As(err, &mimeErr)`:
       - if MIME error → `http.Error(w, err.Error(), http.StatusBadRequest)` and return
       - else → `http.Error(w, "failed to save video", http.StatusInternalServerError)` and return
     - Save thumbnail: `_, err = v.saver.SaveFile(thumbFiles[0], "image/jpeg")`; on error:
       - cleanup: `os.Remove(filepath.Join(v.uploadsDir, videoName))` using the `v.uploadsDir` field (best-effort, log failure with `log.Printf("cleanup video file: %v", removeErr)`)
       - if MIME error → `http.Error(w, err.Error(), http.StatusBadRequest)` and return
       - else → `http.Error(w, "failed to save thumbnail", http.StatusInternalServerError)` and return
     - `w.WriteHeader(http.StatusCreated)`
     - `fmt.Fprint(w, "video uploaded")`
   6. Create `internal/handler/video_test.go` with package `handler_test`:
      - Define a `mockVideoSaver` struct implementing `videoSaver` interface (local to test file) with fields: `calls []mockSaveCall` (tracking arguments) and `returns []mockSaveReturn` (controlling return values per invocation index)
      - Table-driven test `TestVideo_ServeHTTP`:
        - valid mp4 + valid jpeg → 201 "video uploaded"
        - missing `video` field → 400
        - missing `thumbnail` field → 400
        - video with wrong MIME → 400 (mock first call returns `&service.ErrInvalidMIMEType{Got: "image/jpeg"}`)
        - thumbnail with wrong MIME → 400; verify cleanup behavior: use a real `t.TempDir()` as uploadsDir; mock first SaveFile writes a real file to that dir and returns its name, second SaveFile returns `&service.ErrInvalidMIMEType{}`; after handler returns, assert the video file no longer exists in the temp dir (observable effect, not "mock called")
        - non-POST method → 405
        - invalid multipart body → 400
      - Build multipart request bodies using `mime/multipart.NewWriter`
      - `t.Parallel()` on test and each subtest

  **Must NOT do**: Do not import `os` for the cleanup without handling the error (log it with `log.Printf`). The `uploadsDir` field on `Video` is the single source of truth for cleanup path — do NOT hardcode `"./uploads"` anywhere in the handler. Do not store `context.Context` in struct.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` — Reason: multipart parsing, interface mock, cleanup logic, multiple error paths
  - Skills: [`golang-pro`, `golang-patterns`, `golang-code-style`, `golang-testing`]
  - Omitted: none relevant to skip

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: T5 | Blocked By: T1, T2

  **References**:
  - Pattern: `.agents/skills/golang-patterns/SKILL.md` — "Define interfaces where they're used" (consumer-side interface)
  - Pattern: `.agents/skills/golang-code-style/SKILL.md` — early returns, eliminate unnecessary else
  - API: `mime/multipart` — `r.MultipartForm.File`, `multipart.FileHeader`, `multipart.NewWriter`
  - Type: `internal/service/video.go:ErrInvalidMIMEType` — used for `errors.As` check
  - Linter: `.golangci.yml` — `noctx` linter enabled; all HTTP client calls need context (not applicable here — server side only)
  - External: https://pkg.go.dev/net/http#Request.ParseMultipartForm

  **Acceptance Criteria**:
  - [ ] `go build ./internal/handler/...` exits 0
  - [ ] `go test -race ./internal/handler/... -run TestVideo` all subtests pass
  - [ ] Valid upload: 201, body "video uploaded"
  - [ ] Missing video field: 400
  - [ ] Missing thumbnail field: 400
  - [ ] Wrong video MIME: 400
  - [ ] Wrong thumbnail MIME: 400, video file absent from temp uploadsDir after handler returns
  - [ ] Non-POST: 405

  **QA Scenarios**:
  ```
  Scenario: Valid upload returns 201
    Tool: go test
    Steps: go test -race -run TestVideo_ServeHTTP/valid_mp4_and_jpeg ./internal/handler/...
    Expected: PASS; recorder.Code == 201; recorder.Body.String() == "video uploaded"
    Evidence: .sisyphus/evidence/task-4-video-handler-tests.txt

  Scenario: Missing thumbnail returns 400
    Tool: go test
    Steps: go test -race -run TestVideo_ServeHTTP/missing_thumbnail ./internal/handler/...
    Expected: PASS; recorder.Code == 400
    Evidence: .sisyphus/evidence/task-4-video-handler-tests.txt

  Scenario: Wrong video MIME returns 400
    Tool: go test
    Steps: go test -race -run TestVideo_ServeHTTP/wrong_video_mime ./internal/handler/...
    Expected: PASS; recorder.Code == 400
    Evidence: .sisyphus/evidence/task-4-video-handler-tests.txt
  ```

  **Commit**: YES | Message: `feat(handler): add Video handler for POST /video with MIME validation` | Files: `internal/handler/video.go`, `internal/handler/video_test.go`

---

- [ ] 5. Create `cmd/server/main.go`

  **What to do**:
  1. Create `cmd/server/main.go` with package `main`
  2. Implement `main()` function:
     - `os.MkdirAll("./uploads", 0o750)` — fail fast with `log.Fatalf` if it errors
     - Instantiate: `videoSvc := service.DefaultVideoService()`
     - Instantiate: `healthHandler := handler.NewHealth()`
     - Instantiate: `videoHandler := handler.NewVideo(videoSvc, "./uploads")`
     - Create `mux := http.NewServeMux()`
     - Register: `mux.Handle("GET /", healthHandler)`
     - Register: `mux.Handle("POST /video", videoHandler)`
     - Create server:
       ```go
       srv := &http.Server{
           Addr:         ":3000",
           Handler:      mux,
           ReadTimeout:  30 * time.Second,
           WriteTimeout: 30 * time.Second,
           IdleTimeout:  60 * time.Second,
       }
       ```
     - Start listener in goroutine: `go func() { if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) { log.Fatalf(...) } }()`
     - `log.Printf("server listening on :3000")`
     - Signal wait: `quit := make(chan os.Signal, 1); signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM); <-quit`
     - `log.Printf("shutting down...")`
     - `ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second); defer cancel()`
     - `if err := srv.Shutdown(ctx); err != nil { log.Fatalf("shutdown: %v", err) }`
     - `log.Printf("server stopped")`
  3. NOTE: Use Go 1.22+ method-prefixed mux patterns: `"GET /"` and `"POST /video"` (requires Go 1.22+; module is 1.26.4 ✓)

  **Must NOT do**: No `init()` functions. No global variables (other than `log` package usage which is stdlib). No third-party router. No `panic`.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` — Reason: graceful shutdown pattern, signal handling, wiring
  - Skills: [`golang-pro`, `golang-patterns`]
  - Omitted: `golang-testing` — no tests for main package (integration covered by manual QA)

  **Parallelization**: Can Parallel: NO | Wave 3 | Blocks: F1–F4 | Blocked By: T3, T4

  **References**:
  - Pattern: `.agents/skills/golang-patterns/SKILL.md` — "Graceful Shutdown" section (search for `func GracefulShutdown`). The exact pattern to implement inline in `main()`:
    ```go
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }
    log.Println("Server exited")
    ```
  - Pattern: `.agents/skills/golang-pro/references/project-structure.md` — "cmd/" section describes `cmd/server/main.go` as the correct entry point location
  - API: `net/http` — `http.NewServeMux()`, Go 1.22+ method routing syntax `"METHOD /path"`
  - Type: `internal/service/video.go:DefaultVideoService()` — import path `github.com/CharlesLuxinger/fakeflix/internal/service`
  - Type: `internal/handler/health.go:NewHealth()` — import path `github.com/CharlesLuxinger/fakeflix/internal/handler`
  - Type: `internal/handler/video.go:NewVideo()` — signature `NewVideo(saver videoSaver, uploadsDir string) *Video`; import path `github.com/CharlesLuxinger/fakeflix/internal/handler`

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/server/...` exits 0
  - [ ] `go build ./...` exits 0
  - [ ] Server starts and binds `:3000` (`curl http://localhost:3000/ -s` returns "Hello World!")
  - [ ] `curl -s http://localhost:3000/` → HTTP 200, body "Hello World!"
  - [ ] Valid multipart POST → HTTP 201, body "video uploaded", file exists in `./uploads/`
  - [ ] SIGINT causes graceful shutdown within 30s

  **QA Scenarios** (PowerShell — workspace is Windows/win32):
  ```
  Scenario: GET / returns Hello World
    Tool: Bash (PowerShell + curl)
    Steps: |
      $proc = Start-Process -NoNewWindow -FilePath go -ArgumentList "run","./cmd/server" -PassThru
      Start-Sleep -Seconds 2
      $code = (Invoke-WebRequest -Uri http://localhost:3000/ -UseBasicParsing).StatusCode
      Write-Output $code   # expect 200
      # Cleanup: Send SIGTERM-equivalent and wait for graceful exit
      taskkill /PID $proc.Id 2>$null
      $proc.WaitForExit(30000) | Out-Null
      Write-Output "exit=$($proc.ExitCode)"
    Expected: StatusCode 200, body "Hello World!"; server exits within 30s
    Evidence: .sisyphus/evidence/task-5-main-qa.txt

  Scenario: Valid video upload returns 201
    Tool: Bash (PowerShell + curl)
    Steps: |
      # Create 1KB dummy files (PowerShell)
      [byte[]]$bytes = ,0x00 * 1024
      [System.IO.File]::WriteAllBytes("$env:TEMP\test.mp4", $bytes)
      [System.IO.File]::WriteAllBytes("$env:TEMP\test.jpg", $bytes)
      $response = curl.exe -s -o NUL -w "%{http_code}" `
        -F "video=@$env:TEMP\test.mp4;type=video/mp4" `
        -F "thumbnail=@$env:TEMP\test.jpg;type=image/jpeg" `
        http://localhost:3000/video
      Write-Output $response   # expect 201
    Expected: "201"
    Evidence: .sisyphus/evidence/task-5-main-qa.txt

  Scenario: Wrong MIME returns 400
    Tool: Bash (PowerShell + curl)
    Steps: |
      $response = curl.exe -s -o NUL -w "%{http_code}" `
        -F "video=@$env:TEMP\test.jpg;type=image/jpeg" `
        -F "thumbnail=@$env:TEMP\test.jpg;type=image/jpeg" `
        http://localhost:3000/video
      Write-Output $response   # expect 400
    Expected: "400"
    Evidence: .sisyphus/evidence/task-5-main-qa.txt

  Scenario: Missing thumbnail returns 400
    Tool: Bash (PowerShell + curl)
    Steps: |
      $response = curl.exe -s -o NUL -w "%{http_code}" `
        -F "video=@$env:TEMP\test.mp4;type=video/mp4" `
        http://localhost:3000/video
      Write-Output $response   # expect 400
    Expected: "400"
    Evidence: .sisyphus/evidence/task-5-main-qa.txt
  ```

  **Commit**: YES | Message: `feat(cmd): add server entry point with graceful shutdown` | Files: `cmd/server/main.go`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)
> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**
> **Never mark F1-F4 as checked before getting user's okay.** Rejection or user feedback → fix → re-run → present again → wait for okay.

- [ ] F1. Plan Compliance Audit — oracle

  **What to check**: Verify that the implemented code matches the 2 defined endpoints exactly: `GET /` (200 "Hello World!") and `POST /video` (201 "video uploaded" on success, 400 on MIME error or missing file). Confirm graceful shutdown is present. Confirm all Metis guardrails are respected (no DB, no auth, no non-stdlib router, no video processing).

  **QA Scenario**:
  ```
  Scenario: Audit behavioral parity
    Tool: oracle (code read + reasoning)
    Steps:
      1. Read cmd/server/main.go — confirm port :3000, graceful shutdown via SIGINT/SIGTERM
      2. Read internal/handler/health.go — confirm GET / returns 200 "Hello World!"
      3. Read internal/handler/video.go — confirm POST /video returns 201 "video uploaded" on success, 400 on MIME error or missing field
      4. Read internal/service/video.go — confirm MIME validated from Content-Type header, filename is {unix_ms}-{uuid}{ext}, stored to uploadsDir
      5. Grep codebase for "postgres", "auth", "jwt", "gin", "echo", "fiber", "chi" — expect zero matches
    Expected: All 5 checks pass. Zero guardrail violations.
    Evidence: .sisyphus/evidence/final-f1-compliance.txt
  ```

- [ ] F2. Code Quality Review — unspecified-high

  **What to check**: Run static analysis tools and report all failures.

  **QA Scenario**:
  ```
  Scenario: Lint and vet pass
    Tool: Bash (PowerShell)
    Steps:
      go vet ./...
      make lint
      go test -race ./...
    Expected: All commands exit 0; no lint errors; all tests pass with race detector
    Evidence: .sisyphus/evidence/final-f2-quality.txt
  ```

- [ ] F3. Real Integration QA — unspecified-high

  **What to check**: Start the server and run live HTTP requests using PowerShell curl.

  **QA Scenario**:
  ```
  Scenario: Full integration smoke test (PowerShell)
    Tool: Bash (PowerShell)
    Steps: |
      # 1. Build and start server
      go build -o .\fakeflix.exe .\cmd\server
      $proc = Start-Process -NoNewWindow -FilePath .\fakeflix.exe -PassThru
      Start-Sleep -Seconds 2

      # 2. GET /
      $r1 = Invoke-WebRequest -Uri http://localhost:3000/ -UseBasicParsing
      # Expect: $r1.StatusCode -eq 200; $r1.Content -eq "Hello World!"

      # 3. Create dummy files
      [byte[]]$bytes = ,0x00 * 1024
      [System.IO.File]::WriteAllBytes("$env:TEMP\test.mp4", $bytes)
      [System.IO.File]::WriteAllBytes("$env:TEMP\test.jpg", $bytes)

      # 4. Valid upload POST /video
      $r2 = curl.exe -s -o NUL -w "%{http_code}" `
        -F "video=@$env:TEMP\test.mp4;type=video/mp4" `
        -F "thumbnail=@$env:TEMP\test.jpg;type=image/jpeg" `
        http://localhost:3000/video
      # Expect: $r2 -eq "201"

      # 5. Wrong MIME POST /video
      $r3 = curl.exe -s -o NUL -w "%{http_code}" `
        -F "video=@$env:TEMP\test.jpg;type=image/jpeg" `
        -F "thumbnail=@$env:TEMP\test.jpg;type=image/jpeg" `
        http://localhost:3000/video
      # Expect: $r3 -eq "400"

      # 6. Missing thumbnail POST /video
      $r4 = curl.exe -s -o NUL -w "%{http_code}" `
        -F "video=@$env:TEMP\test.mp4;type=video/mp4" `
        http://localhost:3000/video
      # Expect: $r4 -eq "400"

      # 7. Verify file written to uploads/
      $files = Get-ChildItem -Path .\uploads\ -File
      # Expect: at least 1 file matching \d{13}-[0-9a-f-]{36}\.mp4

      # 8. Graceful shutdown via SIGTERM; process must exit within 30s
      taskkill /PID $proc.Id 2>$null
      $exited = $proc.WaitForExit(30000)
      if (-not $exited) { throw "Server did not shut down within 30s" }
      Write-Output "exit_code=$($proc.ExitCode)"
    Expected: All 7 checks pass; server exits cleanly within 30s of SIGTERM
    Evidence: .sisyphus/evidence/final-f3-integration.txt
  ```

- [ ] F4. Scope Fidelity Check — deep

  **What to check**: Deep read of all source files to confirm nothing outside the original commit's scope was added.

  **QA Scenario**:
  ```
  Scenario: Scope boundary audit
    Tool: deep (code analysis)
    Steps:
      1. Read all .go files under cmd/, internal/ — list every import
      2. Confirm only stdlib packages + github.com/google/uuid are imported
      3. Confirm no SQL, database/sql, GORM, or persistence code
      4. Confirm no JWT, bcrypt, or authentication logic
      5. Confirm no video transcoding, ffmpeg, or media processing
      6. Confirm HTTP status codes match spec: GET / → 200, POST /video success → 201, validation error → 400
      7. Confirm response bodies: GET / → "Hello World!", POST /video → "video uploaded"
    Expected: All 7 checks pass
    Evidence: .sisyphus/evidence/final-f4-scope.txt
  ```

---

## Commit Strategy
Each task commits independently. Final state after all tasks:
```
git log --oneline
feat(cmd): add server entry point with graceful shutdown
feat(handler): add Video handler for POST /video with MIME validation
feat(handler): add Health handler for GET /
chore(deps): add github.com/google/uuid dependency
feat(service): add VideoService with SaveFile and MIME validation
```

---

## Success Criteria
```bash
# All pass:
go build ./...
go test -race ./...
make lint

# Runtime verification:
curl http://localhost:3000/                                    # 200 "Hello World!"
curl -X POST -F "video=@test.mp4;type=video/mp4" \
     -F "thumbnail=@test.jpg;type=image/jpeg" \
     http://localhost:3000/video                              # 201 "video uploaded"
curl -X POST -F "video=@bad.jpg;type=image/jpeg" \
     -F "thumbnail=@test.jpg;type=image/jpeg" \
     http://localhost:3000/video                              # 400
curl -X POST -F "video=@test.mp4;type=video/mp4" \
     http://localhost:3000/video                              # 400 (missing thumbnail)
```
