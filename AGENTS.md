# AGENTS.md — KafClaw Project Governance

## 1. Project Identity

KafClaw is a personal AI assistant with a Go backend, Electron desktop app, WhatsApp integration, and web dashboard. It connects to LLM providers (OpenAI, OpenRouter) and exposes tools for filesystem, shell, and web operations — all behind a security-first model.

The codebase lives at `github.com/kamir/KafClaw`. Private specs, research, and governance live in `KafClaw-PRIVATE-PARTS` (also mirrored as the gitignored `KafClaw/private/` directory).

---

## 2. Parallel Session Protocol

Two Claude Code sessions work concurrently on KafClaw:

| Session | Focus | Primary Repo |
|---------|-------|--------------|
| **Session A** | Features, tests, code changes | `KafClaw/` |
| **Session B** | Research, docs, specs optimization | `KafClaw-PRIVATE-PARTS/` |

### Coordination Rules

1. **Session A owns the source code.** All Go code, Electron code, CI, and build changes happen in `KafClaw/`.
2. **Session B owns the docs and research.** New specs, requirements, and research happen in `KafClaw-PRIVATE-PARTS/`.
3. **Neither session modifies the other's primary repo directly.** Changes cross the boundary only through the sync scripts.
4. **Sync before starting work.** Each session should pull the latest from the other side before beginning a work block.
5. **Communicate via documents.** If Session B discovers something Session A needs to act on, write it as a requirement or task in `v2/requirements/` or `v2/tasks/`. Session A picks it up after sync.

---

## 3. Sync Workflow

Three sync scripts exist. All are idempotent and safe to re-run. None use `--delete` — both sides can add files independently.

| Script | Location | Direction |
|--------|----------|-----------|
| `sync-private.sh` | `nanobot/migration/` | `KafClaw/private/` → `KafClaw-PRIVATE-PARTS/` |
| `sync-from-kafclaw.sh` | `KafClaw-PRIVATE-PARTS/` | `KafClaw/private/` → `KafClaw-PRIVATE-PARTS/` |
| `sync-to-kafclaw.sh` | `KafClaw-PRIVATE-PARTS/` | `KafClaw-PRIVATE-PARTS/` → `KafClaw/private/` |

### Typical Flow

**Session A** finishes work that touched `private/`:
```bash
cd /path/to/nanobot/migration && ./sync-private.sh
# or from private repo:
cd /path/to/KafClaw-PRIVATE-PARTS && ./sync-from-kafclaw.sh
```

**Session B** finishes work in the private repo:
```bash
cd /path/to/KafClaw-PRIVATE-PARTS && ./sync-to-kafclaw.sh
```

### Conflict Resolution

Since `--delete` is not used, conflicts are additive (both sides may create the same file). If a conflict arises:
1. The file with more recent content wins.
2. If both have meaningful changes, merge manually.
3. Document the resolution in a commit message.

---

## 4. v2 Document Convention

New documents go under `private/v2/`. Legacy migrated docs stay in their original directories as read-only reference.

```
private/
├── v2/                    ← ALL NEW WORK GOES HERE
│   ├── specs/
│   ├── requirements/
│   ├── tasks/
│   ├── research/
│   └── docs/
└── v1/                    ← Legacy (read-only reference)
    ├── specs/
    ├── tasks/
    ├── research/
    ├── docs/
    ├── governance/        ← Legacy AGENTS.md, soul files
    └── requirements/
```

See `private/v2/README.md` for naming conventions.

---

## 5. Source of Truth

- **Code is authoritative.** If documentation contradicts the source code, the documentation is wrong. Fix the documentation.
- **No speculative docs.** Only document what is implemented and merged. Do not document planned features.
- **Same-PR rule.** Documentation updates ship in the same PR as the code change they describe.

---

## 6. Release Hygiene

Before tagging a release, verify:

- [ ] Docs updated for any CLI changes
- [ ] Specs updated for any behavior changes
- [ ] Tests added/updated for new or changed flows
- [ ] `docs/USER_MANUAL.md` reviewed for user-facing changes
- [ ] `docs/OPERATIONS_GUIDE.md` reviewed for API/port/DB changes
- [ ] `docs/ADMIN_GUIDE.md` reviewed for config/security/policy changes

### Documentation Trigger Matrix

| Change Type | USER_MANUAL | OPERATIONS_GUIDE | ADMIN_GUIDE |
|---|---|---|---|
| New/changed CLI command or flag | Yes | — | — |
| New/changed API endpoint | — | Yes | — |
| New/changed config key or env var | — | — | Yes |
| New/changed tool | Yes (usage) | — | Yes (extending) |
| Database schema change | — | Yes | — |
| Security/policy change | — | — | Yes |
| Build/release/CI change | — | Yes | — |
| WhatsApp flow change | Yes | — | Yes |
| Port/network change | — | Yes | — |
| Web dashboard feature | Yes | Yes (API) | — |

---

## 7. Build/Release Issue Log

### BUG-RELEASE-001: electron-builder fails with "Cannot detect repository" (2026-02-16)

**Symptom:** `build-electron` job fails with `Cannot detect repository by .git/config`.

**Fix:** Added `--publish never` to electron-builder command; added `repository` field to `package.json`.

### BUG-RELEASE-002: .deb build fails with "Please specify author email" (2026-02-16)

**Symptom:** `.deb` target requires maintainer email.

**Fix:** Changed `author` in `package.json` from string to object with `name` and `email`.

---

## 8. Version

AGENTS spec: 3.0 — KafClaw Post-Migration Edition
