# codex-notifier

`codex-notifier` is a macOS notifier for Codex. The primary mode is `codex app-server` integration: it watches app-server events and sends Japanese notifications for approval-required states and turn completion.

## Quick Start

```bash
go build ./cmd/codex-notifier
./codex-notifier run-local
```

`run-local` starts `codex app-server` in the target worktree, waits for the WebSocket endpoint to become ready, then starts the notifier watcher against that server.

## Main Mode

`run-local` and `watch-app-server` listen to app-server JSON-RPC messages and send notifications for:

- `item/commandExecution/requestApproval`
- `item/fileChange/requestApproval`
- `item/tool/requestUserInput`
- `turn/completed`
- `thread/status/changed` with `waitingOnApproval` as a fallback approval signal

Notifications are shown in Japanese. The notifier is observe-only: it does not approve, deny, or answer requests.

## Commands

```bash
./codex-notifier run-local \
  --worktree /path/to/worktree \
  --notify-on action-required,turn-completed \
  --dedupe-window 30s

./codex-notifier watch-app-server \
  --server ws://127.0.0.1:4500 \
  --notify-on action-required,turn-completed \
  --dedupe-window 30s

./codex-notifier serve \
  --listen 127.0.0.1:8787 \
  --notify-on action-required \
  --dedupe-window 30s

./codex-notifier emit-hook \
  --server http://127.0.0.1:8787/events \
  --event-name stop

./codex-notifier notify \
  --kind approval-pending \
  --summary "これから昇格権限の確認を求めます"

./codex-notifier init
```

### `run-local`

- starts `codex app-server` and the notifier together
- uses the current directory as the default worktree
- chooses an available localhost port automatically unless `--port` is provided
- recommended for `1 worktree = 1 server`

### `watch-app-server`

- `--server`
  - Codex app-server WebSocket URL
- `--notify-on`
  - `action-required`
  - `turn-completed`
- `--dedupe-window`
  - suppress duplicate notifications for the same event key

## Legacy Modes

The following commands remain for compatibility, but they are no longer the primary path:

- `watch-app-server`
  - manual watcher for an already running app-server
- `serve` / `emit-hook`
  - old hook-based HTTP bridge
- `notify`
  - manual notification command
- `init`
  - old setup helper for hook-based flows only

## Notes

- `run-local` assumes the `codex` executable is available in `PATH`, unless `--codex-bin` is provided.
- `watch-app-server` assumes the app-server is already running and reachable.
- Approval-related app-server events depend on current Codex runtime behavior and subscription behavior.
- Notifications use `osascript`.

## Official Docs

- App Server: https://developers.openai.com/codex/app-server
- Agent approvals & security: https://developers.openai.com/codex/agent-approvals-security
- Hooks: https://developers.openai.com/codex/hooks
- AGENTS.md: https://developers.openai.com/codex/guides/agents-md
- Rules: https://developers.openai.com/codex/rules
- Configuration Reference: https://developers.openai.com/codex/config-reference
