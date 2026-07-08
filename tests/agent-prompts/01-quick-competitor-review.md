# Prompt 1: 경쟁사 특허 빠른 1차 리뷰

## The Prompt

> 경쟁사가 특허 US12514139B2를 새로 등록했다는 소식을 들었어. 오전 미팅 전까지
> 시간이 별로 없으니까, 제목·출원인·출원일만 빠르게 한 줄로 보여주고, 청구항
> 전문은 나중에 검토할 수 있게 `./review` 폴더에 텍스트 파일로 저장해줘.

## What This Tests

- `lookup`의 기본 JSON 출력과 `--fields`(복수)를 이용한 필드 선택
- `--format text`로 사람이 읽기 좋은 출력 만들기
- `--field`(단수) + `--output-dir` 조합으로 단일 필드를 파일로 저장하기
- `--field`와 `--fields`의 차이를 상황에 맞게 선택하는 능력

## Expected Behavior

1. 먼저 `gp-cli lookup US12514139B2 --fields title,assignee,filing_date --format text`
   같은 명령으로 핵심 정보를 추출
2. 청구항은 `gp-cli lookup US12514139B2 --field claims --output-dir ./review`로 저장
3. 저장된 파일 경로를 사용자에게 알려줌
4. raw JSON을 그대로 화면에 쏟아내지 않고 핵심만 정리해 보고

## Pass Criteria

- `--fields`에 콤마로 구분된 필드 목록을 올바르게 전달했는가
- `./review/US12514139B2.txt` (또는 동등한 경로)에 청구항 파일이 생성되었는가
- 사용자에게 "제목/출원인/출원일"을 한눈에 보이는 형태로 요약했는가
- 불필요하게 전체 JSON을 두 번 이상 출력하지 않았는가
