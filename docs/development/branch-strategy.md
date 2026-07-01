# Git Branch Strategy

The project adopts a streamlined Git Flow workflow:

- `main`: The branch containing the most stable source code running on production.
- `dev`: The main integration branch for the staging environment and general development.
- Feature branches: Created from `dev` with the prefix `feat/` (e.g., `feat/digital-sample-lab`).
- Bugfix branches: Created from `dev` or `main` with the prefix `bugfix/` or `hotfix/`.

Every Pull Request (PR) must be reviewed by at least 1 other member and must pass all CI checks (Lints, Tests, Build) before merging.
