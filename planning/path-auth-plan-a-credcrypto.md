# Path Auth — Plan A: credcrypto Module

## Goal
Create the `coordinator/internal/credcrypto` package with encrypt, decrypt, rekey, and the CLI subcommand.

## Tasks
- [ ] 1. Create `coordinator/internal/credcrypto/crypto.go` with `ErrKeyNotSet`, `ErrKeyInvalid`, `LoadKey()`, `Encrypt()`, `Decrypt()` → Verify: `go build ./coordinator/internal/credcrypto/...` passes
- [ ] 2. Create `coordinator/internal/credcrypto/rekey.go` with `Rekey(db, oldKey, newKey)` — full transaction, all rows or none → Verify: compiles cleanly
- [ ] 3. Create `coordinator/internal/credcrypto/crypto_test.go` — round-trip test, `ErrKeyNotSet` when env unset, `ErrKeyInvalid` when key wrong length → Verify: `go test ./coordinator/internal/credcrypto/...` passes
- [ ] 4. Create `coordinator/internal/credcrypto/rekey_test.go` — rekey with in-memory SQLite (happy path + rollback on mid-walk failure) → Verify: all tests pass
- [ ] 5. Add `rekey` subcommand to `coordinator/main.go` — check `os.Args` before HTTP server starts, call `credcrypto.Rekey`, exit 0 on success / exit 1 on error → Verify: `arcvault-coordinator rekey --help` prints usage without starting the server

## Done When
- [ ] `go test ./coordinator/internal/credcrypto/...` — all pass
- [ ] `go build ./coordinator/...` — no errors
- [ ] `arcvault-coordinator rekey --old-key <hex> --new-key <hex>` runs and exits without starting HTTP server
