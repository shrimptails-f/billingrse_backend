tables:
  users:
    role: "ユーザー情報（認証・メール検証状態を含む）"
    relation: ["emails (1:N)", "email_projects (1:N)", "entry_timings (1:N)", "email_keyword_groups (1:N)", "email_position_groups (1:N)", "email_work_type_groups (1:N)", "email_candidates (1:N)"]
    note: "メール解析データはすべてこのユーザーに紐づく。ON DELETE CASCADE で子テーブルも削除される。"
