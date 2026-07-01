# crux-cli

[Chrome UX Report (CrUX)](https://developer.chrome.com/docs/crux) のデータをコマンドラインからクエリするツールです。2つのデータソースに対応します：

- **BigQuery**（`crux device`）— `chrome-ux-report.materialized.device_summary` のオリジン別・月次集計。クエリコストはご自身の Google Cloud プロジェクトに課金され、ローカルキャッシュが効きます。
- **CrUX API**（`crux history` / `crux record`）— 無料の公開 [CrUX API](https://developer.chrome.com/docs/crux/api?hl=ja)。**オリジンまたは特定の URL** について、直近28日間のローリング分布を週次の時系列または最新スナップショットとして取得します。APIキーのみで利用できます。

## 機能

- `chrome-ux-report.materialized.device_summary` をコマンドラインから直接クエリ（BigQuery）
- CrUX API でオリジン**または特定の URL** を、週次の履歴または最新レコードとして取得
- デバイス種別 / フォームファクタでフィルタ（phone / desktop / tablet / all）
- 複数オリジンの一括クエリ（競合比較に便利）
- オリジン × 期間 単位のローカルキャッシュ — キャッシュミスしたオリジンのみ BigQuery に問い合わせ
- CrUX の最新月を毎日1回確認し、新しい月が公開されたらキャッシュを自動的に無効化
- table / JSON / CSV 形式で出力

## 必要なもの

- Go 1.21 以上
- Google Cloud プロジェクト（BigQuery のクエリコストはお客様のプロジェクトに課金されます）
- [Google Cloud SDK](https://cloud.google.com/sdk)（Application Default Credentials の設定に使用）

## インストール

```bash
git clone https://github.com/ideamans/crux-cli.git
cd crux-cli
go build -o crux ./cmd/crux
# 必要に応じて PATH の通ったディレクトリに移動
mv crux /usr/local/bin/
```

## 認証

### BigQuery（`crux device`）

crux-cli は **Application Default Credentials (ADC)** を使用します。最初に一度だけ実行してください：

```bash
gcloud auth application-default login
```

次に BigQuery プロジェクト ID を設定します：

```bash
crux auth set-project YOUR_GCP_PROJECT_ID
```

> **注意:** `chrome-ux-report` は Google の公開データセットですが、クエリコストはご自身のプロジェクトに課金されます。

### CrUX API（`crux history` / `crux record`）

CrUX API の利用には API キーが必要です。[Google Cloud Console](https://console.cloud.google.com/apis/credentials) で *Chrome UX Report API* を有効化して API キーを発行し、次のいずれかの方法で指定します：

```bash
# 設定ファイル（~/.crux-cli/config.json）に保存
crux auth set-api-key YOUR_CRUX_API_KEY

# …またはシェルの環境変数で指定
export CRUX_API_KEY=YOUR_CRUX_API_KEY

# …またはコマンドで直接指定
crux history -o https://example.com --api-key YOUR_CRUX_API_KEY
```

優先順位は `--api-key` > `CRUX_API_KEY` 環境変数 > 設定ファイル です。CrUX API は無料です（キーあたり 150 クエリ/分のレート制限）。

## クイックスタート

```bash
# 直近12ヶ月（デフォルト指標: LCP / CLS / INP）
crux device -o https://example.com

# スマートフォンのみ
crux device -o https://example.com -d phone

# 複数オリジンを比較
crux device -o https://example.com -o https://competitor.com

# FID 以外の全指標
crux device -o https://example.com --full-metrics

# 特定の指標を指定
crux device -o https://example.com --metrics fcp,ttfb,ol

# 直近3ヶ月を JSON 出力
crux device -o https://example.com --months 3 -f json

# 期間を直接指定
crux device -o https://example.com --from 202401 --to 202412

# --- CrUX API（APIキーが必要） ---

# オリジンの週次履歴（最新スナップショットは crux record）
crux history -o https://web.dev

# 特定ページの最新レコードをデスクトップで
crux record -u https://web.dev/learn -d desktop
```

## コマンド

### `crux device`

device_summary を照会し、Web Vitals 指標を表示します。

```
crux device --origin <origin> [フラグ]
```

| フラグ | 短縮 | デフォルト | 説明 |
|---|---|---|---|
| `--origin` | `-o` | 必須 | クエリ対象のオリジン。繰り返しまたはカンマ区切りで複数指定可（最大50件） |
| `--device` | `-d` | `all` | `phone` / `desktop` / `tablet` / `all` |
| `--months` | `-m` | `12` | 取得する月数 |
| `--from` | | | 開始月 `YYYYMM`（`--months` より優先） |
| `--to` | | | 終了月 `YYYYMM`（デフォルト: CrUX の最新月） |
| `--metrics` | | | 表示する指標をカンマ区切りで指定。指定可能: `lcp,cls,inp,fcp,ttfb,ol,rtt,fid` |
| `--full-metrics` | | | `fid` 以外の全指標を表示 |
| `--format` | `-f` | `table` | `table` / `json` / `csv` |
| `--project` | | | BigQuery プロジェクト ID（設定ファイルより優先） |
| `--no-cache` | | | キャッシュをスキップして常に BigQuery に問い合わせ |

**利用可能な指標**

| キー | 指標名 | Good 閾値 |
|---|---|---|
| `lcp` | Largest Contentful Paint | 2500ms 以下 |
| `cls` | Cumulative Layout Shift | 0.1 以下 |
| `inp` | Interaction to Next Paint | 200ms 以下 |
| `fcp` | First Contentful Paint | 1800ms 以下 |
| `ttfb` | Time to First Byte | 600ms 以下 |
| `ol` | Overall Load | 4000ms 以下 |
| `rtt` | Round Trip Time | 75ms 以下 |
| `fid` | First Input Delay（レガシー） | 100ms 以下 |

デフォルト表示は `lcp`, `cls`, `inp` の3指標。`--full-metrics` で `fid` 以外の全指標を表示。`fid` は `--metrics fid` と明示した場合のみ表示されます。

### `crux history`（CrUX API — 週次の時系列）

[CrUX History API](https://developer.chrome.com/docs/crux/history-api?hl=ja) を使い、オリジンまたは特定 URL の週次の時系列（最大40期間）を取得します。

```
crux history (--origin <origin> | --url <url>) [フラグ]
```

| フラグ | 短縮 | デフォルト | 説明 |
|---|---|---|---|
| `--origin` | `-o` | | クエリ対象のオリジン。繰り返しまたはカンマ区切りで複数指定可 |
| `--url` | `-u` | | 特定ページの URL。繰り返しまたはカンマ区切りで複数指定可 |
| `--device` | `-d` | `all` | フォームファクタ: `phone` / `desktop` / `tablet` / `all`。`all` は全デバイスの集計 |
| `--connection` | `-c` | `all` | 実効接続タイプ: `4g` / `3g` / `2g` / `slow-2g` / `offline` / `all`。`all` は全接続タイプの集計 |
| `--periods` | | `25` | 取得する週次期間の数（`1`〜`40`） |
| `--metrics` | | | 取得する指標をカンマ区切りで指定。指定可能: `lcp,cls,inp,fcp,ttfb,rtt` |
| `--format` | `-f` | `table` | `table` / `json` / `csv` |
| `--api-key` | | | CrUX API キー（`CRUX_API_KEY` および設定ファイルより優先） |

`--origin` または `--url` を少なくとも1つ指定してください。

```bash
# オリジンの直近25週（スマートフォン）、デフォルト指標
crux history -o https://web.dev -d phone

# 特定ページ、全指標、40期間
crux history -u https://web.dev/learn --metrics lcp,cls,inp,fcp,ttfb,rtt --periods 40

# スマートフォンかつ 4G 接続のユーザーのみ
crux history -o https://web.dev -d phone -c 4g

# オリジンとページを JSON で比較
crux history -o https://web.dev -u https://web.dev/learn -f json
```

### `crux record`（CrUX API — 最新スナップショット）

CrUX API で最新の単一レコード（直近28日間のスナップショット）を取得します。フラグは `--periods` を除いて `crux history` と同じです。

```bash
crux record -o https://web.dev
crux record -u https://web.dev/learn -d desktop -f json
```

> **注意:** `crux device`（BigQuery）と異なり、ここでの `--device all` は**全フォームファクタの集計**（単一レコード）を意味し、デバイス別の内訳ではありません。CrUX API は直近28日間のローリング分布を返し、週次の期間は終了日でラベル付けされます。

### `crux auth`

```bash
crux auth status                      # 現在のプロジェクト・認証情報・CrUX API キーの状態を確認
crux auth set-project <project-id>    # デフォルトの BigQuery プロジェクト ID を設定
crux auth set-api-key <api-key>       # CrUX API キーを設定
```

### `crux --llm`

LLM / AI エージェント向けの詳細リファレンスを表示して終了します。2つのデータソース、全コマンド・全フラグ、指標の閾値、そして `-f json` 出力の各項目の意味（`fast_*` 等の密度、`p75_*` の単位、CrUX API の `good/needs_improvement/poor/p75` 構造）まで網羅しています。エージェントに `crux` を自律的に使わせる際のコンテキストとして利用できます。

```bash
crux --llm            # 全体ガイド
crux history --llm    # 同じガイド（任意のサブコマンドで有効）
```

### `crux cache`

```bash
crux cache dir                # キャッシュディレクトリのパスを表示
crux cache list               # キャッシュ一覧と最新月情報を表示
crux cache clear              # 全デバイスキャッシュを削除
crux cache clear -o <origin>  # 特定オリジンのキャッシュを削除
```

## キャッシュの仕組み

キャッシュファイルはデフォルトで `~/.crux-cli/cache/` に保存されます。

```
~/.crux-cli/cache/
  latest_month.json                   # CrUX の最新月（24時間ごとに更新確認）
  device/
    <オリジンのハッシュ>/
      <YYYYMM>_<YYYYMM>.json.gz       # 1オリジン × 期間の全行をgzip圧縮したJSON
```

- キャッシュキーは `(オリジン, 開始月, 終了月)` の組み合わせです。
- CrUX は毎月中旬（12〜15日頃）に前月分のデータを追加します。最新月が変わると終了月がキャッシュキーに含まれているため、旧キャッシュは自然にミスとなり新しいデータが取得されます。
- 複数オリジンを指定した場合、キャッシュミスのオリジンのみ BigQuery にクエリします。
- `--no-cache` でキャッシュを完全にスキップできます。

## 設定ファイル

設定は `~/.crux-cli/config.json` に保存されます。

| キー | デフォルト | 説明 |
|---|---|---|
| `project_id` | | BigQuery プロジェクト ID |
| `api_key` | | CrUX API キー |
| `cache_dir` | `~/.crux-cli/cache` | キャッシュディレクトリのパス |
| `default_format` | `table` | デフォルトの出力形式 |
| `default_months` | `12` | デフォルトの取得月数 |

環境変数でも設定可能です：

| 変数名 | 説明 |
|---|---|
| `CRUX_PROJECT` | BigQuery プロジェクト ID |
| `CRUX_API_KEY` | CrUX API キー |
| `CRUX_CACHE_DIR` | キャッシュディレクトリのパス |

## ライセンス

MIT
