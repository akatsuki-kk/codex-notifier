# codex-notifier

`codex-notifier` は Codex CLI の通知補助ツールです。`Stop` hook による完了通知と、`AGENTS.md` から呼ぶ `notify` コマンドによる擬似的な承認前通知を macOS 通知へ変換します。

## 概要

- `notify`
  - `AGENTS.md` の指示から直接呼ぶローカル通知
  - 承認前通知の体験改善向け
- `serve` + `emit-hook`
  - `Stop` hook からイベントを受けて完了通知を送る
  - `127.0.0.1:8787/events` を既定で使用
- `init`
  - `~/.codex` 配下の初期設定を自動生成・更新
  - `AGENTS.md`, `rules/default.rules`, `config.toml`, `hooks.json` を追記マージ

初期実装は `macOS` 専用です。通知から承認や入力回答は行いません。

## クイックスタート

```bash
go build ./cmd/codex-notifier
./codex-notifier init
./codex-notifier serve --listen 127.0.0.1:8787
```

その後 Codex を再起動します。

## コマンド

```bash
./codex-notifier init \
  --codex-home ~/.codex \
  --binary-path /ABSOLUTE/PATH/TO/codex-notifier \
  --server-url http://127.0.0.1:8787/events

./codex-notifier serve \
  --listen 127.0.0.1:8787 \
  --notify-on action-required \
  --dedupe-window 30s

./codex-notifier emit-hook \
  --server http://127.0.0.1:8787/events \
  --event-name stop

./codex-notifier notify \
  --kind approval-pending \
  --summary "About to request elevated permissions" \
  --details "running command outside the sandbox"
```

## `init` が行うこと

- `~/.codex/AGENTS.md`
  - 承認が必要そうな操作の前に `codex-notifier notify` を打つ指示を追加
- `~/.codex/rules/default.rules`
  - `notify` だけを承認なしで実行できる `prefix_rule` を追加
- `~/.codex/config.toml`
  - `[features] codex_hooks = true` を追加または更新
- `~/.codex/hooks.json`
  - `Stop` hook から `emit-hook` を呼ぶ設定を追加または更新

既存ファイルがある場合は内容を残したまま追記マージし、変更前の内容は `.bak` として保存します。

## 運用イメージ

### 手動通知

通常 CLI をそのまま使いたい場合は `notify` を使います。Codex は `AGENTS.md` の指示に従って、承認が必要になりそうな操作の前に次のようなコマンドを実行します。

```bash
./codex-notifier notify \
  --kind approval-pending \
  --summary "About to request elevated permissions"
```

利用できる `kind`:

- `approval-pending`
- `mcp-approval-pending`
- `permission-request-pending`
- `skill-approval-pending`

### `Stop` hook 通知

`serve` を起動しておくと、Codex の `Stop` hook から `emit-hook` 経由でイベントを受け取り、完了通知を送ります。

## 注意点

- `notify` は Codex への指示に依存するため、承認イベントの完全捕捉は保証しません。
- hook 通知は公式の `hooks.json` discovery を前提にしています。
- 通知は `osascript` を使って送信します。
- `init` 実行後は Codex の再起動が必要です。

## 公式ドキュメント

- Hooks: https://developers.openai.com/codex/hooks
- AGENTS.md: https://developers.openai.com/codex/guides/agents-md
- Rules: https://developers.openai.com/codex/rules
- Configuration Reference: https://developers.openai.com/codex/config-reference
