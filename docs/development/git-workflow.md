# Git Workflow

## 1. Agent Git Usage Policy

*   **No autonomous git**: Agent **MUST NOT** run any `git` command (`commit`, `push`, `tag`, `branch`, `checkout`, `merge`, `rebase`, `reset`, `pull`) without explicit developer permission or request. Agent may read `git status/log/diff` for inspection.
*   **English only**: All `commit` messages, `pull request` titles/descriptions, `tag` messages, and related git artifacts **MUST be in English**. Follow Conventional Commits (e.g., `feat:`, `fix:`, `docs:`, `chore:`).
*   **Branch naming**: `feat/<name>`, `fix/<name>`, `chore/<name>` or `###-feature-name` for Spec-kit features.
*   **Commit hygiene**: Commits must be **large enough to be meaningful** — batch related small changes (code + its tests + doc touch-ups) into one commit instead of splitting into fragmented micro-commits; no commit spam (one-line typos, stray file removals, ticked checkboxes alone). A commit should represent a complete, buildable unit of work. `make wire && make test` must pass before commit.

## 2. Tag & Push Tag for Deployment (CI/CD)

Dev tạo tag để trigger deployment trên CI/CD (GitHub Actions). Tag phải theo semver.

**Tạo tag local (annotated):**

```bash
# 1. Đảm bảo nhánh đã push và CI xanh
git checkout main
git pull origin main
git tag -a v1.2.3 -m "release: v1.2.3"

# Hoặc cho pre-release
git tag -a v1.2.3-rc.1 -m "release: v1.2.3-rc.1"
```

**Push tag lên remote để trigger CI/CD:**

```bash
# Push 1 tag
git push origin v1.2.3

# Push nhiều tag
git push origin --tags
```

**Xóa/sửa tag (nếu cần):**

```bash
# Xóa tag local và remote
git tag -d v1.2.3
git push origin :refs/tags/v1.2.3

# Tạo lại và push lại
git tag -a v1.2.3 -m "release: v1.2.3"
git push origin v1.2.3
```

**Lưu ý:**
- Tag `v*` sẽ trigger workflow `deploy.yml` trên CI/CD — chỉ tag khi đã sẵn sàng release.
- Không tạo tag từ nhánh feature, chỉ từ `main` sau khi PR merged.
- Kiểm tra tag đã push: `git ls-remote --tags origin` hoặc GitHub Releases.
