# Go Application Development Standards: Best Practices & Security Rules

When generating, refactoring, or reviewing Go application code, strictly enforce these rules:

## 1. Idiomatic Design & Architecture
- **Accept Interfaces, Return Concrete Types:** Declare interfaces in consumer packages, keeping them small (1-3 methods). Return concrete structs.
- **Clean Architecture & Layout:** Follow the standard Go layout (`cmd/`, `internal/domain`, `internal/service`, `internal/repository`, `internal/handler`). Keep entity domains free of external dependencies.
- **Avoid Package Stuttering:** Never name types like `user.UserService` (use `user.Service` or `user.Handler`).
- **Constructors & Dependencies:** Use explicit constructors `NewService(...)` with constructor injection.

## 2. Error Handling
- **Never Ignore Errors:** Do not use `_ = err` or discard errors returned by functions.
- **Error Wrapping:** Wrap with `%w` for upstream context (`fmt.Errorf("fetching order %s: %w", id, err)`).
- **Inspection:** Use `errors.Is(err, target)` and `errors.As(err, &customErr)`.
- **No Panics:** Do not panic in HTTP handlers or business logic. Recover safely if panics can occur in worker pools.

## 3. Concurrency & Context Safety
- **Goroutine Ownership:** Every goroutine must have an explicit exit trigger (context cancellation, channel closure, or `errgroup.Group`).
- **Context First:** Always pass `ctx context.Context` as the first argument in I/O, database, and network functions. Never store context inside a struct.
- **Race Condition Prevention:** Protect shared mutable memory with `sync.Mutex` or `sync/atomic`. Always test with `go test -race ./...`.
- **Channel Ownership:** Senders own and close channels; receivers never close channels.

## 4. Security & Hardening (OWASP Compliance)
- **SQL Injection:** Mandatory parameterized queries with `database/sql` (`$1`, `?`). Never format/concatenate SQL queries.
- **Crypto & Randomness:** Use `crypto/rand` for tokens, UUIDs, salts, and session IDs. Never use `math/rand` for security.
- **Password Hashing:** Use `golang.org/x/crypto/bcrypt` (cost >= 12) or `argon2id`.
- **Server Timeouts:** Always configure `http.Server` with explicit `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` to prevent Slowloris attacks.
- **Request Body Limits:** Always limit payload sizes with `http.MaxBytesReader(w, r.Body, maxBytes)` before parsing JSON/multipart.
- **Path Traversal:** Clean file paths with `filepath.Clean` and verify target paths start with the expected base directory prefix.
- **Constant-Time Comparison:** Use `crypto/subtle.ConstantTimeCompare` for API keys, hashes, and authentication tokens.

## 5. Resource Management & Performance
- **Always Close Resources:** Ensure `defer resp.Body.Close()`, `defer file.Close()`, and `defer rows.Close()` are called immediately after error checks.
- **Drain Response Bodies:** Drain `io.Copy(io.Discard, resp.Body)` before closing to enable TCP connection reuse.
- **Preallocate Slices:** Use `make([]T, 0, count)` when length or capacity is known in advance.

## 6. Observability & Logging
- **Structured Logging:** Use Go 1.21+ `log/slog` with JSON handler in production. Do not use unstructured `fmt.Println` or `log.Printf`.
- **Context Propagation:** Include correlation IDs (`request_id`, `trace_id`) in log contexts.
