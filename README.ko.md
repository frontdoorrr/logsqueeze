[English](README.md) | [한국어](README.ko.md)

# logsqueeze

[![Go Reference](https://pkg.go.dev/badge/github.com/frontdoorrr/logsqueeze.svg)](https://pkg.go.dev/github.com/frontdoorrr/logsqueeze)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Claude Code와 모든 MCP 호환 에이전트를 위한 로그 압축 도구.

---

## 왜 logsqueeze인가

대용량 로그 파일을 그대로 붙여넣으면 Claude Code의 컨텍스트가 금방 가득 찹니다. logsqueeze는 XDrain 알고리즘을 사용해 수백만 줄의 로그를 소수의 템플릿으로 압축한 뒤 모델에 전달합니다. **모든 처리는 로컬에서만 이루어지며, 데이터는 외부로 전송되지 않습니다.**

---

## 출력 예시

```
Compressed 1,200,000 lines → 12 templates (100,000x compression)

[x1,185,000] worker ready shard=<*> [1..48]
  samples: 14:22:11 worker ready shard=1 | 14:22:12 worker ready shard=2

[x12,000] pool acquire <*> [20..480ms p50=240ms]
  samples: 14:22:15 pool acquire 240ms | 14:22:16 pool acquire 480ms

[x3] ERROR psycopg2.OperationalError: connection <*> [timeout,refused,reset]
  samples: 14:22:16 ERROR psycopg2.OperationalError: connection timeout
```

`<*>` 슬롯은 해당 위치에서 변한 값들을 요약합니다 — 숫자라면 min/max/p50, 문자열이라면 고유값 목록으로 표시됩니다.

---

## 설치

```bash
go install github.com/frontdoorrr/logsqueeze@latest
```

---

## Claude Code 연동

`~/.claude.json`에 추가:

```json
{
  "mcpServers": {
    "logsqueeze": {
      "command": "logsqueeze",
      "args": ["mcp", "serve"]
    }
  }
}
```

Claude Code를 재시작하면 아래 세 가지 툴이 모든 세션에서 사용 가능해집니다.

---

## MCP 툴

| 툴 | 설명 |
|----|------|
| `compress_logs` | 인라인으로 전달한 로그 텍스트를 압축 |
| `compress_file` | 파일 경로를 받아 로그 파일을 읽고 압축 |
| `compress_command` | 쉘 커맨드를 실행하고 stdout을 압축 |

**Claude Code에서 호출하는 예시:**

```
# 로그 텍스트 직접 전달
compress_logs(logs="<로그 내용 붙여넣기>")

# 로그 파일 경로
compress_file(path="/var/log/app.log")

# 실시간 커맨드 출력
compress_command(command="kubectl logs -n prod deploy/api --tail=5000")
compress_command(command="docker logs --tail=2000 my-container")
compress_command(command="journalctl -u nginx --since '1 hour ago'")
```

---

## 작동 원리

logsqueeze는 **XDrain** 로그 템플릿 마이닝 알고리즘을 구현합니다. prefix 트리를 사용해 구조적 유사도 기준으로 로그 라인을 그룹화하며, 순서 편향을 제거하기 위해 배치를 셔플한 뒤 처리합니다. 각 라인은 토큰 로테이션을 통해 여러 트리에서 매칭을 시도하고 다수결로 그룹을 결정합니다.

가변 위치는 `<*>` 와일드카드로 치환되며, 숫자 슬롯은 수집된 모든 값으로 정확한 min/max/p50을 계산하고, 문자열 슬롯은 최대 6개의 고유값을 샘플로 보여줍니다.

결과는 LLM이 로그의 통계적 패턴을 원본 볼륨 없이 파악할 수 있는 간결한 요약입니다.

---

## 지원 로그 포맷

대부분의 경우 `format` 옵션 없이 자동 감지됩니다.

| 포맷 | 예시 |
|------|------|
| ISO 8601 syslog | `2024-01-01T14:22:11Z INFO worker ready` |
| 공백 구분 syslog | `2024-01-01 14:22:11 [INFO] worker ready` |
| 슬래시 날짜 syslog | `2024/01/01 14:22:11 INFO worker ready` |
| JSON (구조화 로그) | `{"time":"...","level":"info","msg":"worker ready"}` |
| 레벨만 있는 경우 | `INFO: worker ready` / `[ERROR] pool acquire 240ms` |
| 일반 텍스트 | `worker ready shard=1` |

JSON 필드 지원: `time`/`timestamp`/`ts`/`@timestamp`, `level`/`severity`/`lvl`, `msg`/`message`/`log`/`text`.

---

## 라이선스

MIT — [LICENSE](LICENSE) 참고.
