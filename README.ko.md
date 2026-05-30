# gp-cli — Google Patents CLI

Google Patents에서 특허 메타데이터를 조회하고, PDF·도면 이미지를 다운로드하는 커맨드라인 도구입니다.

> English documentation: [README.md](README.md)

---

## 설치

### macOS

터미널(Terminal)을 열고 아래 명령어를 복사·붙여넣기 하세요:

```sh
curl -fsSL https://raw.githubusercontent.com/noaa/patent-cli/main/install.sh | sh
```

설치 후 터미널을 **새로 열면** `gp-cli` 명령을 바로 사용할 수 있습니다.

---

### Windows

**방법 A — PowerShell** (시작 메뉴 → "PowerShell" 검색):

```powershell
irm https://raw.githubusercontent.com/noaa/patent-cli/main/install.ps1 | iex
```

**방법 B — 명령 프롬프트(CMD)** (시작 메뉴 → "cmd" 검색):

```cmd
curl -fsSL https://raw.githubusercontent.com/noaa/patent-cli/main/install.bat -o "%TEMP%\gp-cli-install.bat" && "%TEMP%\gp-cli-install.bat"
```

> Windows 10 build 1803 이상이 필요합니다 (curl·tar 내장).

설치 후 터미널을 **새로 열면** `gp-cli` 명령을 바로 사용할 수 있습니다.

---

### 수동 설치 (직접 다운로드)

자동 설치가 안 될 경우 [Releases 페이지](https://github.com/noaa/patent-cli/releases)에서 직접 다운로드하세요:

| OS | 파일 |
|----|------|
| macOS (M1/M2/M3) | `gp-cli-darwin-arm64.tar.gz` |
| macOS (Intel) | `gp-cli-darwin-amd64.tar.gz` |
| Windows | `gp-cli-windows-amd64.exe.zip` |
| Linux | `gp-cli-linux-amd64.tar.gz` |

압축 해제 후 `gp-cli`(또는 `gp-cli.exe`)를 원하는 폴더에 넣으면 됩니다.

---

### 소스 코드로 직접 빌드

[Go 1.21+](https://go.dev/dl/)가 설치되어 있어야 합니다.

```sh
git clone https://github.com/noaa/patent-cli.git
cd patent-cli
go build -o gp-cli ./cmd/gp-cli/
```

빌드된 바이너리를 `PATH`에 있는 디렉토리로 이동합니다:

```sh
# macOS / Linux
mv gp-cli ~/.local/bin/

# Windows (PowerShell)
Move-Item gp-cli.exe $env:LOCALAPPDATA\gp-cli\gp-cli.exe
```

---

## 사용법

### 특허 메타데이터 조회

```sh
# JSON 형식 (기본값)
gp-cli lookup US12514139B2

# 사람이 읽기 편한 텍스트 형식
gp-cli lookup US12514139B2 --format text

# 특정 필드만 출력
gp-cli lookup US12514139B2 --fields title,assignee,filing_date

# 단일 필드 값만 출력 (plain text)
gp-cli lookup US12514139B2 --field title

# JSON 한 줄 출력 (들여쓰기 없음)
gp-cli lookup US12514139B2 --minify

# 진행 메시지 억제 (스크립트에서 유용)
gp-cli lookup US12514139B2 --quiet

# 비영어권 특허 기계번역 영문으로 조회
gp-cli lookup KR102355140B1 --language en

# 구조화된 청구항·설명 (opt-in 필드)
gp-cli lookup US12514139B2 --fields claims_structured,description_structured

# 파일로 저장 (stdout 출력 억제)
gp-cli lookup US12514139B2 --output-dir ./output

# 단일 필드를 파일로 저장
gp-cli lookup US12514139B2 --field claims --output-dir ./output
```

### PDF 다운로드

```sh
gp-cli download US12514139B2
gp-cli download US12514139B2 --output-dir ./pdfs
```

### 도면 이미지 다운로드

```sh
gp-cli images US12514139B2
gp-cli images US12514139B2 --output-dir ./figures
# US12514139B2_fig01.png, US12514139B2_fig02.png, ... 형태로 저장됨
```

### 사용 가능한 필드 목록

```sh
gp-cli fields
```

---

## 출력 형식

| 옵션 | 설명 |
|------|------|
| `--format json` | JSON (기본값) — `{"ok": true, "results": {...}}` 형태 |
| `--format text` | 레이블 + 값 텍스트 |
| `--format tsv` | 탭 구분 (Excel 붙여넣기용) |
| `--output-dir DIR` | 파일로 저장; stdout 출력 억제 |
| `--no-header` | TSV 헤더 행 생략 (루프에서 행 추가 시 유용) |
| `--minify` | JSON 한 줄 출력 (들여쓰기 없음) |

## 전역 플래그

| 플래그 | 설명 |
|--------|------|
| `--quiet`, `-q` | stderr 진행 메시지 억제 (스크립트용) |
| `--minify` | JSON 한 줄 출력 (들여쓰기 없음) |
| `--verbose`, `-v` | 디버그 로그를 stderr에 출력 |
| `--language LANG` | Google 기계번역 페이지로 조회 (예: `en`). 비영어권 특허 영문 조회에 유용. |

---

## 에러 처리

에러는 **stderr**에 구조화된 JSON으로 출력되며, 유형별 exit code와 함께 종료됩니다.

```json
{
  "ok": false,
  "error": {
    "type": "NOT_FOUND",
    "message": "patent not found: US99999999X1"
  }
}
```

| Exit code | 의미 |
|-----------|------|
| `0` | 성공 |
| `1` | 일반 오류 |
| `4` | 특허 미발견 |
| `6` | 서버 오류 (봇 차단, 5xx) |

에러가 stderr로 분리되므로 stdout은 항상 순수한 데이터입니다. 파일 리디렉션이나 `jq` 파이프라인에서 에러 JSON 혼입 걱정 없이 사용할 수 있습니다.

```sh
# 안전: TSV 데이터만 파일에 저장, 에러는 터미널에 표시
while IFS= read -r num; do
  gp-cli lookup "$num" --format tsv --quiet
done < list.txt > results.tsv
```

---

## 스크립트 & 파이프라인

### 헤더 중복 없이 TSV 일괄 생성

```sh
first=1
while IFS= read -r num; do
  if [ $first -eq 1 ]; then
    gp-cli lookup "$num" --fields publication_number,title,assignee --format tsv --quiet --delay 1000
    first=0
  else
    gp-cli lookup "$num" --fields publication_number,title,assignee --format tsv --quiet --no-header --delay 1000
  fi
done < patent_list.txt > summary.tsv
```

### `jq`로 인용 특허번호 추출

```sh
gp-cli lookup US8725880B2 --field backward_citations --quiet \
  | jq -r '.[].publication_number'
```

### 독립항만 필터링

```sh
gp-cli lookup US12514139B2 --fields claims_structured --quiet \
  | jq '[.results.claims_structured[] | select(.type == "independent")]'
```

### 청구항 1과 직접 종속항만 추출

```sh
gp-cli lookup US12514139B2 --fields claims_structured --quiet \
  | jq '[.results.claims_structured[] | select(.number == "1" or (.depends_on // [] | any(. == "1")))]'
```

### 여러 특허 도면 일괄 다운로드 (파일명 충돌 없음)

```sh
# US11125686B2_fig01.png, EP3025568B1_fig01.png, ... 형태로 저장
for num in US11125686B2 EP3025568B1; do
  gp-cli images "$num" --output-dir ./figures --quiet --delay 1000
done
```

---

## Opt-in 구조화 필드

기본 출력에 포함되지 않으며, `--field` 또는 `--fields`로 명시적으로 요청해야 합니다:

| 필드 | 스키마 |
|------|--------|
| `claims_structured` | `{"number": "1", "type": "independent"\|"dependent", "depends_on": ["N"], "text": "…"}` |
| `description_structured` | `{"number": "1", "id": "…", "text": "…"}` |

`type`과 `depends_on`은 HTML `<claim-ref>` 태그를 통해 US/EP 특허에서만 감지됩니다. 번역 페이지(`--language en`)나 KR/JP/CN 원문에서는 이 마크업이 없어 해당 필드가 생략됩니다.

```sh
gp-cli lookup US12514139B2 --fields claims_structured
gp-cli lookup US12514139B2 --fields claims_structured,description_structured
gp-cli fields   # opt-in 필드 포함 전체 필드 목록 확인
```

### 데이터 품질 경고 (`_warnings`)

`claims_structured`를 요청하면 JSON 엔벨로프의 `ok`, `results` 옆에 `_warnings` 배열이 추가될 수 있습니다. 문제가 없으면 키 자체가 생략됩니다.

```json
{
  "ok": true,
  "results": { "claims_structured": [...] },
  "_warnings": [
    {
      "field": "claims_structured",
      "code": "TRANSLATED_PAGE_NO_TYPE_INFO",
      "message": "Claim type and dependency data unavailable; page was served as a machine translation"
    },
    {
      "field": "claims_structured",
      "code": "SUSPICIOUSLY_SHORT_CLAIM_TEXT",
      "message": "Claim(s) 1 have text ≤ 20 chars; likely translation artifacts — verify against source"
    }
  ]
}
```

| 경고 코드 | 의미 |
|---|---|
| `TRANSLATED_PAGE_NO_TYPE_INFO` | 모든 청구항에 `type`이 없음 — 기계번역 페이지로 `depends_on`도 없음 |
| `SUSPICIOUSLY_SHORT_CLAIM_TEXT` | 텍스트가 20자 이하인 청구항 존재 — 번역 아티팩트일 가능성 높음 |

---

## 설정 (프록시 / CA 인증서)

회사 내부망에서 프록시를 사용하는 경우:

```sh
gp-cli configure
```

설정 파일 위치: `~/.patent-cli.toml`

```toml
[proxy]
https = "http://proxy.corp:8080"
http  = "http://proxy.corp:8080"

[ssl]
ca_bundle = "/path/to/ca.pem"

[request]
delay_ms = 500   # 요청마다 대기 시간(ms) — 루프에서 봇 차단 방지용
```

---

## 버전 확인 & 업데이트

```sh
gp-cli --version
gp-cli update           # 최신 버전으로 자동 업데이트
gp-cli update --check   # 버전 정보만 확인 (업데이트 미진행)
```
