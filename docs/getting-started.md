# Getting Started

This guide gets KafClaw from zero to a working setup.

## 1. Prerequisites

- Go `1.24+`
- Git
- Optional for group mode: Kafka reachable from your machine
- Optional for desktop: Electron dependencies

## 2. Build

```bash
cd /Users/alo/Development/scalytics/KafClaw
make build
```

Binary target: `./kafclaw`

## 3. Onboard (Mode + LLM)

Run onboarding wizard:

```bash
./kafclaw onboard
```

You will be asked for:

- runtime mode: `local`, `local-kafka`, or `remote`
- LLM setup: `cli-token`, `openai-compatible`, or `skip`

Before writing config, onboarding shows a summary and asks for confirmation.

### Non-interactive examples

Local:

```bash
./kafclaw onboard --non-interactive --profile local --llm skip
```

Local + Kafka:

```bash
./kafclaw onboard --non-interactive --profile local-kafka --kafka-brokers localhost:9092 --group-name kafclaw --agent-id agent-local --role worker --llm skip
```

Remote + Ollama/vLLM:

```bash
./kafclaw onboard --non-interactive --profile remote --llm openai-compatible --llm-api-base http://localhost:11434/v1 --llm-model llama3.1:8b
```

## 4. Verify

```bash
./kafclaw status
./kafclaw doctor
```

Optional hygiene fix:

```bash
./kafclaw doctor --fix
```

This merges discovered env files into `~/.config/kafclaw/env` and enforces mode `600`.

## 5. Run

Local gateway:

```bash
./kafclaw gateway
```

Check:

- API: `http://localhost:18790`
- Dashboard: `http://localhost:18791`

Single prompt test:

```bash
./kafclaw agent -m "hello"
```

## 6. Systemd Setup (Linux)

To install systemd unit/override/env during onboarding:

```bash
sudo ./kafclaw onboard --systemd
```

This can create service user, install unit files, and write runtime env file.

## 7. Where Config Lives

- Main config: `~/.kafclaw/config.json`
- Runtime env: `~/.config/kafclaw/env`
- State DB: `~/.kafclaw/timeline.db`

## Next

- [Operations and Maintenance](./maintenance.md)
- [User Manual](./v2/user-manual.md)
- [Admin Guide](./v2/admin-guide.md)
