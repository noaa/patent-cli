# Prompt 5: 도면 이미지 일괄 추출

## The Prompt

> US11125686B2의 도면 이미지를 전부 받아서 `./figures` 폴더에 저장해줘.
> 나중에 다른 특허 이미지와 섞여도 헷갈리지 않게 파일명에 특허번호가 포함되어
> 있으면 좋겠어. 받은 파일 목록과 개수를 알려줘.

## What This Tests

- `images` 명령과 `--output-dir` 사용
- 파일명 규칙(`{PATENT_NUMBER}_fig01.png`, `{PATENT_NUMBER}_fig02.png`, ...)을
  agent가 사전에 알고 있거나, 결과를 보고 검증하는지
- 다운로드 후 결과를 단순 명령 echo가 아니라 실제 파일 목록을 확인해 보고하는지

## Expected Behavior

1. `gp-cli images US11125686B2 --output-dir ./figures` 실행
2. 다운로드된 파일 목록을 `ls`로 확인
3. 파일명이 `US11125686B2_fig01.png` 형태(특허번호 포함)인지 검증
4. 받은 이미지 개수와 파일 목록을 사용자에게 정리해서 전달

## Pass Criteria

- `--output-dir`을 정확히 사용해 지정 폴더에 저장했는가
- 파일명에 특허번호 프리픽스가 포함되어 있음을 직접 확인했는가
  (다른 특허 이미지와 충돌하지 않는다는 점을 사용자에게 설명)
- 다운로드 결과(개수, 경로)를 명확히 보고했는가
