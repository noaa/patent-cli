# gp-cli Agent Stress Test Prompts

특허 담당자가 실제로 물어볼 법한 자연어 요청 6개. 각 프롬프트는 `gp-cli`의
주요 기능을 자연스럽게 끌어내도록 설계되었으며, agent에게 새 세션에서 그대로
전달해 사용한다. (참고: [smcronin/uspto-cli](https://github.com/smcronin/uspto-cli)의
`tests/agent-prompts/` 구조를 참고해 도입함 — 분석 내용은
`temp/uspto-cli-agent-ux-analysis-20260607.md` 참조)

## Coverage Matrix

| Prompt | Commands Exercised | Formats | Edge Cases |
|--------|-------------------|---------|------------|
| 1 | lookup (`--fields`, `--field`, `--output-dir`, `--format text`) | text | `--field` 단수와 `--output-dir` 조합 |
| 2 | download/lookup bulk (`--input-file`, `--no-header`, `--delay`, `--quiet`), TSV 누적 | tsv | 존재하지 않는 특허(exit 4), stderr/stdout 분리 |
| 3 | lookup (`--language en`, `--fields claims_structured,description_structured`, `--output-dir`) | json | 번역 페이지의 `_warnings`, 구조화 필드 스키마 |
| 4 | lookup (`--field backward_citations`/`forward_citations`), jq 파이프라인 | json | 중첩 객체 배열 스키마 추론 |
| 5 | images (`--output-dir`), 파일명 규칙 | binary | 파일명 패턴(`{번호}_fig01.png`) 검증 |
| 6 | family-group (`--format`, `--output-dir`, `--no-header`), `fields`/`configure`/`update --check` | tsv/json | 다중 특허 묶음 처리, 설정/업데이트 보조 명령 |

## How to Score

각 프롬프트마다 다음을 관찰해 기록한다:

- agent가 어떤 플래그를 써야 하는지 스스로 알아냈는가, 아니면 헤맸는가?
- `--field`(단수) vs `--fields`(복수), `--output-dir`, `--no-header`, `--delay` 같은
  비슷해 보이는 옵션을 올바른 상황에 선택했는가?
- 예상되는 실패(존재하지 않는 특허, 번역 페이지의 구조 정보 누락 등)를 우아하게 처리했는가?
- 결과를 raw JSON 덤프가 아니라 사람이 읽기 좋은 형태로 종합했는가?
- 불필요한 호출 없이 명령을 논리적으로 연결했는가?

## Real Patent Numbers Used

- **US12514139B2** — 최근 등록 특허, 메타데이터/청구항 풍부
- **US8725880B2** 패밀리 — `tests/integration/family_group_test.go`에서 쓰는 다국가 패밀리
  (US/EP/KR/JP/CN/GB/DE/AU/BR/MX/TW/WO 등)
- **KR102355140B1** — 한국어 원문 특허, `--language en` 번역 테스트용
- **US11125686B2** — 도면(images)이 풍부한 특허

## Running the Eval Loop

```bash
# 전체 6개 프롬프트 실행 (claude -p 헤드리스 모드로 호출)
./eval_runner.sh

# 특정 프롬프트만
./eval_runner.sh 1 3

# 결과는 results/promptNN.txt (raw) / results/promptNN.json (자가평가 JSON)에 저장됨
```

`eval_runner.sh`는 각 프롬프트 끝에 "마지막에 평가 결과를 JSON으로 출력하라"는
지시문을 덧붙여 agent 스스로 `success`/`commands`/`ax`(사용 경험)/`suggestions`를
보고하게 만든다. 사람은 `results/eval_summary.md`를 보고 점수표를 채우면 된다.
