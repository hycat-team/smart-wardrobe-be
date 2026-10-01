# Agent Rules & Navigation

## 1. Startup & Navigation

- Read this `AGENTS.md` completely before starting.
- Read only task-relevant docs. Never scan the entire `docs/` directory or read `docs/archive/` unless explicitly requested.
- Choose docs by task category:
    - Scope: `docs/project/`, `docs/product/`
    - Architecture: `docs/system-design/`, `docs/decisions/`
    - API: `docs/api/` | Development: `docs/development/`
- **Skills** are loadable workflows, not documentation:
    - `.opencode/skills/` and `.agents/skills/` — invoke as `@<skill-name>`
    - For a whole feature end to end use `@speckit-orchestrate`. It drives analyze → implement → converge, and owns its own git, gating and verification policy. The individual `@speckit-*` skills remain available for one-off steps.

## 2. Workflow Compliance

- **MANDATORY** — Full compliance with [agent-workflow.md](docs/development/agent-workflow.md):
    - **Large tasks/features**: Follow Spec-kit workflow step-by-step.
    - **Small/trivial tasks** (simple fixes, doc edits, configs): Spec-kit workflow is optional, but you MUST still follow the **core principles** in agent-workflow.md §IV.
    - **Propose and take the next step yourself** instead of handing it back. Stop only where the active workflow's stop conditions say to.
- **Only apply speckit when requested**: If the user does not invoke `@speckit-*`, there is no need to run the speckit workflow, but you MUST still follow the docs in `docs/development/` and `docs/system-design/`.
- **MANDATORY** — Git operations MUST follow [git-workflow.md](docs/development/git-workflow.md):
    - **No autonomous git**: Agent MUST NOT run any git mutation (`commit`, `push`, `tag`, `branch`, `checkout`, `merge`, `rebase`, `reset`, `amend`, history rewrite) without the dev's explicit permission or request. Read-only inspection (`git status/log/diff`) is allowed. This applies even when the dev approved a plan that mentions committing — each git mutation still requires explicit instruction. EXCEPTION: invoking a skill that declares its own git policy (e.g. `@speckit-orchestrate`) is a standing authorization for that run. That skill's policy governs, including its commit cadence and its ask-first list.
    - Commit messages in English, Conventional Commits, commits large enough to avoid noise (see git-workflow.md §1).
    - **Gates scale with the size of the change.** The table below is the single source of truth for gates; do not restate a gate in another document.
        | Change | Gate |
        | --- | --- |
        | Small fix (typo, doc, config, single task) | `make lint` + the tests for the package you touched |
        | One phase of a feature | `make lint` + `go test -race -count=3` scoped to the packages you touched |
        | Before any `push` or `tag` | `make check` in full |

        `make test-race` is the race detector across five workspace modules and `make check` runs seven steps. Neither belongs inside a per-typo loop.

## 3. Code Compliance

- **MANDATORY** — Full compliance with all documents in:
    - `docs/development/` — development process, coding conventions, handler standards, validation, routing, testing, migration
    - `docs/system-design/` — overall architecture, design patterns, event-driven design, AI pipeline, module boundaries
- **MANDATORY** — [code-hygiene.md](docs/development/code-hygiene.md): structural limits (file/function length, struct-per-file, module boundary scan) with thresholds. The `make hygiene-scan` command **MUST ONLY be executed when explicitly permitted or requested by the developer**; NEVER run it autonomously as it generates report files that clutter the source repository.
- When writing code, you **MUST** read and cross-reference the relevant docs for the task before proceeding.
- **MANDATORY** — Environment files: every comment inside any file whose name starts with `.env` (e.g. `.env.example`, `.env.production.example`, `.env.aws.example`) **MUST be written in English**.
- If conflicts arise between documents, priority order: Constitution > system-design > development > other docs.

## 4. Core Principles

- **Constitution Compliance (MANDATORY)**: For **every action** — whether it follows the Speckit workflow or not — you MUST read and comply with [`.specify/memory/constitution.md`](.specify/memory/constitution.md). This is the project's authoritative architectural document. It applies to all tasks: bug fixes, feature development, refactoring, config changes, doc edits, etc. In case of conflict between `constitution.md` and other docs, `constitution.md` takes precedence for architectural boundaries and principles.
- **Conflict Resolution**: If docs and source code disagree on architecture, data model or user-visible behaviour, STOP, explain it, and ask which is authoritative. For implementation detail the documents leave open, decide, then record the decision and its reason in the report.
- **Makefile**: Always prefer Makefile targets over manual commands if equivalent targets exist.
- **Communication language**: Use Vietnamese when communicating with the developer.
- **Ask with the question UI, not prose**: when a decision from the dev is needed, use the `question` tool with concrete options rather than a written question. Each option gets a short label plus a description stating its consequence, so the dev picks in one click instead of typing a reply. Mark the option you prefer with "(Recommended)". Set `multiple: true` only when the choices genuinely combine. Keep prose for open-ended questions that options would flatten.
- **Documents in `specs/`**: Write in Vietnamese per agent-workflow.md §IV.7.
- **Relative Paths**: Always use relative paths instead of absolute paths (e.g., avoid `file:///C:/...` or absolute local paths) when referencing files, writing documentation, specs, and code links.
