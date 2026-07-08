#!/usr/bin/env bash
# gp-cli Agent Eval Runner
#
# tests/agent-prompts/NN-*.md 의 "## The Prompt" 블록을 추출해 `claude -p`
# 헤드리스 세션으로 실행하고, 결과(raw 응답 + 자가평가 JSON)를 results/ 에 저장한다.
#
# Usage:
#   ./eval_runner.sh            # 모든 프롬프트 실행
#   ./eval_runner.sh 1 3        # 1번, 3번 프롬프트만 실행
#   ./eval_runner.sh --dry-run  # 실제 호출 없이 합성된 프롬프트만 출력
#
# 결과:
#   results/promptNN.txt   - claude -p 의 raw stdout
#   results/promptNN.json  - 응답 끝의 자가평가 JSON 블록 (추출 성공 시)
#   results/eval_summary.md - 사람이 채울 점수표 템플릿 (없으면 생성)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"
mkdir -p "$RESULTS_DIR"

DRY_RUN=0
PROMPT_NUMS=()
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=1 ;;
    *) PROMPT_NUMS+=("$arg") ;;
  esac
done

# 프롬프트 파일 끝에 덧붙일 공통 지시문.
# eval_runner.py(uspto-cli)의 "FINAL STEP" 패턴을 본떠, agent가 스스로
# 사용 경험과 개선 제안을 구조화된 JSON으로 보고하도록 강제한다.
read -r -d '' OUTPUT_INSTRUCTION <<'EOF' || true

---
FINAL STEP — REQUIRED: 위 작업을 모두 마친 뒤, 응답의 마지막 줄에 아래 스키마를
따르는 한 줄짜리 JSON을 출력하라 (다른 텍스트로 감싸지 말고 그대로 한 줄로):

{"success": true|false, "commands": ["gp-cli lookup ...", ...], "ax": "이 CLI를 쓰면서 무엇이 잘 됐고 무엇이 헷갈렸는지", "suggestions": "CLI나 문서를 개선할 제안"}
EOF

CONSTRAINT="중요: gp-cli 바이너리(또는 'go run ./cmd/gp-cli/')만 사용하라. 다른 특허 검색 도구나 외부 API를 직접 호출하지 마라."

extract_prompt_block() {
  # "## The Prompt" 다음에 등장하는 "> " 로 시작하는 인용 블록을 추출해
  # 인용 기호를 제거하고 합쳐서 출력한다.
  local file="$1"
  awk '
    /^## The Prompt/ { in_section=1; next }
    in_section && /^>/ {
      line=$0
      sub(/^> ?/, "", line)
      print line
      found=1
      next
    }
    in_section && found && /^$/ { exit }
  ' "$file"
}

run_prompt() {
  local file="$1"
  local base
  base="$(basename "$file" .md)"
  local num
  num="$(echo "$base" | grep -oE '^[0-9]+')"
  local out_txt="$RESULTS_DIR/prompt${num}.txt"
  local out_json="$RESULTS_DIR/prompt${num}.json"

  local scenario
  scenario="$(extract_prompt_block "$file")"
  if [[ -z "$scenario" ]]; then
    echo "[prompt $num] '## The Prompt' 인용 블록을 찾지 못함 — 건너뜀: $file" >&2
    return
  fi

  local full_prompt
  full_prompt="$(printf '%s\n\n%s\n%s\n' "$CONSTRAINT" "$scenario" "$OUTPUT_INSTRUCTION")"

  if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "===== prompt $num ($base) ====="
    echo "$full_prompt"
    echo
    return
  fi

  echo "[prompt $num] 실행 중: $base"
  if ! claude -p "$full_prompt" --output-format text --permission-mode bypassPermissions > "$out_txt" 2> "$RESULTS_DIR/prompt${num}.err"; then
    echo "[prompt $num] claude -p 실행 실패 — $RESULTS_DIR/prompt${num}.err 확인" >&2
    return
  fi

  # 응답 마지막 줄(또는 마지막 JSON 객체)을 자가평가 결과로 추출.
  if grep -oE '\{.*"success".*\}' "$out_txt" | tail -n 1 > "$out_json" 2>/dev/null && [[ -s "$out_json" ]]; then
    echo "[prompt $num] 완료 — raw: $out_txt / 자가평가: $out_json"
  else
    rm -f "$out_json"
    echo "[prompt $num] 완료 — raw: $out_txt (자가평가 JSON 추출 실패, 수동 확인 필요)"
  fi
}

PROMPT_FILES=()
while IFS= read -r f; do
  PROMPT_FILES+=("$f")
done < <(find "$SCRIPT_DIR" -maxdepth 1 -name '[0-9][0-9]-*.md' | sort)

if [[ ${#PROMPT_NUMS[@]} -gt 0 ]]; then
  FILTERED=()
  for f in "${PROMPT_FILES[@]}"; do
    n="$(basename "$f" | grep -oE '^[0-9]+' | sed 's/^0*//')"
    for want in "${PROMPT_NUMS[@]}"; do
      [[ "$n" == "$want" ]] && FILTERED+=("$f")
    done
  done
  PROMPT_FILES=("${FILTERED[@]}")
fi

for f in "${PROMPT_FILES[@]}"; do
  run_prompt "$f"
done

if [[ "$DRY_RUN" -eq 0 ]]; then
  SUMMARY="$RESULTS_DIR/eval_summary.md"
  if [[ ! -f "$SUMMARY" ]]; then
    {
      echo "# gp-cli Agent Eval Summary"
      echo
      echo "| Prompt | success | 주요 발견 (ax/suggestions 요약) | 등급(A-F) |"
      echo "|--------|---------|-------------------------------|-----------|"
      for f in "${PROMPT_FILES[@]}"; do
        n="$(basename "$f" | grep -oE '^[0-9]+' | sed 's/^0*//')"
        echo "| $n | | | |"
      done
    } > "$SUMMARY"
    echo "템플릿 생성됨: $SUMMARY (results/promptNN.json 의 ax/suggestions를 보고 채워 넣을 것)"
  fi
fi
