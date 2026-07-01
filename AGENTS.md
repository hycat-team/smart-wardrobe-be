# Conversation Startup

Before performing any task:

- Read this AGENTS.md completely.
- Use this file as the primary navigation guide.
- Read only the documentation required for the current task.
- Never scan the entire `docs/` directory unless explicitly requested.
- Ignore `docs/archive/` unless historical implementations are specifically requested.

---

# Documentation Navigation

Choose documentation according to the task.

General project scope

- docs/project/
- docs/product/

Business rules

- docs/domain/
- docs/flows/

Architecture

- docs/system-design/
- docs/decisions/

API implementation

- docs/api/

Development workflow

- docs/development/

Deployment

- docs/deployment/

Historical reference

- docs/archive/ (only when explicitly requested)

---

# Source of Truth

When documentation and source code are inconsistent:

- Never assume either one is correct.
- Stop implementation.
- Explain the inconsistency.
- Ask the user which source should be treated as authoritative.

Never silently resolve conflicts by guessing.

---

# Makefile

Always prefer Makefile targets over manually typed commands whenever an equivalent target exists.
