# KafClaw Architecture — Overview

A quick reference for the KafClaw system architecture. For the comprehensive deep-dive, see [architecture-detailed.md](./architecture-detailed.md).

> See also: [FR-009 System Architecture](../requirements/FR-009-system-architecture.md), [FR-013 Package Design](../requirements/FR-013-package-design.md)

---

## Component Overview

### 1. Core Agent Loop (`internal/agent`)

The heart of the system. Manages the state machine of a conversation:
- **Loop** — Orchestrates interaction between LLM and tools. Agentic loop runs up to 20 iterations per message.
- **Context Builder** — Dynamically assembles the system prompt from soul files, working memory, observations, skills, and RAG context.

### 2. Message Bus (`internal/bus`)

Asynchronous pub-sub decoupling channels from the agent loop:
- **Inbound** — Messages from the outside world (WhatsApp, CLI, Web UI, scheduler).
- **Outbound** — Responses from the agent delivered to channels.
- Buffered channels: 100 inbound, 100 outbound.

### 3. Channels (`internal/channels`)

Interface to the outside world:
- **WhatsApp** — Native Go via `whatsmeow`. No Node.js bridge required.
- **CLI** — Direct terminal interaction.
- **Web UI** — Browser-based chat via dashboard API.

### 4. Tool Framework (`internal/tools`)

The agent's capabilities, gated by a 3-tier policy engine:
- **Filesystem** — Read/write/edit with path safety (writes confined to work repo).
- **Shell** — Execute commands with deny-pattern filtering and strict allow-list.
- **Memory** — Remember/recall via semantic vector search.

### 5. Memory System (`internal/memory`)

6-layer semantic memory with vector embeddings:
- Soul files, conversations, tool results, group sharing, ER1 sync, observations.
- SQLite-vec backend (zero external dependencies).
- Working memory (per-user scratchpad) and observer (LLM compression).

### 6. Provider Layer (`internal/provider`)

LLM abstraction supporting OpenAI-compatible APIs:
- Chat completions, embeddings, transcription (Whisper), TTS.
- Default model: `anthropic/claude-sonnet-4-5` via OpenRouter.

---

## System Diagram

```
WhatsApp ---+
CLI --------+                                 +-- Filesystem Tools
Web UI -----+-- Message Bus -- Agent Loop -- LLM +-- Shell Execution
Scheduler --+       |              |              +-- Web Search
                    |              |              +-- Memory (remember/recall)
                    |         Context Builder
                    |         +-- Soul Files (AGENTS.md, SOUL.md, ...)
                    |         +-- Working Memory (per-user scratchpad)
                    |         +-- Observations (compressed history)
                    |         +-- RAG Injection (vector search)
                    |         +-- Tool + Skill definitions
                    |
                Timeline DB (SQLite)
                +-- Event log
                +-- Memory chunks (embeddings)
                +-- Settings, tasks, approvals
                +-- Group roster, skill channels
                +-- Orchestrator hierarchy
```

## Design Principles

- **Message bus decoupling** — Channels never call the agent directly.
- **Graceful degradation** — Memory, group, orchestrator, ER1 are all optional.
- **Secure defaults** — Binds 127.0.0.1, tier-restricted tools, deny-pattern filtering.
- **Single SQLite database** — All persistent state in `~/.kafclaw/timeline.db`.
