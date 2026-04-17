# codex-notifier

`codex-notifier` は、通常の `codex` CLI hook から送られたイベントを受け取り、ユーザーの介入が必要なタイミングを macOS 通知へ変換する Go ツールです。

## 対応範囲

- `stop` hook 由来の通知

初期実装は `macOS` 専用です。通知から承認や入力回答は行わず、Codex 側で対応します。

## セットアップ

```bash
go build ./cmd/codex-notifier
```

## 使い方

1. notifier サーバーを起動します。

```bash
./codex-notifier serve --listen 127.0.0.1:8787
```

2. `~/.codex/config.toml` で hook feature を有効化します。

```toml
[features]
codex_hooks = true

[hooks]
config = "/ABSOLUTE/PATH/TO/hooks.json"
```

3. hook 設定ファイルを用意します。`hooks.json` の例:

```json
{
  "stop": [
    {
      "type": "command",
      "command": [
        "/ABSOLUTE/PATH/TO/codex-notifier",
        "emit-hook",
        "--server",
        "http://127.0.0.1:8787/events",
        "--event-name",
        "stop"
      ]
    }
  ]
}
```

4. あとは通常どおり `codex` を起動します。`stop` hook が発火したタイミングで通知が届きます。

## コマンド

```text
./codex-notifier serve \
  --listen 127.0.0.1:8787 \
  --notify-on action-required \
  --dedupe-window 30s

./codex-notifier emit-hook \
  --server http://127.0.0.1:8787/events \
  --event-name stop
```

- `serve --listen`
  - ローカル HTTP サーバーの listen アドレス
- `--notify-on`
  - 通知対象カテゴリ
  - `action-required`
- `--dedupe-window`
  - 同一イベントの重複通知を抑止する時間
- `emit-hook --server`
  - hook イベント送信先の URL
- `emit-hook --event-name`
  - notifier に送る独自イベント名
  - MVP では `stop` を想定

## 注意点

- 通知は app-server の内部イベント互換ではなく、hook から送る独自イベントです。
- `hooks.json` の書式や `config.toml` の hook 設定は、利用している Codex CLI 側の hook 機能に依存します。
- notifier は `127.0.0.1` 向けのローカル受信を前提にしています。
- macOS 通知は `osascript` を使って送信します。
