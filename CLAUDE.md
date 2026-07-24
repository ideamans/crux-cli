# CLAUDE.md — crux-cli

Chrome UX Report (CrUX) のフィールドデータを引く CLI。**バイナリ名は `crux`**、
goreleaser のプロジェクト名も `crux`（リポジトリ名だけ `crux-cli`）。

データソースが2系統ある（BigQuery = `device` / CrUX API = `history`・`record`）
点がこの CLI の本質で、取り違えると数値の意味が変わる。

## 変更時の必須手順

**機能を追加した、フラグを増やした、既存の挙動を変えた — このいずれかをしたら、
3か所すべてを更新してから終わること。**

| 更新先 | 対象 | やり方 |
| --- | --- | --- |
| ① ドキュメント | `README.md` / `README_ja.md` | 使い方が変わったときのみ |
| ② ヘルプ | cobra の `Short` / `Long` / フラグ説明 | コード内。**カタログはここから生成される** |
| ③ **LLMナレッジ** | `internal/llmdocs/00-guide.md` | データソースの選び方・認証が変わったら |
| | `internal/llmdocs/10-metrics.md` | 指標・しきい値が増減したら |
| | `internal/llmdocs/20-schemas.md` | **JSON 出力のフィールドを変えたら必ず** |
| | `internal/llmdocs/30-gotchas.md` | 制限値（50 origins、40 periods、レート制限）や罠が変わったら |
| | `internal/llmdocs/90-commands.md` | **生成物。手編集しない** → `go generate ./...` |
| | `plugins/crux-cli/skills/*/SKILL.md` | 手順や前提が変わったとき |
| | `context7.json` の `rules` | 新しい落とし穴が生まれたとき |

③ を忘れやすい。ドキュメントとヘルプは人間が読んで気づくが、**LLMナレッジが
古いことには誰も気づかない**（エージェントが黙って間違えるだけ）。

判断に迷ったときの目安:

- **JSON 出力のフィールドを1つでも変えた** → `20-schemas.md` は必須。エージェントは
  ここを信じてパースするため、ズレると黙って誤読する
- 指標を追加した → `10-metrics.md` のしきい値表と、BigQuery/API どちらで使えるかの明記
- 制限値を変えた（origins 上限、periods、レート制限）→ `30-gotchas.md` と
  `context7.json`
- 新しいサブコマンドを足した → ②を書いてから `go generate ./...`
- BigQuery 課金に関わる挙動を変えた → `crux-usage` の SKILL.md。課金は
  ユーザーの財布に直結するので、事前告知の手順を保つこと

## リリース

`PluginVersion`（`cmd/crux/main.go`）と `plugin.json` の `version` と git タグの
3つを揃える。テストとリリースワークフローが不一致を検出する。手順は
`plugins/crux-cli/PUBLISH.md`。

## 確認

```bash
go generate ./...     # 生成物を作り直す
git diff --exit-code  # 差分が出たらコミット漏れ
go test ./...         # SKILL.md 検証とバージョン整合を含む
go run ./cmd/crux llm | head
```

## 参照

- 標準: <https://github.com/ideamans/go-llm-cli-kit/blob/main/LLM.md>
- 生成物と原本の対応: `.claude/rules/ai-artifacts-policy.md`
- 再生成: `/regen-ai`
