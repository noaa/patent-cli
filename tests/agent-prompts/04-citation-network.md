# Prompt 4: 인용 네트워크 데이터 수집

## The Prompt

> US12514139B2가 인용한 선행기술 특허 번호들만 깔끔한 목록으로 뽑아줘.
> jq 같은 걸로 후처리할 수 있게 만들어주면 좋겠어. 그리고 이 특허를 인용한
> 후행 특허는 몇 건이나 되는지도 알려줘.

## What This Tests

- `--field backward_citations` / `--field forward_citations`로 인용 데이터 추출
- 반환되는 JSON이 문자열 배열이 아니라 `{publication_number, priority_date,
  assignee, ...}` 형태의 **객체 배열**이라는 점을 스스로 알아내는지
- `jq` 파이프라인을 올바른 경로(`jq -r '.[].publication_number'`)로 구성하는지
- 스키마를 사전에 확인하기 위해 `gp-cli fields`나 작은 샘플 호출을 활용하는지

## Expected Behavior

1. `gp-cli lookup US12514139B2 --field backward_citations`로 데이터 형태를 먼저 확인
2. 단순히 `jq -r '.[]'`로 끝내지 않고, 객체 배열임을 인지해
   `jq -r '.[].publication_number'`처럼 올바른 경로로 추출
3. `forward_citations`에 대해서도 동일하게 처리해 건수를 집계
4. 최종 결과를 특허번호 리스트 + 건수 요약으로 제시

## Pass Criteria

- 인용 데이터가 객체 배열임을 인지하고 잘못된 jq 경로(`'.[]'`만 사용)로 끝내지
  않았는가
- `backward_citations`와 `forward_citations`를 구분해서 다뤘는가
- 최종 출력이 raw JSON 덤프가 아니라 정리된 번호 목록/집계인가
- 스키마를 추측이 아니라 실제 호출로 확인하는 절차를 거쳤는가
