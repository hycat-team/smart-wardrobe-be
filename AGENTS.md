# Agent Rules & Navigation

## 1. Startup & Navigation

- Read this `AGENTS.md` completely before starting.
- Read only task-relevant docs. Never scan the entire `docs/` directory or read `docs/archive/` unless explicitly requested.
- Choose docs by task category:
    - Scope: `docs/project/`, `docs/product/`
    - Architecture: `docs/system-design/`, `docs/decisions/`
    - API: `docs/api/` | Development: `docs/development/`

## 2. Workflow Compliance

- Strictly comply with [agent-workflow.md](docs/development/agent-workflow.md):
    - **Large tasks/features**: Follow Spec-kit workflow step-by-step.
    - **Small/trivial tasks** (simple fixes, doc edits, configs): Spec-kit is optional.
    - **Always suggest the next command/skill** for the dev to run after completing any workflow step.

## 3. Core Principles

- **Conflict Resolution**: If docs and source code are inconsistent, STOP, explain the conflict, and ask the user which is authoritative. Never guess.
- **Makefile**: Always prefer Makefile targets over manual commands if equivalent targets exist.
