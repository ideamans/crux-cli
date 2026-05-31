# crux-cli

[Chrome UX Report (CrUX)](https://developer.chrome.com/docs/crux) の BigQuery データセットをコマンドラインからクエリするツールです。オリジン別の Core Web Vitals などのパフォーマンス指標を取得し、ローカルキャッシュによって BigQuery のクエリコストを最小化します。

## 機能

- `chrome-ux-report.materialized.device_summary` をコマンドラインから直接クエリ
- デバイス種別でフィルタ（phone / desktop / tablet / all）
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

crux-cli は **Application Default Credentials (ADC)** を使用します。最初に一度だけ実行してください：

```bash
gcloud auth application-default login
```

次に BigQuery プロジェクト ID を設定します：

```bash
crux auth set-project YOUR_GCP_PROJECT_ID
```

> **注意:** `chrome-ux-report` は Google の公開データセットですが、クエリコストはご自身のプロジェクトに課金されます。

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

### `crux auth`

```bash
crux auth status                      # 現在のプロジェクトと認証情報を確認
crux auth set-project <project-id>    # デフォルトの BigQuery プロジェクト ID を設定
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
| `cache_dir` | `~/.crux-cli/cache` | キャッシュディレクトリのパス |
| `default_format` | `table` | デフォルトの出力形式 |
| `default_months` | `12` | デフォルトの取得月数 |

環境変数でも設定可能です：

| 変数名 | 説明 |
|---|---|
| `CRUX_PROJECT` | BigQuery プロジェクト ID |
| `CRUX_CACHE_DIR` | キャッシュディレクトリのパス |

## ライセンス

MIT
