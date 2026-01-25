# 接続情報
- ホスト:localhost
- ポート:3306
- ユーザー:user
- パスワード:password
- データベース:development
# 接続設定例
A5:SQL Mk-2ではこのように接続設定をしていました。
![Image](https://github.com/user-attachments/assets/cd082944-b866-4bb5-8219-8609f2537c6c)
# SQL例
お好きなDBクライアントで以下のクエリを実行してください。
**注意**: user_id でフィルタすることで、特定ユーザーのデータのみを取得します。
```
SELECT DISTINCT
  e.user_id,
  e.gmail_id,
  DATE_FORMAT(e.received_date, '%m/%d ')as '受信日',
  ep.project_title,
  ep.work_location as '勤務地',
  ep.remote_type as 'リモート種別',
  ep.remote_frequency as 'リモート頻度',
--  ep.entry_timing,
--  ep.end_timing,
  ep.positions as 'ポジション名',
--  ep.work_types as '業務種別',
  ep.price_from as '単価FROM',
  ep.price_to as '単価TO',
  ep.frameworks as 'フレームワーク',
  ep.languages as  '言語',
  e.is_read as '既読',
  e.is_good as 'good',
  e.is_bad as 'bad'
FROM emails e
JOIN email_projects ep ON e.id = ep.email_id AND e.user_id = ep.user_id
-- 技術キーワード（MUST/WANT/LANGUAGE/FRAMEWORK）
LEFT JOIN email_keyword_groups ekg ON e.id = ekg.email_id AND e.user_id = ekg.user_id
LEFT JOIN keyword_groups kg ON ekg.keyword_group_id = kg.keyword_group_id
-- ポジション
LEFT JOIN email_position_groups epg ON e.id = epg.email_id AND e.user_id = epg.user_id
LEFT JOIN position_groups pg ON epg.position_group_id = pg.position_group_id
-- 業務
LEFT JOIN email_work_type_groups ewtg ON e.id = ewtg.email_id AND e.user_id = ewtg.user_id
LEFT JOIN work_type_groups wtg ON ewtg.work_type_group_id = wtg.work_type_group_id
-- 入場時期keyword_group_idkeyword_group_id
LEFT JOIN entry_timings et ON ep.email_id = et.email_id AND ep.user_id = et.user_id
WHERE
e.user_id = 1 // ★必須: ユーザーIDを指定してデータを分離
AND e.category = '案件' // メール区分を指定　案件 or 人材
AND e.received_date > '2025-05-31' // 受信日を指定
-- AND ep.price_from > 700000 // 単価を指定する場合。
-- AND ep.price_fo > 700000 // 単価を指定する場合。
-- AND ep.remote_type NOT IN ('不可')
-- AND e.gmail_id = 'GメールIDを記載'
-- AND kg.name = 'Go' // 言語を指定する場合
ORDER BY `受信日` DESC
;
```