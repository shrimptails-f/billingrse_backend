#!/bin/bash
input=$(cat)

cwd=$(echo "$input" | jq -r '.cwd // empty')
branch=$(git --git-dir="$cwd/.git" --work-tree="$cwd" branch --no-optional-locks 2>/dev/null | sed -e '/^[^*]/d' -e 's/* \(.*\)/ (\1)/')

# Context window usage
used_pct=$(echo "$input" | jq -r '.context_window.used_percentage // empty')
ctx_part=""
if [ -n "$used_pct" ]; then
  ctx_part=$(printf " \033[33mctx:%.0f%%\033[00m" "$used_pct")
fi

# Token counts from last API call
in_tok=$(echo "$input" | jq -r '.context_window.current_usage.input_tokens // empty')
out_tok=$(echo "$input" | jq -r '.context_window.current_usage.output_tokens // empty')
tok_part=""
if [ -n "$in_tok" ] && [ -n "$out_tok" ]; then
  tok_part=$(printf " \033[36min:%s out:%s\033[00m" "$in_tok" "$out_tok")
fi

# Claude.ai subscription rate limits
five_pct=$(echo "$input" | jq -r '.rate_limits.five_hour.used_percentage // empty')
week_pct=$(echo "$input" | jq -r '.rate_limits.seven_day.used_percentage // empty')
rate_part=""
if [ -n "$five_pct" ] || [ -n "$week_pct" ]; then
  rate_str=""
  [ -n "$five_pct" ] && rate_str=$(printf "5h:%.0f%%" "$five_pct")
  [ -n "$week_pct" ] && rate_str="$rate_str $(printf "7d:%.0f%%" "$week_pct")"
  rate_part=$(printf " \033[35m%s\033[00m" "$(echo "$rate_str" | xargs)")
fi

printf "%s@%s:%s\033[34m%s\033[00m%s%s%s" "$(whoami)" "$(hostname -s)" "$cwd" "$branch" "$ctx_part" "$tok_part" "$rate_part"
