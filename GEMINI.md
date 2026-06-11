# Hariti Project Instructions (GEMINI.md)

This document defines the design principles, implementation boundaries, and development processes for the `go-hariti` repository as actionable Project Instructions.

---

## 1. Foundational Mandates & Authority

### Required Reading
Before making architectural or structural changes, read the following documents:
1. `docs/architecture.md`
2. `docs/graph-ir.md`
3. `docs/generation.md`

These documents are normative.

If additional design documents are referenced from the above documents, read them before proceeding.

Do not make architectural decisions based solely on the current implementation.

### Architecture Authority
Architecture is the source of truth.

When architecture and implementation disagree:

**Architecture wins.**

The current implementation may be incomplete, transitional, under refactoring, or contain historical design debt.

Do not assume the implementation is correct simply because it exists.

### Design Change Policy
If you believe the architecture is incorrect:
1. Explain the issue.
2. Propose an architecture change.
3. Wait for approval.
4. Update the implementation accordingly.

Do not silently modify the implementation model.

Do not introduce new architectural concepts without discussion.

### Hariti Constraints
Hariti is a Vim Runtime Distribution Builder.

Hariti is NOT:
* A Vim Plugin Manager
* A Vim Runtime Framework
* A Runtime Dependency Resolver
* A Vim Startup Automation System

Hariti resolves plugin graphs before Vim startup.

Hariti generates immutable runtime distributions.

Hariti does not perform dependency resolution, VCS operations, build execution, or runtime graph mutation during Vim startup.

The only runtime evaluation currently allowed is `enable_if`.

### Implementation Priority
When making implementation decisions, prioritize:
1. Architectural consistency
2. Responsibility boundaries
3. Reproducibility
4. Operational simplicity
5. Extensibility

Do not introduce complexity solely to increase flexibility.

Do not add abstractions without a clear architectural purpose.

### Refactoring Policy
During refactoring:
* Preserve architectural boundaries.
* Prefer moving responsibilities to the correct layer over adding compatibility code.
* Remove accidental coupling when discovered.
* Keep implementation aligned with the documented architecture.

The goal of refactoring is convergence toward the architecture, not preservation of historical implementation details.

---

## 2. Purpose & Roles

`hariti` is not a simple Vim Plugin Manager. Its primary objective is to resolve a declared Plugin Graph and build a reproducible Vim Runtime Distribution.

### Roles and Boundaries

| Action / Role (What Hariti IS) | Non-Role (What Hariti IS NOT) |
| :--- | :--- |
| **Runtime Distribution Builder**: Generates the Vim Runtime Distribution as a `generation`. | **Vim Runtime Framework**: Providing a framework for the Vim runtime itself. |
| **Plugin Graph Resolver**: Resolves dependencies from the DSL to build the Plugin Graph. | **Vim's Internal Package Manager**: Package management executed inside/during Vim runtime. |
| **Generation Manager**: Manages multiple generations (`generations`) and handles rollbacks, etc. | **Daemon / Resident Process**: Monitoring processes running in the background. |

---

## 3. Execution Principles

### Core Concept
No complex operations must run during Vim execution. Complex operations must be completed before starting Vim.

#### Normal Workflow
1. The developer executes `hariti sync`.
2. A new `generation` is created and laid out.
3. The developer starts `vim`.

### Vim Startup Constraints

| Forbidden | Allowed |
| :--- | :--- |
| Performing the following operations during Vim runtime or from any processes (including asynchronous ones) spawned by Vim is strictly prohibited:<br>・ VCS operations such as `git clone`, `git fetch`, or `git pull`<br>・ Dependency Resolution<br>・ Running build operations (`Build`) <br>・ Runtime Graph Modification | Only evaluating conditions in `enable_if` is allowed.<br>・ `has('python3')`<br>・ `has('win32')`<br>・ `exists('g:vscode')`<br><br>*Note: This simply selects active bundles from a pre-determined Plugin Graph; it does not modify the Graph itself.* |

---

## 4. Terminology & Semantic Definitions

The following concepts must adhere strictly to these semantic definitions throughout the codebase and the DSL parsing logic:

**Dependency**
: The `depends` keyword must always refer to a **Canonical Identity**. Relying on aliases or ambiguous identifiers is prohibited.

**Replace**
: Dependency overrides must be performed via `replace`.
: Examples: `replace mattn/webapi-vim with kamichidu/webapi-vim` or `replace mattn/webapi-vim with local ../webapi-vim`
: `replace` overrides the "implementation" of a node in the Plugin Graph without modifying the Graph's "Identity" itself.

**Alias**
: Intended solely for display purposes and Vim-side identification. Aliases must never be used for resolving dependencies.

**Include**
: Used for splitting configuration files into multiple files to simplify configuration management.
: Example: `include java.hariti`
: `include` is a configuration management utility and must not be used to represent dependencies.

**Lockfile**
: Ensures reproducibility. The Single Source of Truth in a Lockfile is the **Commit Hash** (`revision`). The `tag` is auxiliary metadata.

**Generation**
: Generated Runtime Distributions must always be **Immutable**. When updating a `generation`, do not modify the existing generation's files or directory directly.
: Always create a new directory named after a timestamp (e.g., `generations/20260611-103000/`) and control updates and rollbacks by updating the `current` symbolic link.

**IR (Intermediate Representation)**
: The `hariti` IR represents **only the Plugin Graph**. The IR must not represent `Vimscript`, `RuntimePath`, `Generation Layout`, or `Filesystem Layout`.

---

## 5. Development & Coding Standards

### VCS (Version Control System) Abstraction
Git is the only supported VCS at present, but specific VCS commands must never be tightly coupled with the core codebase. Access must always be mediated through an abstraction layer (Adapter).
- **Forbidden**: Calling `exec.Command("git", ...)` directly from arbitrary parts of the codebase.
- **Recommended**: Interacting through the `VCS` interface and its implementation, `GitAdapter`.

### Phased Implementation Roadmap

Always establish clear boundaries before adding features. Do not compromise or blur architectural boundaries for the sake of feature delivery speed. Follow these implementation phases strictly:

1. Define `Constitution`
2. Design and define `DSL`
3. Implement `Parser`
4. Build the `AST`
5. Implement `Plugin Graph IR`
6. Implement `JSON Dump` (VCS/Git operations must not be implemented before this phase)
7. Implement `Git Adapter`
8. Implement `Lockfile`
9. Implement `Generation`
10. Implement `RuntimePath Projector`

---

## 6. Repository Documentation & Metadata Placement Policy

To prevent documentation bloat, stale metadata, and context window pollution for developers and AI agents, we enforce a strict separation of permanent architecture and operational rules:

### A. Keep in the `docs/` Directory (Permanent Design & Contracts)
The `docs/` directory is reserved for permanent, language-agnostic (where possible) design concepts and contracts:
* **Durable Design Contracts**: System design guarantees, architectural models, and lifecycle policies.
* **Responsibility Boundaries**: Explicit separation of concerns (e.g. 3-Tier Rule).
* **Inter-Layer Inputs and Outputs**: Core interface payloads and event flow definitions.
* **External and LSP Specifications**: Alignment with standardized protocols (e.g. LSP Mapping).
* **Catastrophic Constraints**: Rules and limits that, if violated, would dismantle the system's core design or performance targets.

**Exclusion Rule (No Implementation Diary)**:
To prevent the `docs/` directory from becoming an implementation diary, do not place code-level implementation details in `docs/`. This includes:
* Private helper function names and concrete internal function call sequences.
* Local implementation tricks and temporary compatibility wrappers.
* Current package-internal refactoring notes and test-only structures.
* Implementation-specific helper names (unless they define a durable, public architectural contract).

If such information is useful for current development or AI review, place it in `GEMINI.md` instead (under Section B), and only after explicit human approval.
Conversely, `docs/` may mention concrete protocol types or externally visible contracts when they are part of the design surface (such as LSP methods, JSON-RPC payloads, persistent database concepts, or stable subsystem responsibilities).

### B. Keep in `GEMINI.md` (Operational Guidelines & AI Constraints)
`GEMINI.md` serves as the live instruction manifest for active development, code review, and AI agent execution:
* **Implementation Names**: Specific component, module, or package names.
* **Function and Type Names**: Specific code symbols and core API names.
* **Testing Perspectives & Coverage**: Testing strategy and coverage standards.
* **Review Perspectives**: Guardrails and checkpoints for code reviews.
* **AI Agent Prohibitions & Guidelines**: Directives and constraints for automated tools.
* **Current Operational Conventions**: Practical workflows dependent on the active code structure.

### C. DO NOT Write Anywhere (Anti-Patterns for Docs)
To maintain extreme maintainability, do not write the following in either `docs/` or `GEMINI.md`:
* **Temporary Work Procedures**: Step-by-step setup guides or transient dev instructions.
* **Details Obvious from Code**: Self-documenting functions, basic variables, or obvious flow mechanics.
* **Volatile Internal Names**: Highly localized internal helper names, local variables, or short-lived private symbols.

### D. Documentation Update Decision Flow
Every implementation change must be evaluated for documentation impact:
1. **Permanent Architectural Contract**: Determine whether the change affects a permanent architectural contract.
   - If yes, propose a `docs/` update.
2. **Implementation Constraints / Operations**: Determine whether the change affects implementation constraints, review policies, AI-agent behavior, or operational conventions.
   - If yes, propose a `GEMINI.md` update.
3. **No Impact**: If neither applies, do not update documentation.

### E. GEMINI.md Change Approval
Changes to `GEMINI.md` are treated as constitutional modifications.
AI agents must never modify, append, or remove `GEMINI.md` rules autonomously.
Instead, they must:
* Explicitly explain the proposed change,
* Justify why the current constitution is insufficient,
* Request human approval before applying the modification.

Human approval is strictly required before any `GEMINI.md` update.

---

## 7. Documentation Projection Principle

Architectural documentation must describe intent, contracts, responsibilities, and guarantees.

Architectural documentation must not describe the current implementation mechanics.

Prefer documenting WHY and WHAT.

Avoid documenting HOW.

If a statement can be trivially derived from source code, it probably belongs in code comments, tests, or `GEMINI.md` rather than `docs/`.

---

## 8. Repository Layout

```text
/Users/e-sekito/local/src/github.com/kamichidu/go-hariti/
|-- cmd/
|   `-- hariti/             # Main application entry point
|       |-- main.go
|       `-- subcmd/         # Subcommand implementation files
|-- docs/
|   `-- architecture.md     # Architectural design documentation
|-- encoding/
|   `-- hariti/             # DSL lexing, parsing, and type definitions
|       |-- lexer.go
|       |-- parser.go.y
|       `-- types.go
|-- vcs/
|   `-- git/                # VCS abstraction adapter (Git implementation)
|-- bundle.go               # Bundle representations
|-- context.go              # Context management
|-- hariti.go               # Core logic
`-- vcs.go                  # VCS common interface
```
