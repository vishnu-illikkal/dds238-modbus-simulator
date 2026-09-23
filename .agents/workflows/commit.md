---
description: Stages changes, performs comprehensive diff inspection, formats rich multi-paragraph Conventional Commit messages, and safely commits changes.
---

# High-Quality Conventional Commit Workflow

This workflow ensures every git commit in **Syam's Clinic & Pharmacy Management System** has a comprehensive, informative, and well-structured commit message detailing the **why**, **what**, and **architectural impact** of the changes.

---

## Pre-requisites & Permission Gate
1. **Explicit Permission Required:** Never execute `git commit` unless the user explicitly used `/commit` or gave clear instruction in the current conversation turn.
2. **Quality Gates:** Verify code compiles, `go vet ./...` passes, and tests run cleanly before committing.

---

## Step 1: Inspect Staged & Working Tree Changes
Inspect both file status and detailed diffs:
```bash
git status
git diff --stat
git diff
```

---

## Step 2: Stage Target Files
Stage specific files or all workspace modifications:
```bash
git add .
```
Verify staged diff:
```bash
git diff --cached --stat
```

---

## Step 3: Commit Message Standard & Structure

Every commit message MUST follow this structured, rich multi-line format:

```text
<type>(<scope>): <concise high-level summary (max 72 chars)>

### Why
<1-2 sentences explaining the motivation, business context, or bug root cause>

### Key Changes
- **<Component/Layer 1>**: <Detailed bullet describing what was added/modified>
- **<Component/Layer 2>**: <Detailed bullet describing logic, algorithms, or API routes>
- **<Database/Persistence>**: <Schema changes, migrations, indexes, or queries updated>
- **<UI / Frontend>**: <Visual styling, keyboard shortcuts, modal changes, or UX flow>

### Architectural & Safety Highlights
- <e.g., Transactional atomicity (BeginTx), FEFO ordering guarantee, concurrency mutexes, zero-float financial precision>

### Verification
- <e.g., Unit tests with race detection passed (`go test -race ./...`), manual POS checkout flow tested>
```

### Type Matrix:
- `feat`: New user-facing or domain feature (e.g. `feat(pharmacy): implement FEFO batch auto-deduction`)
- `fix`: Bug fix or calculation error correction (e.g. `fix(billing): resolve round-off discrepancy in multi-tender split`)
- `refactor`: Code restructuring without behavior change (e.g. `refactor(repository): isolate raw SQL queries behind interface`)
- `perf`: Performance optimization (e.g. `perf(inventory): add composite index on medicine_batches for instant lookup`)
- `schema`: Database schema, tables, foreign keys, or migrations (e.g. `schema(procurement): create purchase_orders and grn tables`)
- `docs`: Documentation or workflow configuration updates (e.g. `docs(agents): update clinic skill blueprints`)
- `test`: Adding or updating test suites (e.g. `test(service): add concurrency race tests for batch stock deduction`)

---

## Step 4: Execute Commit with Rich Multi-line Message

Pass multi-paragraph sections using multiple `-m` flags or a formatted string:

```bash
git commit -m "feat(pharmacy): implement FEFO batch deduction engine and stock movement ledger" \
  -m "### Why
Ensures dispensing always prioritizes the nearest-to-expire medicine batches automatically, preventing expired stock write-offs and meeting healthcare regulatory compliance.

### Key Changes
- **Domain Entity**: Added BatchAllocation and StockMovement models with strict validation.
- **Service Logic**: Implemented AllocateBatchesFEFO with row-level locking (FOR UPDATE) inside atomic transactions.
- **Repository**: Added parameterized SQL queries for batch retrieval and stock delta updates.
- **Audit Ledger**: Created immutable stock_movement tracking for all sales, returns, and adjustments.

### Architectural & Safety Highlights
- All batch updates run inside sql.Tx with automatic rollback on insufficient quantity.
- Enforced integer minor units (paise/cents) for MRP and selling prices to avoid float precision loss.

### Verification
- Passed unit tests including TestFEFOAllocation and concurrent race detector (go test -v -race ./...)."
```

---

## Step 5: Semantic Version Tagging

1. Inspect existing tags to find the latest version:
   ```bash
   git tag --sort=-v:refname | head -n 5
   ```
2. Determine next semantic version (`vMajor.Minor.Patch`, e.g., `v0.1.0` -> `v0.1.1`).
3. Create the annotated release tag:
   ```bash
   git tag -a <next_version> -m "Release <next_version>: <short summary>"
   ```
4. Verify the tag is attached:
   ```bash
   git tag -n
   ```