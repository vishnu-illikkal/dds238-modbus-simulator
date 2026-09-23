---
description: Stages changes, performs comprehensive diff inspection, formats rich multi-paragraph Conventional Commit messages, and safely commits changes.
---

# High-Quality Conventional Commit Workflow

This workflow ensures every git commit in **Hiking DDS238-2 ZN/S Modbus RTU & TCP Simulator** has a comprehensive, informative, and well-structured commit message detailing the **why**, **what**, and **architectural impact** of the changes, and **automatically increments the semantic version release tag (`vX.Y.Z`) on every commit**.

---

## Pre-requisites & Permission Gate
1. **Explicit Permission Required:** Never execute `git commit` unless the user explicitly used `/commit` or gave clear instruction in the current conversation turn.
2. **Quality Gates:** Verify code compiles (`go vet ./...`) and tests pass with data race detection (`go test -v -race ./...`) before committing.

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
<1-2 sentences explaining the motivation, protocol requirements, or bug root cause>

### Key Changes
- **<Component/Layer 1>**: <Detailed bullet describing what was added/modified>
- **<Component/Layer 2>**: <Detailed bullet describing logic, algorithms, or API routes>
- **<Registers/Modbus Engine>**: <Register map updates, CRC handling, or FC changes>
- **<UI / Frontend>**: <Telemetry dashboard, packet inspector, or code generator improvements>

### Architectural & Safety Highlights
- <e.g., Thread-safe RWMutex state guards, big-endian register framing, strict CRC-16 (0xA001) validation>

### Verification
- <e.g., Unit tests with race detector passed (`go test -v -race ./...`), YAT loopback tested>
```

### Type Matrix:
- `feat`: New user-facing or protocol feature (e.g. `feat(modbus): implement dynamic server ID routing`)
- `fix`: Bug fix or register calculation error (e.g. `fix(meter): correct reactive power signed integer encoding`)
- `refactor`: Code restructuring without behavior change (e.g. `refactor(web): decouple packet inspector broadcaster`)
- `perf`: Performance optimization (e.g. `perf(simulation): optimize physics loop timer ticks`)
- `docs`: Documentation, README, or workflow updates (e.g. `docs(readme): add YAT terminal testing guide`)
- `test`: Adding or updating test suites (e.g. `test(modbus): add multi-meter RTU routing tests`)

---

## Step 4: Execute Commit with Rich Multi-line Message

Pass multi-paragraph sections using multiple `-m` flags:

```bash
git commit -m "feat(web): add 1-click YAT hex copy button and EOL guidance banner" \
  -m "### Why
Eliminates communication errors when users connect serial terminals (like YAT) to the simulator by ensuring EOL (CR/LF) is omitted and frames are properly formatted.

### Key Changes
- **Web UI**: Added Copy for YAT (\h format) button in the client reader sandbox.
- **Documentation**: Added dedicated terminal testing section with ready-to-use Modbus test commands.

### Architectural & Safety Highlights
- Frame hex strings preserve exact CRC-16 bytes calculated by the Modbus protocol engine.

### Verification
- Tested with go test -v -race ./... and verified YAT TCP client connection."
```

---

## Step 5: MANDATORY Semantic Version Tagging

> [!IMPORTANT]
> **NEVER SKIP THIS STEP**: Every `/commit` MUST immediately determine and create the next semantic release tag (`vMajor.Minor.Patch`) so that release tags and badges never lag behind commits.

1. **Inspect existing tags to find the current version**:
   ```bash
   git tag --sort=-v:refname
   ```
2. **Determine next semantic version**:
   - `Patch` (bug fixes, docs, tweaks, minor features): e.g. `v1.0.0` -> `v1.0.1` -> `v1.0.2`
   - `Minor` (new major protocol features, new transport, major UI overhaul): e.g. `v1.0.2` -> `v1.1.0`
   - `Major` (breaking architectural or API changes): e.g. `v1.x.x` -> `v2.0.0`
3. **Create the annotated release tag immediately after commit**:
   ```bash
   git tag -a <next_version> -m "Release <next_version>: <short commit summary>"
   ```
4. **Verify tag attachment**:
   ```bash
   git tag -n -l "<next_version>"
   ```