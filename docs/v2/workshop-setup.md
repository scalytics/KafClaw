# Workshop Setup — 4-Agent Group Deployment

Deploy a KafClaw Workshop: 1 desktop Electron agent (Mac) + 3 headless Docker agents sharing one gateway and one Kafka bus.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Host Machine (Mac)                   │
│                                                          │
│  ┌──────────────────────┐    ┌────────────────────────┐ │
│  │   Electron App       │    │   Docker Compose        │ │
│  │   (desktop-orch.)    │    │                          │ │
│  │                      │    │  Kafka + Zookeeper       │ │
│  │  Gateway: 18790/91   │    │  KafScale LFS Proxy     │ │
│  │  WhatsApp channel    │    │                          │ │
│  │  Dashboard UI        │    │  agent-researcher        │ │
│  │  Role: orchestrator  │    │  agent-coder             │ │
│  │                      │    │  agent-reviewer           │ │
│  └──────────────────────┘    └────────────────────────┘ │
│         │                              │                 │
│         └──────── Kafka (9092) ────────┘                 │
└─────────────────────────────────────────────────────────┘
```

## Prerequisites

- Docker + Docker Compose v2
- Git
- Go 1.24+ (for building the Electron app / local gateway)
- An OpenRouter API key

## Quick Start

```bash
cd gomikrobot

# Interactive setup — creates .env, builds image, starts stack
make workshop-setup

# Or manual steps:
cp .env.example .env        # edit with your API key
make docker-build            # build kafclaw:local image
make workshop-up             # start Kafka + 3 agents
```

## Configuration

All configuration is via the `.env` file (see `.env.example` for all options).

### Required variables

| Variable | Description |
|----------|-------------|
| `OPENROUTER_API_KEY` | Shared LLM key for all agents |
| `AGENT_AUTH_TOKEN` | Random secret for headless agent APIs |

### Agent work repos

Each agent can mount a host directory or auto-clone a Git URL:

```bash
# Mount a local directory
AGENT1_WORK_REPO=/Users/you/repos/research-repo

# Or auto-clone from GitHub (needs GH auth on host)
AGENT1_WORK_REPO_GIT_URL=https://github.com/you/research-repo.git
```

### Agent identity (SOUL.md)

Each agent's personality is defined by its workspace `SOUL.md`. Example templates are in `examples/workshop/`:

```
examples/workshop/
├── agent-researcher/workspace/SOUL.md
├── agent-coder/workspace/SOUL.md
└── agent-reviewer/workspace/SOUL.md
```

Customize these before starting the stack.

## Desktop Agent (Electron / CLI)

The Mac runs as the 4th agent and orchestrator. Configure group settings in `~/.gomikrobot/config.json`:

```json
{
  "group": {
    "enabled": true,
    "groupName": "workshop",
    "agentId": "desktop-orchestrator",
    "kafkaBrokers": "localhost:9092",
    "lfsProxyUrl": "http://localhost:8080"
  },
  "orchestrator": {
    "enabled": true,
    "role": "orchestrator"
  }
}
```

Then start:

```bash
make run-full             # CLI gateway
# or
make electron-start-full  # Electron desktop app
```

## Operations

```bash
make workshop-up     # Start all containers
make workshop-down   # Stop all containers
make workshop-logs   # Tail logs
make workshop-ps     # Container status
```

## How It Works

1. **Kafka** is the shared message bus. All agents publish and consume from group topics.
2. **KafScale LFS Proxy** bridges large file transfers over Kafka.
3. **Headless agents** run `gomikrobot gateway` in Docker with `MIKROBOT_GATEWAY_HOST=0.0.0.0`. Their ports are not exposed to the host — they communicate only via Kafka.
4. **The desktop agent** exposes ports 18790 (API) and 18791 (dashboard) and owns the WhatsApp channel.
5. On startup, each agent auto-scaffolds its workspace (soul files) if missing, and auto-joins the group specified in its config.

## Verification

1. Check all containers are running: `make workshop-ps`
2. Check agent logs for group join: `make workshop-logs`
3. Open dashboard at `http://localhost:18791` — group roster should show 4 agents
4. Send a message via the dashboard — it should be delegated to a headless agent

## Troubleshooting

| Issue | Fix |
|-------|-----|
| Kafka not healthy | Wait 30s for startup, then check: `docker compose -f docker-compose.group.yml logs kafka` |
| Agent can't join group | Verify `MIKROBOT_GROUP_KAFKA_BROKERS` points to `kafka:29092` (internal) for Docker agents, `localhost:9092` for host |
| Work repo empty | Set `WORK_REPO_GIT_URL` in `.env` or pre-populate the mounted directory |
| Soul files missing | Agent auto-scaffolds defaults on startup. Customize `examples/workshop/*/workspace/SOUL.md` |
