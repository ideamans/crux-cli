# crux-cli 仕様書

## 概要

`crux` は Chrome UX Report (CrUX) の BigQuery データセットをコマンドラインからクエリするための CLI ツール。  
sitespeed-chronicle が Firebase Functions 内で行っている BigQuery アクセスをローカルで再現し、指標の参照・比較を手軽に行えるようにする。

### 設計参考

特定のツールを直接模倣したわけではないが、以下を参考にしている：

- **[gh](https://github.com/cli/gh)** (GitHub CLI) — cobra ベースのサブコマンド構成・出力フォーマット切り替えのシンプルさ
- **[gcloud](https://cloud.google.com/sdk/gcloud)** — BigQuery 系ツールにおける認証フロー（ADC / サービスアカウント）の慣習
- **[kubectl](https://kubernetes.io/docs/reference/kubectl/)** — `-o` / `--output` オプションによる出力形式指定の慣習

---

## 基本設計

### ツール名

```
crux
```

### サブコマンド構造

```
crux <subcommand> [options]
```

| サブコマンド | 説明 |
|---|---|
| `device`   | `chrome-ux-report.materialized.device_summary` をクエリ |
| `auth`     | BigQuery 認証の管理 |
| `cache`    | キャッシュの管理 |

---

## サブコマンド詳細

### `crux device`

`chrome-ux-report.materialized.device_summary` テーブルからオリジン別・デバイス別の Core Web Vitals 指標を取得する。

#### 使い方

```
crux device --origin <origin> [--origin <origin2> ...] [options]
```

オリジンは `--origin`（`-o`）を繰り返すか、カンマ区切りで複数指定できる。最大 **50オリジン**。

#### オプション

| オプション | 短縮 | デフォルト | 説明 |
|---|---|---|---|
| `--origin` | `-o` | (必須) | クエリ対象のオリジン。繰り返し指定またはカンマ区切りで複数可（最大50） |
| `--device` | `-d` | `all` | デバイス種別: `phone` / `desktop` / `tablet` / `all` |
| `--months` | `-m` | `12` | 直近何ヶ月分を取得するか |
| `--from` |  | | 開始月 (YYYYMM形式)。`--months` より優先 |
| `--to` |  | 最新月 | 終了月 (YYYYMM形式) |
| `--metrics` |  | (全指標) | 表示する指標をカンマ区切りで指定: `lcp,cls,inp,fcp,ttfb,fid,rtt` |
| `--format` | `-f` | `table` | 出力形式: `table` / `json` / `csv` |
| `--project` |  | 設定ファイル参照 | BigQuery プロジェクト ID |
| `--no-cache` |  | false | キャッシュを使用せず BigQuery を直接クエリ |

#### 複数オリジン指定の上限

BigQuery の `IN UNNEST(@array)` パラメータは配列サイズに厳密な制限はないが、以下の理由から **50オリジン** を上限とする：

- 1回のクエリで返る行数が `50オリジン × 月数 × デバイス数` に膨れる（例: 50 × 12 × 3 = 1,800行）
- キャッシュの粒度（オリジン × 月 × デバイス）を維持するため、大量一括取得は設計上の例外とする
- 実用上、競合比較は数サイト〜十数サイト規模が想定される

50を超える場合はエラーメッセージを出力して終了する。

#### 出力例（table形式、単一オリジン）

```
Origin: https://example.com  Device: phone  Period: 2024-05 ~ 2025-04

Month    LCP p75   LCP Good%  CLS p75  CLS Good%  INP p75   INP Good%
2025-04  2300ms    62%        0.08     89%        185ms     78%
2025-03  2450ms    59%        0.09     87%        192ms     76%
...
```

#### 出力例（table形式、複数オリジン）

複数オリジンの場合、オリジンを行方向に並べ、月を列方向に展開するか、オリジンごとにセクションを分ける。デフォルトはセクション分割。

```
=== https://example.com (phone) ===
Month    LCP p75   LCP Good%  ...
2025-04  2300ms    62%        ...

=== https://competitor.com (phone) ===
Month    LCP p75   LCP Good%  ...
2025-04  2800ms    48%        ...
```

#### クエリ対象テーブル

```sql
SELECT
  origin, yyyymm, device,
  fast_lcp, avg_lcp, slow_lcp, p75_lcp,
  fast_cls, avg_cls, slow_cls, p75_cls,
  fast_inp, avg_inp, slow_inp, p75_inp,
  fast_fcp, avg_fcp, slow_fcp, p75_fcp,
  fast_ttfb, avg_ttfb, slow_ttfb, p75_ttfb,
  fast_fid, avg_fid, slow_fid, p75_fid
FROM `chrome-ux-report.materialized.device_summary`
WHERE
  origin IN UNNEST(@origins)
  AND CAST(yyyymm AS INT64) >= CAST(@monthFrom AS INT64)
  AND CAST(yyyymm AS INT64) <= CAST(@monthTo AS INT64)
  AND (@device = 'all' OR device = @device)
ORDER BY origin, yyyymm DESC, device
```

#### 指標と閾値

| 指標 | カラムプレフィックス | Good | Poor | 備考 |
|---|---|---|---|---|
| LCP | `*_lcp` | 2500ms | 4000ms | Core Web Vital |
| CLS | `*_cls` | 0.1 | 0.25 | Core Web Vital |
| INP | `*_inp` | 200ms | 500ms | Core Web Vital |
| FCP | `*_fcp` | 1800ms | 3000ms | |
| TTFB | `*_ttfb` | 600ms | 1200ms | |
| FID | `*_fid` | 100ms | 300ms | レガシー |
| RTT | `*_rtt` | 75ms | 275ms | |

---

### `crux auth`

BigQuery 認証情報の管理。

```
crux auth login          # Application Default Credentials でログイン (gcloud ラップ)
crux auth logout         # 認証情報を削除
crux auth status         # 現在の認証状態を確認
crux auth set-project <project-id>  # デフォルト BigQuery プロジェクトを設定
```

---

### `crux cache`

ローカルキャッシュの管理。

```
crux cache list          # キャッシュ済みエントリ一覧
crux cache clear         # 全キャッシュを削除
crux cache clear --origin <origin>   # 特定オリジンのキャッシュを削除
crux cache dir           # キャッシュディレクトリのパスを表示
```

---

## 認証

### 優先順位

1. `--credentials` オプションで指定したサービスアカウントキーファイル (JSON)
2. `GOOGLE_APPLICATION_CREDENTIALS` 環境変数
3. Application Default Credentials (`gcloud auth application-default login` で設定)

### プロジェクト ID の解決順序

1. `--project` オプション
2. `CRUX_PROJECT` 環境変数
3. `~/.crux-cli/config.yaml` の `project_id`
4. Application Default Credentials に紐づくプロジェクト

> **Note:** `chrome-ux-report` データセットは Google の公開データセットのため、クエリコストは自分のプロジェクトに課金される。プロジェクト ID は必須。

---

## キャッシュ機構

### 設計思想

CrUX の `device_summary` は **毎月中旬（概ね12〜15日頃）に前月分のデータが追加**される月次更新。  
BigQuery のクエリコストは高額になりやすいため、取得済みデータをローカルにキャッシュして再利用する。

キャッシュの粒度は **オリジン × 開始月 × 終了月** とし、シンプルなファイル単位で管理する。  
複数オリジンをまとめてクエリする際は、キャッシュヒットしたオリジンを除外し、ミスしたオリジンのみ BigQuery に投げて効率化する。

### キャッシュディレクトリ

```
~/.crux-cli/cache/
```

`CRUX_CACHE_DIR` 環境変数または `~/.crux-cli/config.yaml` の `cache_dir` で変更可能。

### キャッシュファイル構造

```
~/.crux-cli/cache/
  latest_month.json                        # CrUX の最新月情報（24時間 TTL）
  device/
    <origin_hash>/
      <monthFrom>_<monthTo>.json.gz        # オリジン×期間 単位の圧縮キャッシュ
```

- `origin_hash`: オリジン URL の SHA256 ハッシュ（先頭16文字）
- ファイル名例: `202001_202504.json.gz`
- 内容: そのオリジン × 期間の全行（全デバイス・全月）を JSON 配列で保持し、gzip 圧縮

#### `latest_month.json` の構造

```json
{
  "yyyymm": "202504",
  "checked_at": "2025-05-20T09:12:00Z"
}
```

- `yyyymm`: 現在 CrUX に存在する最新月
- `checked_at`: BigQuery で確認した日時（RFC3339）

### キャッシュの有効性判定

キャッシュファイルのパス自体がキーになるため、別途有効期限フラグは不要：

- **キャッシュヒット条件**: `device/<origin_hash>/<monthFrom>_<monthTo>.json.gz` が存在する
- **キャッシュミス（自動無効化）**: 最新月が変わると `monthTo` が変化するためファイル名が一致せず、自動的にミス扱いになる

例：最新月が `202504` → `202605` に変わった場合、`*_202504.json.gz` は参照されなくなる（旧ファイルは残るが使われない）。  
古いキャッシュファイルの削除は `crux cache clear` で手動実施、または将来のクリーンアップ機能に委ねる。

### 最新月情報の管理

`latest_month.json` の `checked_at` から **24時間以上**経過している場合のみ BigQuery に問い合わせる：

1. `SELECT MAX(yyyymm) FROM \`chrome-ux-report.materialized.device_summary\`` を実行
2. 結果で `latest_month.json` を更新（月が変わっていなくても `checked_at` は更新）
3. 以降のキャッシュキー（`monthTo`）には常にこの最新月を使用

### キャッシュロジックフロー

```
crux device -o A -o B -o C --months 60 を実行
      ↓
1. latest_month.json を確認
   ├─ 24時間以上前 or 存在しない
   │      → BigQuery で MAX(yyyymm) を取得 → latest_month.json 更新
   └─ 24時間以内 → そのまま使用
      ↓
   monthTo = latest_month.yyyymm
   monthFrom = monthTo から 60ヶ月前
      ↓
2. 各オリジンのキャッシュを確認
   ├─ A: device/<hash_A>/202101_202504.json.gz → ヒット → ファイルから読み込み
   ├─ B: device/<hash_B>/202101_202504.json.gz → ミス
   └─ C: device/<hash_C>/202101_202504.json.gz → ミス
      ↓
3. ミスしたオリジン [B, C] のみ BigQuery にクエリ
   → 結果をそれぞれ圧縮してキャッシュファイルに保存
      ↓
4. A（キャッシュ）+ B + C のデータを結合して出力
```

---

## 設定ファイル

### パス

```
~/.crux-cli/config.yaml
```

### フォーマット

```yaml
# BigQuery プロジェクト ID（必須）
project_id: "your-gcp-project-id"

# キャッシュディレクトリ（デフォルト: ~/.crux-cli/cache）
cache_dir: ""

# デフォルト出力フォーマット: table / json / csv
default_format: "table"

# デフォルト取得月数
default_months: 12
```

---

## プロジェクト構造（Go）

```
crux-cli/
├── cmd/
│   └── crux/
│       └── main.go           # エントリポイント
├── internal/
│   ├── bigquery/
│   │   └── client.go         # BigQuery クライアント・クエリ実行
│   ├── cache/
│   │   └── cache.go          # キャッシュ読み書き・有効期限管理
│   ├── config/
│   │   └── config.go         # 設定ファイル読み書き
│   ├── formatter/
│   │   └── formatter.go      # table / json / csv 出力
│   └── crux/
│       └── device.go         # device サブコマンドのロジック
├── go.mod
├── go.sum
└── spec.md
```

### 主要依存ライブラリ

| ライブラリ | 用途 |
|---|---|
| `cloud.google.com/go/bigquery` | BigQuery クライアント |
| `github.com/spf13/cobra` | CLI フレームワーク |
| `github.com/spf13/viper` | 設定ファイル管理 |
| `golang.org/x/oauth2` | 認証（必要に応じて） |
| `github.com/olekukonko/tablewriter` | テーブル出力 |

---

## 実装フェーズ

### Phase 1（最小実装）

- [ ] Go モジュール初期化
- [ ] `crux device` コマンドの基本実装
  - BigQuery クエリ実行（`IN UNNEST(@origins)` で複数オリジン対応）
  - `table` 形式出力（単一オリジン：通常テーブル、複数オリジン：セクション分割）
  - `--origin`（繰り返し・カンマ区切り）、`--device`、`--months` オプション
- [ ] Application Default Credentials 認証
- [ ] キャッシュ基本実装
  - `latest_month.json`（24時間 TTL で最新月を保持）
  - 月・デバイス別データキャッシュ（永続）

### Phase 2

- [ ] `json` / `csv` 出力形式
- [ ] `crux auth` サブコマンド
- [ ] `crux cache` サブコマンド
- [ ] 設定ファイル（`~/.crux-cli/config.yaml`）
- [ ] `--from` / `--to` オプション
- [ ] `--metrics` フィルタ

### Phase 3

- [ ] 複数オリジンの比較出力
- [ ] グラフ出力（ASCII チャート）
- [ ] sitespeed-chronicle との連携オプション

---

## 使用例

```bash
# 直近12ヶ月のデータをテーブル表示（単一オリジン）
crux device --origin https://example.com

# 複数オリジン（--origin 繰り返し）
crux device -o https://example.com -o https://competitor.com

# 複数オリジン（カンマ区切り）
crux device -o https://example.com,https://competitor.com

# スマートフォンの LCP / CLS のみ表示（JSON）
crux device -o https://example.com -d phone --metrics lcp,cls -f json

# 複数オリジンを競合比較（直近3ヶ月、CSV出力）
crux device -o https://example.com -o https://rival.com --months 3 -f csv

# プロジェクト指定
crux device -o https://example.com --project my-gcp-project

# 認証設定
crux auth set-project my-gcp-project
crux auth status

# キャッシュ確認・削除
crux cache list
crux cache clear --origin https://example.com
```
