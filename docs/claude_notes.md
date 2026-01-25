# Claude Code CLI 使い方

## 基本の使い方
- 実行バイナリ: `/home/dev/.local/bin/claude`
- 対話継続: `claude`（同じシェルで質問を順に入力）
- ワンショット: `claude --print "質問"` / 直前セッション継続は `--continue --print`
- `approval_policy=never` でも既存の `~/.claude` は書き込みできるので追加設定は不要。独自 `CLAUDE_CONFIG_DIR` を指定すると認証情報が消えて失敗する点に注意。
- 別ディレクトリを使う必要がある場合のみ `CLAUDE_CONFIG_DIR`/`HOME` を切り替え、`mkdir -p <dir> && touch <dir>/.claude.json` などで最低限のファイルを用意してから起動する。
- 作業開始時は必ず 以下のファイルを `--system-prompt` で読み込ませる。
  - `/home/dev/backend/tmp/prompts/worker.md`
  - `/home/dev/backend/docs/architecture.md`
  - `/home/dev/backend/docs/coding_rules.md`
  - CLI は `--system-prompt` を 1 つしか受け取れないため、以下のように連結して渡す。
    ```bash
    cat Orchestrator/prompts/worker.md \
        Orchestrator/prompts/coding_conventions.md \
        Orchestrator/prompts/architecture_overview.md \
        > tmp/system_prompt.txt
    claude --dangerously-skip-permissions \
      --system-prompt "$(cat tmp/system_prompt.txt)" \
      --print "$(cat tmp/worker_instruction.txt)"
    ```

## 推奨フロー
1. **長文指示はファイル化**  
   ```bash
   cat <<'EOF' > tmp/worker_instruction.txt
   ...TLからの指示...
   EOF
   ```
2. **CLI 実行**  
   ```bash
   claude --dangerously-skip-permissions \
     --system-prompt "$(cat Orchestrator/prompts/worker.md)" \
     --print "$(cat tmp/worker_instruction.txt)"
   ```
   - 途中で止まったら `--continue` を付けて再実行する。
3. **結果確認**  
   - ログ: `~/.claude/debug/latest`
   - Todo/進捗: `~/.claude/todos/*.json`

## よくあるハマりどころ
- 直接コマンドラインに ``POST /...`` などを含む長文を埋め込むとシェルが誤解釈して失敗する → かならずファイル経由で渡す。
- 権限確認ポップアップを待つ環境では `--dangerously-skip-permissions` を付けないとタイムアウトする。
- 指示が通らない場合はセッションを作り直して再送すればよい（`claude` → `/exit` → 再実行）。
