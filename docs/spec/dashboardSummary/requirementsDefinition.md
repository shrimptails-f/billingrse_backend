# Dashboard Summary API 要件定義

## 本書の位置づけ

- 本書は `Dashboard Summary API` の To-Be 要件を整理する。
- 基本設計は [design.md](./design.md) を参照する。

## 背景

- 既存実装では `GET /api/v1/billings` と `GET /api/v1/billings/summary/*` により、請求一覧と請求サマリは取得できる。
- 一方でダッシュボード初期表示では、一覧や月次明細ではなく「今月の解析成功件数」「累計保存請求件数」「今月の補完件数」を軽く表示したい。
- 今回必要な 3 指標は `parsed_emails` と `billings` にまたがるため、既存 `billings/summary` にそのまま寄せると責務がぶれやすい。
- DDD 上でも、これら 3 指標を 1 つの業務集約としては定義していないため、新しい aggregate 追加ではなく画面専用の read API として扱う。

## 解決したい課題

- ダッシュボード初期表示のために、複数 API を組み合わせずに KPI を一括取得したい。
- 解析成功件数は workflow 監査集計ではなく、現在値に近い保存済み解析結果ベースで取得したい。
- 請求総件数と今月の補完件数は、既存 `billings` の保存結果と意味がずれない形で取得したい。

## 目標

- `GET /api/v1/dashboard/summary` で、認証済みユーザー自身の KPI を取得できるようにする。
- v1 では次の 3 指標を返せるようにする。
  - `current_month_analysis_success_count`
  - `total_saved_billing_count`
  - `current_month_fallback_billing_count`
- フロントのモックを追加変換なしでそのまま描画できる shape にする。

## 対象範囲

- ダッシュボード KPI 取得 API の責務定義
- 3 指標の意味と集計元の定義
- 月境界と所有範囲の扱い
- read API としての package 境界整理

## 非対象

- 解析成功件数の日次推移、月次推移 API
- 請求金額合計、通貨別合計、Vendor 内訳
- connection 単位、provider 単位の breakdown
- frontend ごとのタイムゾーン切り替え
- workflow 監査 API の置き換え

## 機能要件

- `FR-1`
  - `GET /api/v1/dashboard/summary` は認証済みユーザーのみ利用できる。
- `FR-2`
  - API は自分自身が所有するデータのみを集計対象にする。
- `FR-3`
  - レスポンスは少なくとも以下 3 項目を返せる。
    - `current_month_analysis_success_count`
    - `total_saved_billing_count`
    - `current_month_fallback_billing_count`
- `FR-4`
  - `current_month_analysis_success_count` は「当月に解析ステージで成功し、保存された件数」を意味する。
- `FR-5`
  - `total_saved_billing_count` は「これまでに請求として実際に保存された総件数」を意味する。
- `FR-6`
  - `current_month_fallback_billing_count` は「当月に請求として扱われる保存済みデータのうち、`billing_date` がなく、メール受信日 fallback で判定した件数」を意味する。
- `FR-7`
  - データが存在しない場合でも `200 OK` を返し、各項目は `0` を返せる。

## 非機能要件

- `NFR-1`
  - HTTP 契約は `docs/api_design.md` に従う。
- `NFR-2`
  - JSON フィールド名は `lower_snake_case` とする。
- `NFR-3`
  - ダッシュボード初期表示向けに軽量なレスポンスとし、不要な一覧や内訳は返さない。
- `NFR-4`
  - 集計は固定本数のクエリで完結し、N+1 を起こさない。
- `NFR-5`
  - 月境界の解釈は v1 では UTC に固定する。
- `NFR-6`
  - レスポンスや構造化ログにメール本文、メールアドレス、OAuth token、prompt、生の解析結果などの秘匿情報を含めない。
- `NFR-7`
  - dashboard KPI は read model として扱い、`Billing` aggregate や `manualmailworkflow` aggregate の意味を変更しない。
- `NFR-8`
  - 件数集計は domain に持ち込まず、read repository の SQL 集計で取得する。

## 制約・前提

- `current_month_analysis_success_count` は workflow 履歴 header の合算ではなく、保存済み解析結果を基準に扱う。
- `total_saved_billing_count` と `current_month_fallback_billing_count` は `billings` を基準に扱う。
- `current_month_fallback_billing_count` の判定条件は `billings.billing_date IS NULL` を基準にする。
- `current_month_fallback_billing_count` の月判定は `billings.billing_summary_date` を使い、UTC 当月範囲で集計する。
- `total_saved_billing_count` と `current_month_fallback_billing_count` の「実件数と一致」は、値の定義と集計元の意味を示すものであり、別途 reconcile 用の検証処理や同期処理を要求するものではない。

## 成功条件

- `GET /api/v1/dashboard/summary` で、認証済みユーザー自身の 3 KPI を取得できる。
- フロントのモックを追加変換なしでそのまま描画できる。
- `current_month_analysis_success_count` が保存済み解析結果ベースの件数として説明できる。
- `total_saved_billing_count` が保存済み請求件数の意味で解釈できる。
- `current_month_fallback_billing_count` が当月の `billing_date` fallback 件数の意味で解釈できる。
- データ未作成ユーザーでも response shape が崩れず、`0` / `0` / `0` を返せる。
