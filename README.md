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
  - 実行権限を先に確認し、権限が足りない場合だけ `codex-notifier notify` を打つ指示を追加
- `~/.codex/rules/default.rules`
  - `notify` だけを承認なしで実行できる `prefix_rule` を追加
- `~/.codex/config.toml`
  - `[features] codex_hooks = true` を追加または更新
- `~/.codex/hooks.json`
  - `Stop` hook から `emit-hook` を呼ぶ設定を追加または更新

既存ファイルがある場合は内容を残したまま追記マージし、変更前の内容は `.bak` として保存します。

## 運用イメージ

### 手動通知

通常 CLI をそのまま使いたい場合は `notify` を使います。Codex は `AGENTS.md` の指示に従って、まず対象コマンドやツール呼び出しに実行権限があるかを確認します。追加のユーザー確認が必要な場合だけ、`--kind` と `--summary` を指定して通知します。追加する instruction 文面は英語で統一し、通知メッセージそのものは日本語で書く前提です。

```bash
./codex-notifier notify \
  --kind approval-pending \
  --summary "これから昇格権限の確認を求めます"
```

`--kind` の指定方法:

- `approval-pending`
  - コマンド実行権限が足りず、一般的な追加承認が必要なとき
- `mcp-approval-pending`
  - MCP ツール利用に対する確認が必要なとき
- `permission-request-pending`
  - `request_permissions` による権限要求を出す直前
- `skill-approval-pending`
  - skill 実行に確認が必要なとき

`--summary` の指定方法:

- 1 行の短い日本語文で「これから何の確認を出すか」を書く
- 1 行の短い日本語文で「権限がないため、これから何の確認を出すか」を書く
- instruction 自体は英語で追加し、`--summary` の本文だけ日本語にする
- 例:
  - `これから昇格権限の確認を求めます`
  - `これから MCP postgres_lm_local の利用確認を求めます`
  - `これからネットワークアクセス権限の確認を求めます`

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
