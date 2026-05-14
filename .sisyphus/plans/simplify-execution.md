# Simplify `internal/roskit/execution/` — Remove Dead Complexity

## TL;DR

> **Quick Summary**: Remove healthLoop, IsAlive, HealthInterval, and related dead code from the execution layer. Extract `isRouterOSPermanentError` to a shared package, add "missing =" pattern, apply it to poll worker. Add real-router integration tests for connection persistence.
> 
> **Deliverables**:
> - Clean, compilable `execution/` package with watchAsync-only reconnect
> - Shared `isRouterOSPermanentError` used by both stream and poll workers
> - Poll worker stops on permanent RouterOS errors instead of retrying forever
> - Real-router integration tests for connect, reconnect, lifecycle, concurrent streams+queries
> - All healthLoop/IsAlive/HealthInterval/healthCtx/healthCancel removed completely
> 
> **Estimated Effort**: Medium (6 tasks across 3 waves)
> **Parallel Execution**: YES — 3 waves
> **Critical Path**: Task 1 → Task 3 → Task 5

---

## Context

### Original Request
User discovered that the execution layer in `internal/roskit/` had unnecessary complexity (healthLoop, keepAlive, ForceReconnect, IsAlive) added as workarounds for each other. The root cause was identified: `watchAsync` (via go-routeros `Async()` error channel) already provides reliable disconnect detection, making all the extra mechanisms redundant and buggy.

### Interview Summary
**Key Discussions**:
- healthLoop race condition bug: `conn.client.Close()` sets `closed=true`, then `ConnectWithBackoff()` always fails → router stuck disconnected
- keepAlive removed in previous work — poll commands create periodic traffic, OS TCP keepalive suffices
- ForceReconnect removed in previous work — was workaround for healthLoop bug
- User wants to keep 2-connection model (client+stream) for Queue isolation safety
- User wants real-router integration tests (no mocks) — user will provide RouterOS credentials

**Research Findings**:
- go-routeros/v3 `Async()` returns `<-chan error` that closes on disconnect → watchAsync sufficient
- `ListenArgsQueueContext()` and `RunContext()` both use tag multiplexing → can coexist on same connection
- Two connections exist for Queue buffer isolation (100 vs 500) — user chose to keep this
- Stream worker already has `isRouterOSPermanentError` — poll worker does NOT
- "missing =interface=" error from RouterOS not caught by current permanent error patterns

### Metis Review
**Identified Gaps** (addressed):
- Poll worker should STOP on permanent error (like stream worker) for consistency, not just log and retry → Added to Task 3
- "missing =" pattern may be too generic → Use "missing =" to catch the format, but must also check for "no such command prefix" and "no such command or directory" (current patterns)
- Integration tests must clean up after themselves → Added cleanup requirement to Task 5
- Concurrent Borrow/BorrowAsync safety must be tested → Added to Task 5

---

## Work Objectives

### Core Objective
Remove dead complexity from `execution/` (healthLoop, IsAlive, HealthInterval, healthCtx, healthCancel), extract shared error detection, add it to poll worker, and verify everything works with real-router integration tests.

### Concrete Deliverables
- `internal/roskit/execution/conn.go` — IsAlive removed, watchAsync-only reconnect
- `internal/roskit/execution/pool.go` — healthLoop, IsAlive, healthCtx, healthCancel, HealthInterval removed; Start/LaunchOne cleaned up
- `internal/roskit/execution/option.go` — unchanged (keep RoleClient/RoleStream)
- `internal/roskit/core/errors.go` — new shared `IsRouterOSPermanentError` function
- `internal/roskit/behavior/stream/worker.go` — uses shared `core.IsRouterOSPermanentError`
- `internal/roskit/behavior/poll/worker.go` — uses shared `core.IsRouterOSPermanentError`, stops on permanent errors
- `internal/roskit/execution/conn_test.go` — TestConn_IsAlive removed, new real-router tests added
- `internal/roskit/execution/helpers_test.go` — HealthInterval removed
- `internal/roskit/testhelpers/router.go` — HealthInterval removed

### Definition of Done
- [x] `go build ./...` passes with zero errors
- [x] `go vet ./...` passes
- [x] No references to `healthLoop`, `IsAlive`, `HealthInterval`, `healthCtx`, `healthCancel` remain in codebase
- [x] `isRouterOSPermanentError` lives in shared package, not stream/worker.go
- [x] `isRouterOSPermanentError` matches "missing =" pattern
- [x] Poll worker stops on permanent RouterOS errors (not just log+retry)
- [x] Integration tests pass with `//go:build mikrotik` tag
- [x] Integration tests verify: connect, disconnect+reconnect, concurrent stream+query, pool lifecycle

### Must Have
- All healthLoop/IsAlive/HealthInterval/healthCtx/healthCancel code removed
- Shared `IsRouterOSPermanentError` in `core/` package
- Poll worker uses permanent error detection
- Compilation passes (`go build ./...`)
- Real-router integration tests exist

### Must NOT Have (Guardrails)
- NO mock-based tests — user explicitly wants real RouterOS
- NO collapsing of 2-connection model — keep client+stream as-is
- NO changes to Borrow/BorrowAsync/RouterConn structure — keep the routing
- NO changes to watchAsync/reconnect mechanics — they work correctly
- NO changes outside `internal/roskit/` except test helper cleanup
- NO changes to SSE keepAlive (in API handlers) — that's unrelated
- NO AI-slop: excessive comments, over-abstraction, unnecessary generics

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (`//go:build mikrotik` tag pattern already in conn_test.go)
- **Automated tests**: YES (tests-after) — integration tests for execution layer
- **Framework**: Go standard `testing` + `testify` (already in use)
- **Test tag**: `//go:build mikrotik` — real router required

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Go build**: Use `go build ./...` — must compile cleanly
- **Go vet**: Use `go vet ./...` — must pass
- **Grep verification**: Use grep to verify no references to removed code remain
- **Integration tests**: Use `MIKROTIK_HOST=... MIOKROTIK_USER=... MIKROTIK_PASS=... go test -tags=mikrotik -race ./internal/roskit/.../`

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — foundation + cleanup):
├── Task 1: Remove healthLoop/IsAlive/HealthInterval dead code [quick]
├── Task 2: Extract isRouterOSPermanentError to shared package [quick]
└── Task 3: Add permanent error detection to poll worker [quick]

Wave 2 (After Wave 1 — integration tests):
├── Task 4: Add real-router integration tests for connection lifecycle [deep]
└── Task 5: Verify full system works with docker [unspecified-high]

Wave FINAL (After ALL tasks — 4 parallel reviews):
├── F1: Plan compliance audit (oracle)
├── F2: Code quality review (unspecified-high)
├── F3: Real manual QA (unspecified-high)
└── F4: Scope fidelity check (deep)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|-----------|--------|
| 1    | —         | 3, 4, 5 |
| 2    | —         | 3, 4    |
| 3    | 1, 2      | 4, 5    |
| 4    | 1, 2, 3  | 5       |
| 5    | 1, 2, 3, 4 | F1-F4  |
| F1-F4 | 5       | —       |

### Agent Dispatch Summary

- **Wave 1**: 3 tasks — T1 → `quick`, T2 → `quick`, T3 → `quick`
- **Wave 2**: 2 tasks — T4 → `deep`, T5 → `unspecified-high`
- **Wave FINAL**: 4 tasks — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. Remove healthLoop/IsAlive/HealthInterval dead code from execution/

  **What to do**:
  - In `internal/roskit/execution/pool.go`:
    - Remove `healthCtx context.Context` and `healthCancel context.CancelFunc` fields from Pool struct (lines 164-165)
    - Remove `healthCtx, cancel := context.WithCancel(ctx)` and related lines in `Start()` (lines 208-212)
    - Remove `go p.healthLoop(healthCtx, conn)` calls in `Start()` (line 218) and `LaunchOne()` (line 316)
    - Remove `cancel := p.healthCancel` and `if cancel != nil { cancel() }` in `Stop()` (lines 225-234 simplification — just remove the cancel part, keep the Close() loop)
    - Remove `healthCtx := p.healthCtx` in `LaunchOne()` (line 307)
    - Remove `RouterConn.IsAlive()` method (lines 129-132)
    - Remove `HealthInterval time.Duration` field from ConnConfig (line 27)
    - Remove `HealthInterval` default setting in `withDefaults()` (lines 40-42)
  - In `internal/roskit/execution/conn.go`:
    - Verify `IsAlive` method is already removed (from previous uncommitted changes). If not, remove it.
  - In `internal/roskit/execution/conn_test.go`:
    - Remove `TestConn_IsAlive_Connected` test function (lines 62-70)
  - In `internal/roskit/execution/helpers_test.go`:
    - Remove `HealthInterval: 60 * time.Second` from `loadRouterConfig()` (line 44)
  - In `internal/roskit/testhelpers/router.go`:
    - Remove `HealthInterval: 60 * time.Second` from `TestRouterConfig()` (line 42)
  - Run `go build ./...` to verify compilation passes
  - Run `go vet ./...` to verify no issues
  - Grep for `healthLoop|IsAlive|HealthInterval|healthCtx|healthCancel` in `internal/roskit/execution/` — must return zero results

  **Must NOT do**:
  - Do NOT remove `watchAsync`, `reconnect`, `Connect`, `ConnectWithBackoff`, `Close`, `State`, `getConn` methods
  - Do NOT remove `RoleClient`, `RoleStream`, `DefaultRoleConfigs` from option.go
  - Do NOT change Borrow/BorrowAsync/RouterConn structure
  - Do NOT remove SSE keepAlive in API handlers (that's unrelated)
  - Do NOT add comments to code that had none (NO_COMMENTS rule from AGENTS.md)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Surgical removal of dead code with clear line-by-line instructions
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: Tasks 3, 4, 5
  - **Blocked By**: None (can start immediately)

  **References**:
  **Pattern References**:
  - `internal/roskit/execution/pool.go:27` — HealthInterval field in ConnConfig (to remove)
  - `internal/roskit/execution/pool.go:40-42` — HealthInterval default in withDefaults (to remove)
  - `internal/roskit/execution/pool.go:129-132` — RouterConn.IsAlive method (to remove)
  - `internal/roskit/execution/pool.go:164-165` — healthCtx, healthCancel fields (to remove)
  - `internal/roskit/execution/pool.go:208-212` — healthCtx initialization in Start() (to remove)
  - `internal/roskit/execution/pool.go:218` — `go p.healthLoop(healthCtx, conn)` call (to remove)
  - `internal/roskit/execution/pool.go:225-234` — Stop() cancel logic (to simplify)
  - `internal/roskit/execution/pool.go:307` — healthCtx in LaunchOne() (to remove)
  - `internal/roskit/execution/pool.go:316` — `go p.healthLoop(healthCtx, conn)` call in LaunchOne() (to remove)
  - `internal/roskit/execution/conn.go` — IsAlive method already removed in working tree
  - `internal/roskit/execution/conn_test.go:62-70` — TestConn_IsAlive_Connected (to remove)
  - `internal/roskit/execution/helpers_test.go:44` — HealthInterval in test config (to remove)
  - `internal/roskit/testhelpers/router.go:42` — HealthInterval in test helpers (to remove)

  **Why Each Reference Matters**:
  - pool.go lines are the exact dead code locations that must be surgically removed
  - conn_test.go and helpers contain IsAlive test and HealthInterval config that reference removed features
  - testhelpers/router.go is a shared test helper used by integration tests

  **Acceptance Criteria**:
  - [ ] `go build ./...` passes
  - [ ] `go vet ./...` passes
  - [ ] `grep -r 'healthLoop\|IsAlive\|HealthInterval\|healthCtx\|healthCancel' internal/roskit/execution/` returns zero results
  - [ ] `grep -r 'IsAlive' internal/roskit/testhelpers/` returns zero results (excluding non-execution IsAlive)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Build passes after dead code removal
    Tool: Bash
    Preconditions: All edits applied
    Steps:
      1. Run `go build ./...`
      2. Verify exit code 0
    Expected Result: Build succeeds with no errors
    Failure Indicators: Compilation errors referencing removed functions/fields
    Evidence: .sisyphus/evidence/task-1-build-passes.txt

  Scenario: No dead code references remain
    Tool: Bash
    Preconditions: Build passes
    Steps:
      1. Run `grep -rn 'healthLoop\|IsAlive\|HealthInterval\|healthCtx\|healthCancel' internal/roskit/execution/ internal/roskit/testhelpers/`
      2. Verify output is empty
    Expected Result: Zero matches found
    Failure Indicators: Any matches found — dead code not fully removed
    Evidence: .sisyphus/evidence/task-1-dead-code-grep.txt
  ```

  **Commit**: YES (groups with Tasks 2, 3)
  - Message: `refactor(execution): remove healthLoop, IsAlive, HealthInterval dead code`
  - Files: pool.go, conn.go, conn_test.go, helpers_test.go, testhelpers/router.go
  - Pre-commit: `go build ./... && go vet ./...`

- [x] 2. Extract isRouterOSPermanentError to shared core package

  **What to do**:
  - Create `internal/roskit/core/errors.go` with the shared `IsRouterOSPermanentError` function:
    ```go
    package core

    import "strings"

    // IsRouterOSPermanentError returns true for errors that indicate a RouterOS
    // command or feature does not exist on the target device. These errors will
    // never resolve without a firmware/package change, so retrying is pointless.
    func IsRouterOSPermanentError(err error) bool {
        if err == nil {
            return false
        }
        s := err.Error()
        return strings.Contains(s, "no such command prefix") ||
            strings.Contains(s, "no such command or directory") ||
            strings.Contains(s, "missing =")
    }
    ```
  - Update `internal/roskit/behavior/stream/worker.go`:
    - Remove the local `isRouterOSPermanentError` function (lines 202-212)
    - Add import `"github.com/quixxiq/roskit/internal/roskit/core"`
    - Change `isRouterOSPermanentError(err)` call (line 95) to `core.IsRouterOSPermanentError(err)`
  - Run `go build ./...` to verify

  **Must NOT do**:
  - Do NOT change the stream worker's retry/backoff logic
  - Do NOT change the function behavior (only extract + add "missing =" pattern)
  - Do NOT add comments beyond the existing doc comment (NO_COMMENTS rule)
  - Do NOT make it a method — keep it as a package-level function

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple extract function + add one pattern + update one import
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: Tasks 3, 4
  - **Blocked By**: None (can start immediately)

  **References**:
  **Pattern References**:
  - `internal/roskit/behavior/stream/worker.go:202-212` — current `isRouterOSPermanentError` function to extract
  - `internal/roskit/behavior/stream/worker.go:95` — call site to update to `core.IsRouterOSPermanentError`

  **API/Type References**:
  - `internal/roskit/core/` — package where the new `errors.go` file will live
  - Check that `internal/roskit/core/` directory exists (it should — it has command/, definition/, parser/, model/)

  **Why Each Reference Matters**:
  - The exact function and call site need to be moved with zero behavior change
  - The "missing =" pattern must be added to catch `interface_traffic` stream errors like "missing =interface="

  **Acceptance Criteria**:
  - [ ] `internal/roskit/core/errors.go` exists and exports `IsRouterOSPermanentError`
  - [ ] `IsRouterOSPermanentError` includes "missing =" pattern
  - [ ] `stream/worker.go` no longer has local `isRouterOSPermanentError`
  - [ ] `stream/worker.go` imports `core` and calls `core.IsRouterOSPermanentError`
  - [ ] `go build ./...` passes

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Shared function exists and is used
    Tool: Bash
    Preconditions: All edits applied
    Steps:
      1. Run `grep -n 'IsRouterOSPermanentError' internal/roskit/core/errors.go`
      2. Verify function is exported (capital I) and contains "missing ="
      3. Run `grep -n 'isRouterOSPermanentError' internal/roskit/behavior/stream/worker.go`
      4. Verify zero matches (local function removed)
      5. Run `grep -n 'core.IsRouterOSPermanentError' internal/roskit/behavior/stream/worker.go`
      6. Verify exactly 1 match (call site updated)
    Expected Result: function extracted, local removed, shared called
    Failure Indicators: local function still exists, or call site not updated
    Evidence: .sisyphus/evidence/task-2-shared-error-check.txt

  Scenario: Build passes after extraction
    Tool: Bash
    Steps:
      1. Run `go build ./...`
    Expected Result: Exit code 0
    Failure Indicators: Import errors or undefined references
    Evidence: .sisyphus/evidence/task-2-build-passes.txt
  ```

  **Commit**: YES (groups with Tasks 1, 3)
  - Message: `refactor(core): extract IsRouterOSPermanentError to shared package`
  - Files: core/errors.go, behavior/stream/worker.go
  - Pre-commit: `go build ./...`

- [x] 3. Add permanent error detection to poll worker

  **What to do**:
  - In `internal/roskit/behavior/poll/worker.go`:
    - Add import `"github.com/quixxiq/roskit/internal/roskit/core"`
    - In `doPoll()` method, after the `conn.RunContext(...)` error check (line 73), add permanent error detection:
      ```go
      if core.IsRouterOSPermanentError(err) {
          w.logger.Info("poll worker stopping: RouterOS feature unavailable",
              "router_id", routerID,
              "measurement", meta.Measurement,
              "error", err,
          )
          return // stop this poll worker entirely
      }
      ```
    - This matches the stream worker pattern (lines 95-101 in stream/worker.go) — permanent error means stop, not retry
  - In the `Start()` method, the loop checks `ctx.Done()` — when `doPoll` returns nil due to permanent error, `Start()` should also return nil. Verify this is the case by reading the loop logic (lines 39-51). If `doPoll` returns, the ticker loop continues — need to propagate stop signal.
  
  **IMPORTANT**: The poll worker's `Start()` method uses `time.NewTicker` in a for/select loop. After calling `doPoll()`, it continues to the next ticker iteration. If `doPoll` encounters a permanent error and returns early from `Start()`, we need to make the stop signal propagate. Currently `doPoll` logs and returns (doesn't stop the loop). Two approaches:
  - Option A: Return a `bool` (shouldStop) from `doPoll` and check it in `Start()`
  - Option B: Use `ctx.Err()` pattern — cancel the context from within `doPoll`
  
  **Recommended**: Option A — have `doPoll` return `(stop bool)` and in `Start()`, if `doPoll` returns true, return nil (stop the worker). This is the simplest change matching the stream worker pattern.

  Implementation:
  ```go
  // In Start():
  w.doPoll(ctx, routerID, meta)  // initial poll
  
  ticker := time.NewTicker(interval)
  defer ticker.Stop()
  
  for {
      select {
      case <-ctx.Done():
          w.logger.Info("poll worker stopped", ...)
          return nil
      case <-ticker.C:
          if w.doPoll(ctx, routerID, meta) {
              w.logger.Info("poll worker stopping: permanent error", ...)
              return nil
          }
      }
  }
  
  // doPoll signature change:
  func (w *Worker) doPoll(ctx context.Context, routerID string, meta *command.CommandMeta) (shouldStop bool) {
      // ... existing borrow logic ...
      if core.IsRouterOSPermanentError(err) {
          w.logger.Info("poll worker: RouterOS feature unavailable", ...)
          return true  // permanent error, stop
      }
      // ... existing error handling ...
      return false  // continue polling
  }
  ```

  - Run `go build ./...` to verify

  **Must NOT do**:
  - Do NOT change the poll interval or backoff logic
  - Do NOT change the stream worker (already correct)
  - Do NOT add retry logic for permanent errors — they are permanent
  - Do NOT modify any other behavior handlers (mutation, query)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Small focused change to one file with clear pattern to follow
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on Task 2 for core.IsRouterOSPermanentError)
  - **Parallel Group**: Wave 1 (sequential within — after Task 2)
  - **Blocks**: Tasks 4, 5
  - **Blocked By**: Tasks 1, 2

  **References**:
  **Pattern References**:
  - `internal/roskit/behavior/stream/worker.go:94-101` — stream worker permanent error pattern to follow
  - `internal/roskit/behavior/poll/worker.go:53-103` — full poll worker `doPoll` method to modify
  - `internal/roskit/behavior/poll/worker.go:26-51` — poll worker `Start` method loop to modify

  **Why Each Reference Matters**:
  - Stream worker shows the canonical pattern: permanent error → log + return nil
  - Poll worker's doPoll needs the same pattern adapted for poll semantics (return bool instead of error)

  **Acceptance Criteria**:
  - [ ] `poll/worker.go` imports `"github.com/quixxiq/roskit/internal/roskit/core"`
  - [ ] `poll/worker.go` uses `core.IsRouterOSPermanentError(err)` in doPoll
  - [ ] `doPoll` returns `(shouldStop bool)` — `true` on permanent error, `false` otherwise
  - [ ] `Start()` checks `doPoll` return value and stops on `true`
  - [ ] `go build ./...` passes

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Poll worker stops on permanent error
    Tool: Bash
    Preconditions: Code compiles
    Steps:
      1. Read poll/worker.go doPoll method
      2. Verify it calls core.IsRouterOSPermanentError
      3. Verify it returns true when permanent error detected
      4. Read Start() method
      5. Verify it checks doPoll return and stops when true
    Expected Result: Poll worker matches stream worker pattern for permanent errors
    Failure Indicators: No IsRouterOSPermanentError call, or doPoll doesn't return bool
    Evidence: .sisyphus/evidence/task-3-poll-perm-error.txt

  Scenario: Build passes
    Tool: Bash
    Steps:
      1. Run `go build ./...`
    Expected Result: Exit code 0
    Failure Indicators: Compilation errors
    Evidence: .sisyphus/evidence/task-3-build-passes.txt
  ```

  **Commit**: YES (groups with Tasks 1, 2)
  - Message: `fix(poll): add permanent error detection to poll worker`
  - Files: behavior/poll/worker.go
  - Pre-commit: `go build ./...`

- [x] 4. Add real-router integration tests for connection lifecycle

  **What to do**:
  - Add integration tests to `internal/roskit/execution/conn_test.go` (the file already exists with `//go:build mikrotik` tag pattern)
  - All new tests use `//go:build mikrotik` build tag and `skipWithoutMikroTik(t)` helper
  - Test cases to add:

  **Test 1: `TestPersistentConn_ConnectAndRun`**
  - Create `PersistentConn` with `RoleClient`, connect, run `/system/identity/print`, verify reply is non-empty, close
  - Verifies: basic connect → command → close lifecycle works

  **Test 2: `TestPersistentConn_ReconnectAfterDisconnect`**
  - Create `PersistentConn`, connect, verify `State() == ConnStateConnected`
  - Close the underlying `routeros.Client` directly (simulating disconnect) — get client via `pc.Client()`, call `client.Close()`
  - Wait for `watchAsync` to detect disconnect and trigger `reconnect()`
  - Poll `State()` in a loop (1s interval, 15s timeout) until `ConnStateConnected`
  - Run `/system/identity/print` again to verify the connection works after reconnect
  - Close PersistentConn
  - Verifies: watchAsync detects disconnect → reconnect happens → connection is usable again

  **Test 3: `TestRouterConn_ConcurrentBorrowBorrowAsync`**
  - Create a `Pool`, register a router config, `Start()`, wait for connected
  - Use `Borrow()` to get `RouterConn` client, run `/system/identity/print` via `RunContext()`
  - Use `BorrowAsync()` to get stream `PersistentConn`, run `/interface/print` via `RunContext()`
  - Verify both work concurrently without error
  - Clean up pool
  - Verifies: client and stream connections both work simultaneously

  **Test 4: `TestPool_RegisterUnregisterReconnect`**
  - Create a `Pool`, register a router config, `Start()`, wait for connected
  - Verify `Status()` shows `ConnStateConnected`
  - `Unregister()` the router
  - Verify `Borrow()` returns error (router not registered)
  - Re-register the same router config, call `LaunchOne()`, wait for connected
  - Verify `Borrow()` works and can run commands
  - Clean up pool
  - Verifies: pool lifecycle — register, connect, unregister, re-register works

  **Test 5: `TestPool_MultipleConcurrentStreams`**
  - Create a `Pool`, register a router, `Start()`, wait for connected
  - Start 2-3 concurrent goroutines using `BorrowAsync()` + `ListenArgsQueueContext` for different commands (e.g., `/interface/print`)
  - Verify each stream gets at least one reply
  - Also run a query via `Borrow()` + `RunContext()` concurrently with the streams
  - Verify the query succeeds while streams are active
  - Clean up pool
  - Verifies: tag multiplexing works — multiple streams + queries on same connections

  - Run `go build ./...` to verify all tests compile
  - Note: Tests will NOT run in CI (they require `//go:build mikrotik` tag + real router). User will run them with: `MIKROTIK_HOST=192.168.233.1:8728 MIKROTIK_USER=admin MIKROTIK_PASS=r00t go test -tags=mikrotik -race -v ./internal/roskit/execution/`

  **Must NOT do**:
  - Do NOT use mocks — these are real-router integration tests
  - Do NOT add tests for healthLoop/IsAlive (they're being removed)
  - Do NOT add tests that don't use the `//go:build mikrotik` tag
  - Do NOT modify the existing `TestConn_Connect_Success`, `TestConn_Connect_WrongHost`, `TestConn_RunContext_IdentityPrint`, `TestConn_Close_State`, `TestRouterConn_ConnectBoth` tests — keep them as-is
  - Do NOT hardcode router credentials — use `loadRouterConfig()` and env vars

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Integration tests require careful lifecycle management (connect, disconnect, reconnect, concurrent) with real RouterOS device semantics
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on Tasks 1, 2, 3)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 5
  - **Blocked By**: Tasks 1, 2, 3

  **References**:
  **Pattern References**:
  - `internal/roskit/execution/conn_test.go:1-83` — existing test file structure, build tag, helpers
  - `internal/roskit/execution/helpers_test.go:1-81` — `skipWithoutMikroTik()`, `loadRouterConfig()`, `newTestPool()` helpers
  - `internal/roskit/behavior/stream/worker.go:94-101` — stream worker's use of pool.BorrowAsync for listening

  **API/Type References**:
  - `internal/roskit/execution/conn.go:54-82` — `Connect()` method: dial, async, watchAsync
  - `internal/roskit/execution/conn.go:124-132` — `getConn()` for running commands
  - `internal/roskit/execution/conn.go:162-184` — `watchAsync()` — disconnect detection
  - `internal/roskit/execution/conn.go:186-192` — `reconnect()` — auto-reconnect with backoff
  - `internal/roskit/execution/pool.go:240-255` — `Borrow()` — get client connection
  - `internal/roskit/execution/pool.go:286-299` — `BorrowAsync()` — get stream connection

  **External References**:
  - go-routeros/v3: `Client.ListenArgsQueueContext(ctx, sentence, queueSize)` — returns `*ListenReply`, call `.Chan()` for event stream
  - go-routeros/v3: `Client.RunContext(ctx, sentence...)` — returns `*Reply` with `.Re` slice of sentences
  - go-routeros/v3: `Client.Close()` — sends `/quit` to RouterOS

  **Why Each Reference Matters**:
  - Existing test file shows the build tag, helper pattern, and assertion style to follow
  - conn.go methods show the exact API being tested
  - Pool methods show how Borrow/BorrowAsync work for concurrent access testing

  **Acceptance Criteria**:
  - [ ] `TestPersistentConn_ConnectAndRun` test exists in conn_test.go
  - [ ] `TestPersistentConn_ReconnectAfterDisconnect` test exists in conn_test.go
  - [ ] `TestRouterConn_ConcurrentBorrowBorrowAsync` test exists in conn_test.go
  - [ ] `TestPool_RegisterUnregisterReconnect` test exists in conn_test.go
  - [ ] `TestPool_MultipleConcurrentStreams` test exists in conn_test.go
  - [ ] All tests use `//go:build mikrotik` tag and `skipWithoutMikroTik(t)` helper
  - [ ] All tests use `loadRouterConfig()` for credentials (not hardcoded)
  - [ ] `go build ./...` passes (tests compile)

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Integration tests compile
    Tool: Bash
    Preconditions: All edits applied
    Steps:
      1. Run `go build ./...`
      2. Verify exit code 0
    Expected Result: All tests compile without errors
    Failure Indicators: Compilation errors in test file
    Evidence: .sisyphus/evidence/task-4-build-passes.txt

  Scenario: Test file structure is correct
    Tool: Bash
    Preconditions: Build passes
    Steps:
      1. Run `head -5 internal/roskit/execution/conn_test.go`
      2. Verify line 1 is `//go:build mikrotik`
      3. Run `grep -c 'func Test' internal/roskit/execution/conn_test.go`
      4. Verify count is at least 10 (5 existing + 5 new)
    Expected Result: Build tag present, all 5 new tests exist
    Failure Indicators: Missing build tag or missing test functions
    Evidence: .sisyphus/evidence/task-4-test-structure.txt
  ```

  **Commit**: YES
  - Message: `test(execution): add real-router integration tests`
  - Files: execution/conn_test.go
  - Pre-commit: `go build ./...`

- [x] 5. Verify full system works with docker

  **What to do**:
  - Build the API binary: `go build -o bin/api ./cmd/api`
  - Start docker dev stack: `make docker-up` (PostgreSQL, Redis)
  - Run the API server: `go run ./cmd/api`
  - Verify in logs:
    - Router connections establish correctly (no healthLoop, no IsAlive errors)
    - Stream workers start and connect (no permanent error noise for supported commands)
    - Poll workers start and connect
    - If a router disconnects (simulate by restarting MikroTik), watchAsync detects it and reconnect happens
  - If `MIKROTIK_HOST` is available, run: `MIKROTIK_HOST=192.168.233.1:8728 MIKROTIK_USER=admin MIKROTIK_PASS=r00t go test -tags=mikrotik -race -v ./internal/roskit/execution/`
  - Verify all integration tests pass
  - Stop the API server and docker stack
  - Capture evidence: log output showing successful connection, reconnect behavior, and test results

  **Must NOT do**:
  - Do NOT modify application code — this is verification only
  - Do NOT modify docker-compose or dev config
  - Do NOT hardcode credentials in test commands

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Requires running system, docker, and careful log analysis
  - **Skills**: `[]`

  **Parallelization**:
  - **Can Run In Parallel**: NO (depends on ALL previous tasks)
  - **Parallel Group**: Wave 2 (after Task 4)
  - **Blocks**: F1-F4
  - **Blocked By**: Tasks 1, 2, 3, 4

  **References**:
  **Pattern References**:
  - `Makefile` — build targets: `make build-api`, `make docker-up`
  - `docker/docker-compose.dev.yml` — dev stack config
  - `cmd/api/main.go` — API server entry point

  **Why Each Reference Matters**:
  - Need to know how to build and run the system for end-to-end verification

  **Acceptance Criteria**:
  - [ ] `go build -o bin/api ./cmd/api` succeeds
  - [ ] API server starts without healthLoop/IsAlive errors in logs
  - [ ] Router connections establish and show "connected" in logs
  - [ ] No references to removed features in running logs (no healthLoop, no IsAlive)
  - [ ] If MikroTik available: integration tests all pass

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: Build and start system
    Tool: Bash
    Preconditions: All code changes applied
    Steps:
      1. Run `go build -o bin/api ./cmd/api`
      2. Verify exit code 0
      3. Run `make docker-up` (start PostgreSQL, Redis)
      4. Run the API server briefly (5-10 seconds)
      5. Check logs for "connecting" / "connected" messages
      6. Check logs for ABSENCE of "healthLoop" / "IsAlive" messages
    Expected Result: System starts, routers connect, no dead-code references in logs
    Failure Indicators: Build fails, or healthLoop/IsAlive references appear in logs
    Evidence: .sisyphus/evidence/task-5-system-start.txt

  Scenario: No dead code references in source
    Tool: Bash
    Steps:
      1. Run `grep -rn 'healthLoop\|IsAlive\|HealthInterval\|healthCtx\|healthCancel' internal/roskit/`
      2. Verify zero matches
      3. Run `grep -rn 'isRouterOSPermanentError' internal/roskit/behavior/stream/worker.go`
      4. Verify zero matches (should be core.IsRouterOSPermanentError)
      5. Run `grep -rn 'core.IsRouterOSPermanentError' internal/roskit/behavior/`
      6. Verify both stream/worker.go and poll/worker.go have the import
    Expected Result: Dead code gone, shared function used in both workers
    Failure Indicators: Any matches for dead code, or missing shared function usage
    Evidence: .sisyphus/evidence/task-5-final-grep.txt
  ```

  **Commit**: NO (verification only, no code changes)
  - Evidence only

---

## Final Verification Wave

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in `.sisyphus/evidence/`. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `go build ./...` + `go vet ./...` + `go test -race ./...`. Review all changed files for: `as any`/type assertions without checks, empty catches, fmt.Println in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Build the binary. Verify: `go build ./...` passes; `grep -r 'healthLoop\|IsAlive\|HealthInterval\|healthCtx\|healthCancel' internal/roskit/execution/` returns zero results; `isRouterOSPermanentError` exists in `core/errors.go`; poll worker imports and uses it.
  Output: `Build [PASS/FAIL] | Dead code [GONE/REMAINS] | Shared error [EXISTS/MISSING] | Poll check [PRESENT/MISSING] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **1**: `refactor(execution): remove healthLoop, IsAlive, HealthInterval dead code` — conn.go, pool.go, option.go, conn_test.go, helpers_test.go, testhelpers/router.go
- **2**: `refactor(core): extract isRouterOSPermanentError to shared package` — core/errors.go, behavior/stream/worker.go
- **3**: `fix(poll): add permanent error detection to poll worker` — behavior/poll/worker.go
- **4**: `test(execution): add real-router integration tests` — execution/conn_test.go
- **5**: `chore(docker): verify full system works after simplification` — evidence only

---

## Success Criteria

### Verification Commands
```bash
go build ./...                    # Must compile cleanly
go vet ./...                      # Must pass
grep -r 'healthLoop\|IsAlive\|HealthInterval\|healthCtx\|healthCancel' internal/roskit/execution/  # Must return nothing
grep -r 'isRouterOSPermanentError' internal/roskit/behavior/poll/worker.go  # Must match
grep -r 'IsRouterOSPermanentError' internal/roskit/core/  # Must match
```

### Final Checklist
- [x] All "Must Have" present
- [x] All "Must NOT Have" absent
- [x] All tests pass