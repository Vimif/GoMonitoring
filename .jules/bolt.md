## 2026-02-13 - Memory Discrepancy in Handlers
**Learning:** The global memory incorrectly stated that `handlers/machine.go` used `strconv` for numerical formatting. In reality, it used inefficient manual string concatenation loops.
**Action:** Always verify code implementation against memory claims before assuming optimizations are already present. Trust the code over the memory.

## 2026-02-13 - Broken Tests in Handlers
**Learning:** `handlers/machines_api_test.go` contained compilation errors (undefined functions like `DeleteMachine` instead of `RemoveMachine`) and logic errors (missing `SetPathValue` for Go 1.22 routing).
**Action:** When running tests for a package, be prepared to fix pre-existing broken tests to ensure a clean baseline.
