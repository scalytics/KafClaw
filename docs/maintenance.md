# Operations and Maintenance

Operational runbook for keeping KafClaw healthy in dev and production-like setups.

## Daily Checks

```bash
./kafclaw status
./kafclaw doctor
```

For env hygiene and permissions:

```bash
./kafclaw doctor --fix
```

## Runtime Modes and Expectations

- `local`
  - bind: `127.0.0.1`
  - group/orchestrator: off
- `local-kafka`
  - bind: `127.0.0.1`
  - group/orchestrator: on
- `remote`
  - bind: typically `0.0.0.0` or specific LAN IP
  - auth token: required

If you intentionally run remote, do not force loopback defaults.

## Updating Config Safely

Use CLI config commands:

```bash
./kafclaw config get gateway.host
./kafclaw config set gateway.host "127.0.0.1"
./kafclaw config unset channels.telegram.token
```

Bracket paths are supported:

```bash
./kafclaw config set channels.telegram.allowFrom[0] "alice"
./kafclaw config get channels.telegram.allowFrom[0]
```

## Service Operation (Linux systemd)

If installed as system service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now kafclaw-gateway.service
sudo systemctl status kafclaw-gateway.service
```

Restart after config changes:

```bash
sudo systemctl restart kafclaw-gateway.service
```

## Backup

Minimum backup set:

- `~/.kafclaw/config.json`
- `~/.config/kafclaw/env`
- `~/.kafclaw/timeline.db`
- workspace and work-repo dirs used by your config

## Upgrade Flow

```bash
git pull
make build
go test ./...
```

Then restart runtime/service.

## Troubleshooting Quick Map

- Gateway unreachable from network:
  - check `gateway.host`
  - check firewall ports `18790`, `18791`
- Remote mode blocked:
  - ensure auth token exists
- Kafka group issues:
  - verify brokers and group config
  - use `kafclaw kshark` for diagnostics
- LLM errors:
  - verify API base, model, token in config/env

## Deep References

- [Operations Guide](./v2/operations-guide.md)
- [Admin Guide](./v2/admin-guide.md)
- [Security Risks](./v2/security-risks.md)
