# KafClaw Documentation

Reference documentation for KafClaw — a personal AI assistant framework written in Go with an Electron desktop frontend.

## Guides

| Guide | Description |
|-------|-------------|
| [architecture](./architecture.md) | Quick architecture overview with component diagram |
| [architecture-detailed](./architecture-detailed.md) | Comprehensive system architecture, all packages, memory, API, security |
| [architecture-timeline](./architecture-timeline.md) | Timeline database and memory pipeline design |
| [admin-guide](./admin-guide.md) | Configuration, security model, providers, memory, quotas, extending |
| [operations-guide](./operations-guide.md) | Build, deploy, networking, database, observability, API reference |
| [user-manual](./user-manual.md) | Installation, quick start, CLI, dashboard, WhatsApp, memory, Day2Day, FAQ |
| [security-risks](./security-risks.md) | Threat model, mitigations, best practices |
| [whatsapp-setup](./whatsapp-setup.md) | Default-deny WhatsApp authorization |
| [workspace-policy](./workspace-policy.md) | Fixed workspace path and state isolation |
| [docker-deployment](./docker-deployment.md) | Docker Compose deployment guide |
| [release](./release.md) | Versioning, Make targets, CI/CD |
| [memory-notes](./memory-notes.md) | Personal context notes |

## Related

- [v2 Requirements](../requirements/) — FR-001 through FR-025
- [v2 Tasks](../tasks/) — Active bug reports and task plans
- [v1 Legacy Docs](../../v1/docs/guides/) — Read-only historical reference

## Project Structure

- `KafClaw/` — Go source code + Electron app
- `~/.kafclaw/workspace` — Agent state, sessions, media
- `~/.kafclaw/work-repo` — Agent-generated artifacts
