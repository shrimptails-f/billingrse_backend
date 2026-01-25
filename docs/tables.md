tables:
  users:
    role: "ユーザー情報（認証・メール検証状態を含む）"
    relation: ["emails (1:N)", "email_projects (1:N)", "entry_timings (1:N)", "email_keyword_groups (1:N)", "email_position_groups (1:N)", "email_work_type_groups (1:N)", "email_candidates (1:N)"]
    note: "メール解析データはすべてこのユーザーに紐づく。ON DELETE CASCADE で子テーブルも削除される。"

  emails:
    role: "全メール共通の基本情報（件名・送信元・本文など）"
    relation: ["users (N:1)"]
    note: "user_id + gmail_id でユニーク制約。同じ GmailID でもユーザーごとに保存可能。"

  email_projects:
    role: "案件メール専用の詳細情報（単価・勤務地・技術要素など）"
    relation:
      - emails (1:1)
      - users (N:1)
      - entry_timings (1:N)
    note: "一覧画面用に技術・業務・ポジションなどをカンマ区切り文字列でも保持（二重管理）。user_id を含めて emails とリレーション。"

  entry_timings:
    role: "案件の入場時期（複数）を正規化管理"
    relation: ["email_projects (N:1)", "users (N:1)"]
    note: "user_id + email_id + start_date で複合主キー。ユーザー別に入場時期を管理。"

  keyword_groups:
    role: "正規化された技術キーワードのマスタ（PHP、Reactなど）"
    relation: [key_words (1:N), email_keyword_groups (1:N keyword_group_word_links)]
    note: "ユーザー共通のマスタテーブル（user_id を持たない）"

  keyword_group_word_links:
    role: "keyword_groups と key_words を多対多で結びつける中間テーブル"
    relation: [keyword_groups (N:1), key_words (N:1)]

  key_words:
    role: "キーワードの表記ゆれを keyword_groups に紐付ける"
    relation: ["keyword_group_word_links (N:1)"]

  email_keyword_groups:
    role: "emails と keyword_groups の多対多中間テーブル（type区分あり）"
    relation: ["emails (N:1)", "keyword_groups (N:1)", "users (N:1)"]
    note: "user_id + email_id + keyword_group_id で複合主キー。ユーザー別にキーワードを管理。"

  position_groups:
    role: "正規化されたポジション名のマスタ（例: PM, PL）"
    relation: ["position_words (1:N)", "email_position_groups (1:N)"]
    note: "ユーザー共通のマスタテーブル（user_id を持たない）"

  position_words:
    role: "ポジションの表記ゆれを position_groups に紐付ける"
    relation: ["position_groups (N:1)"]

  email_position_groups:
    role: "emails と position_groups の多対多中間テーブル"
    relation: ["emails (N:1)", "position_groups (N:1)", "users (N:1)"]
    note: "user_id + email_id + position_group_id で複合主キー。"

  work_type_groups:
    role: "正規化された業務種別マスタ（例: バックエンド開発）"
    relation: ["work_type_words (1:N)", "email_work_type_groups (1:N)"]
    note: "ユーザー共通のマスタテーブル（user_id を持たない）"

  work_type_words:
    role: "業務表記ゆれを work_type_groups に紐付ける"
    relation: ["work_type_groups (N:1)"]

  email_work_type_groups:
    role: "emails と work_type_groups の多対多中間テーブル"
    relation: ["emails (N:1)", "work_type_groups (N:1)", "users (N:1)"]
    note: "user_id + email_id + work_type_group_id で複合主キー。"

  email_candidates:
    role: "人材提案メール専用の詳細情報（将来拡張用）"
    relation: ["emails (1:1)", "users (N:1)"]
    note: "user_id を含めて emails とリレーション。"
