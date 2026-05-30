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

## 사용법

### 특허 메타데이터 조회

```sh
# JSON 형식 (기본)
gp-cli lookup US12514139B2

# 사람이 읽기 편한 텍스트 형식
gp-cli lookup US12514139B2 --format text

# 특정 필드만 출력
gp-cli lookup US12514139B2 --fields title,assignee,filing_date

# 제목만 출력
gp-cli lookup US12514139B2 --field title

# JSON 한 줄 출력 (들여쓰기 없음)
gp-cli lookup US12514139B2 --minify

# 진행 메시지 억제 (스크립트에서 유용)
gp-cli lookup US12514139B2 --quiet

# 기계번역 영문 버전으로 조회
gp-cli lookup KR102355140B1 --language en

# 구조화된 청구항·설명 (opt-in 필드)
gp-cli lookup US12514139B2 --fields claims_structured,description_structured

# WO/PCT 특허
gp-cli lookup WO2022123456A1

# KR 특허
gp-cli lookup KR102355140B1
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
```

### 사용 가능한 필드 목록

```sh
gp-cli fields
```

---

## 출력 형식

| 옵션 | 설명 |
|------|------|
| `--format json` | JSON (기본값) — `{"ok": true, "results": {...}}` 형태로 출력 |
| `--format text` | 레이블 + 값 텍스트 |
| `--format tsv` | 탭 구분 (Excel 붙여넣기용) |
| `--output-dir DIR` | 파일로 저장 |
| `--minify` | JSON 한 줄 출력 (들여쓰기 없음) |

## 전역 플래그

| 플래그 | 설명 |
|--------|------|
| `--quiet`, `-q` | stderr 진행 메시지 억제 (스크립트용) |
| `--minify` | JSON 한 줄 출력 (들여쓰기 없음) |
| `--verbose`, `-v` | 디버그 로그를 stderr에 출력 |
| `--language LANG` | Google 기계번역 페이지로 조회 (예: `en`). 비영어권 특허 영문 조회에 유용. |

### Opt-in 구조화 필드

기본 출력에 포함되지 않으며, `--field` 또는 `--fields`로 명시적으로 요청해야 합니다:

| 필드 | 설명 |
|------|------|
| `claims_structured` | 청구항을 `{"number": "1", "text": "…"}` 객체 배열로 반환 |
| `description_structured` | 설명 단락을 `{"number": "1", "id": "…", "text": "…"}` 객체 배열로 반환 |

```sh
gp-cli lookup US12514139B2 --fields claims_structured
gp-cli lookup US12514139B2 --fields claims_structured,description_structured
gp-cli fields   # opt-in 필드 포함 전체 필드 목록 확인
```

---

## 설정 (프록시 / CA 인증서)

회사 내부망에서 프록시를 사용하는 경우:

```sh
gp-cli configure
```

설정 파일 위치: `~/.patent-cli.toml`

---

## 버전 확인

```sh
gp-cli --version
```
