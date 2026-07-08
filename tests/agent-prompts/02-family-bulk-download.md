# Prompt 2: 패밀리 특허 일괄 다운로드 + 요약 정리

## The Prompt

> 클라이언트가 US8725880B2 특허의 해외 패밀리 PDF를 모아달라고 했어. 번호
> 목록은 내가 `patent_list.txt`에 적어둘게 (US/EP/KR/JP/CN 각 1건씩). 이 번호들의
> PDF를 한 폴더에 받고, 제목·출원인·공개일을 정리한 TSV 파일도 하나 만들어줘.
> 목록 중에 실제로 존재하지 않는 번호가 섞여 있을 수도 있으니 그런 건 그냥
> 건너뛰면 돼.

(agent에게 `patent_list.txt`를 직접 만들도록 하거나, 미리
`tests/integration/family_group_test.go`에 등장하는 US8725880B2 패밀리 중
국가별 1건씩을 골라 파일로 준비해 둘 것)

## What This Tests

- `download --input-file` 또는 루프를 통한 일괄 PDF 다운로드
- `lookup --fields ... --format tsv` 반복 호출과 `--no-header`로 헤더 중복 방지
- `--delay`로 요청 간격을 둬 봇 차단을 피하는 패턴
- `--quiet`로 진행 메시지를 억제하면서도 에러는 stderr로 분리되어 파이프라인을
  오염시키지 않는지 확인
- 존재하지 않는 특허(exit code 4)를 만나도 루프가 중단되지 않고 계속 진행하는지

## Expected Behavior

1. 번호 목록 파일을 만들거나 확인
2. `gp-cli download <번호> --output-dir ./family_pdfs --quiet --delay 1000` 형태로
   순회하며 다운로드 (실패 건은 건너뜀)
3. 첫 호출은 헤더 포함, 이후 호출은 `--no-header`를 붙여 TSV를 이어붙임
4. 최종적으로 받은 PDF 개수와 TSV 행 수를 사용자에게 보고

## Pass Criteria

- PDF가 존재하는 번호만큼 `./family_pdfs/`에 저장되었는가 (실패 건은 정상적으로 스킵)
- TSV에 헤더가 한 번만 등장하는가 (`--no-header` 적절히 사용)
- 에러 JSON이 TSV 파일에 섞여 들어가지 않았는가 (stderr 분리 확인)
- 루프 중 `--delay`를 사용해 요청 간격을 확보했는가
