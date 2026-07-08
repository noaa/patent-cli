# Prompt 3: 한국어 특허 번역 열람 + 구조화 청구항 분석

## The Prompt

> 한국 특허 KR102355140B1 내용을 영어로 보고 싶어. 제목·초록을 영문으로 보여주고,
> 청구항을 독립항/종속항 구조까지 분석해서 `./translated` 폴더에 저장해줘.
> 혹시 번역된 페이지라서 분석에 빠지는 정보가 있으면 알려줘.

## What This Tests

- `--language en`으로 Google 기계번역 페이지를 가져오는 흐름
- `--fields claims_structured,description_structured`로 옵트인 구조화 필드 요청
- `--output-dir`과 구조화 필드를 함께 사용할 때의 저장 동작
- 번역 페이지에서 `_warnings`(`TRANSLATED_PAGE_NO_TYPE_INFO` 등)를 인지하고
  사용자에게 한계를 설명하는 능력

## Expected Behavior

1. `gp-cli lookup KR102355140B1 --language en --fields title,abstract` 등으로
   기본 번역 정보 확인
2. `--fields claims_structured,description_structured --output-dir ./translated`로
   구조화 데이터 저장
3. 응답에 포함된 `_warnings`를 확인하고, 번역 페이지에서는 `type`/`depends_on`
   같은 구조 정보가 빠질 수 있음을 사용자에게 안내
4. 독립항/종속항 개수 등 핵심 통계를 요약

## Pass Criteria

- `--language en` 플래그로 영문 번역 데이터를 정상적으로 가져왔는가
- `claims_structured`/`description_structured`가 옵트인 필드라는 점을 알고
  명시적으로 요청했는가 (`fields` 명령으로 사전 확인했다면 가산점)
- `_warnings`가 있을 경우 이를 사용자에게 명확히 전달했는가
- 저장된 파일 경로와 내용 요약을 함께 제시했는가
