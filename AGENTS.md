# Agent Rules & Navigation

## 1. Startup & Navigation

- Read this `AGENTS.md` completely before starting.
- Read only task-relevant docs. Never scan the entire `docs/` directory or read `docs/archive/` unless explicitly requested.
- Choose docs by task category:
    - Scope: `docs/project/`, `docs/product/`
    - Architecture: `docs/system-design/`, `docs/decisions/`
    - API: `docs/api/` | Development: `docs/development/`

## 2. Workflow Compliance

- **MANDATORY** — Full compliance with [agent-workflow.md](docs/development/agent-workflow.md):
    - **Large tasks/features**: Follow Spec-kit workflow step-by-step.
    - **Small/trivial tasks** (simple fixes, doc edits, configs): Spec-kit workflow is optional, but you MUST still follow the **core principles** in agent-workflow.md §IV.
    - **Always suggest the next command/skill** for the dev to run after completing any workflow step.
- **Only apply speckit when requested**: If the user does not invoke `@[/speckit-*]`, there is no need to run the speckit workflow, but you MUST still follow the docs in `docs/development/` and `docs/system-design/`.
- **MANDATORY** — Git operations MUST follow [git-workflow.md](docs/development/git-workflow.md):
    - **No autonomous git**: Agent MUST NOT run any git mutation (`commit`, `push`, `tag`, `branch`, `checkout`, `merge`, `rebase`, `reset`, `amend`, history rewrite) without the dev's explicit permission or request. Read-only inspection (`git status/log/diff`) is allowed. This applies even when the dev approved a plan that mentions committing — each git mutation still requires explicit instruction.
    - Commit messages in English, Conventional Commits, commits large enough to avoid noise (see git-workflow.md §1).
    - Before proposing a commit: `make wire && make lint && make test-race` must pass.

## 3. Code Compliance

- **MANDATORY** — Full compliance with all documents in:
    - `docs/development/` — development process, coding conventions, handler standards, validation, routing, testing, migration
    - `docs/system-design/` — overall architecture, design patterns, event-driven design, AI pipeline, module boundaries
- **MANDATORY** — [code-hygiene.md](docs/development/code-hygiene.md): structural limits (file/function length, struct-per-file, module boundary scan) with thresholds and the `make hygiene-scan` workflow. Run `make hygiene-scan` after structural refactors and before proposing commits for hygiene-sensitive changes.
- When writing code, you **MUST** read and cross-reference the relevant docs for the task before proceeding.
- If conflicts arise between documents, priority order: Constitution > system-design > development > other docs.

## 4. Core Principles

- **Constitution Compliance (MANDATORY)**: For **every action** — whether it follows the Speckit workflow or not — you MUST read and comply with [`.specify/memory/constitution.md`](.specify/memory/constitution.md). This is the project's authoritative architectural document. It applies to all tasks: bug fixes, feature development, refactoring, config changes, doc edits, etc. In case of conflict between `constitution.md` and other docs, `constitution.md` takes precedence for architectural boundaries and principles.
- **Conflict Resolution**: If docs and source code are inconsistent, STOP, explain the conflict, and ask the user which is authoritative. Never guess.
- **Makefile**: Always prefer Makefile targets over manual commands if equivalent targets exist.
- **Communication language**: Use Vietnamese when communicating with the developer.
- **Documents in `specs/`**: Write in Vietnamese per agent-workflow.md §IV.7.
- **Relative Paths**: Always use relative paths instead of absolute paths (e.g., avoid `file:///C:/...` or absolute local paths) when referencing files, writing documentation, specs, and code links.
