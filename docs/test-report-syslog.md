# logsqueeze 테스트 리포트 — macOS install.log

## 테스트 환경

| 항목 | 값 |
|------|----|
| 대상 파일 | `/var/log/install.log` |
| 툴 | `compress_file` |
| 알고리즘 | XDrain (depth=4, simTh=0.4, batchSize=2000) |
| 테스트 일자 | 2026-06-08 |

---

## 압축 결과

```
Compressed 287,746 lines → 7,275 templates (40x compression)
```

| 지표 | 값 |
|------|----|
| 원본 라인 수 | 287,746 |
| 압축 후 템플릿 수 | 7,275 |
| 압축률 | **40x** |

---

## 상위 템플릿 분석

### 1위 — softwareupdated 백그라운드 스캔 (x19,891)

```
[x19,891] +09 MacBook-Pro <*> <*> <*> [1..27.80s p50=1.70s] <*> <*> <*>
  samples: softwareupdated[303]: SUOSUServiceDaemon: BackgroundActivity: Perform background scan
           softwareupdated[303]: Refreshing available updates from scan
           softwareupdated[303]: BackgroundActivity: Finished Background Check Activity
```

- `softwareupdated`, `nbagent`, `loginwindow`, `suhelperd` 등 6개 프로세스의 로그를 하나의 템플릿으로 통합
- 소요 시간 슬롯: 1초~27.8초, p50=1.7초 (자동 업데이트 스캔 시간 분포)

---

### 2위 — 소프트웨어 업데이트 권한 거부 (x12,086)

```
[x12,086] +09 MacBook-Pro <*> <*> Not authorized to clear preference <*>
  samples: softwareupdated[305]: Not authorized to clear preference TimeOfSemiSplatCompletion
           softwareupdated[305]: Not authorized to clear preference DDMPastDuePaddedEnforcementDateKey
```

- 반복적인 권한 거부 로그를 단일 템플릿으로 수렴
- 데이터 전송량 슬롯: -0.2KB~1070KB, p50=0KB

---

### 3위 — macOS 버전 descriptor (x10,149)

```
[x10,149] +09 MacBook-Pro <*> <*> <*> [0..15.20]
  samples: softwareupdated[333]: JS: 15.2
           softwareupdated[333]: CacheDeleteSetCacheable: 1
           installd[15990]: installd: Starting
```

---

### 4위 — oahd Rosetta 번역 로그 (x9,405)

```
[x9,405] +09 MacBook-Pro <*> <*> <*> <*>
  samples: installd[54784]: oahd translated /Applications/Xcode.app/.../libswiftVision.dylib
           system_installd[55019]: oahd translated /Library/Developer/.../SimStreamProcessorObjects
```

- Xcode 및 시스템 라이브러리 Rosetta 번역 이벤트 9,400여 건이 하나의 패턴으로 수렴

---

### 주목할 만한 패턴

**반복 에러 — installation-check 스크립트 TypeError (누적 ~26,000건)**

```
[x5,412] softwareupdated[308]: Package Authoring: error running installation-check script:
         TypeError: null is not an object (evaluating <*> ['cpuFeatures.split', '...version'])
```

여러 `softwareupdated` 프로세스 버전(302, 303, 308, 324, 329, 333)에 걸쳐 동일한 에러가 반복 발생.
합산하면 전체 로그의 약 9%를 차지하는 주요 노이즈 패턴.

**설정값 로그 (x6,873 + x2,862)**

```
[x6,873] <*> [settings,autoUpdate,mdm,...] = <*> [false;,true;]
[x2,862] <*> [commandLine,notification,installTonight,...] = <*> [true;,false;]
```

Boolean 설정 덤프가 키-값 패턴으로 묶임.

**다운로드 진행률 (x2,461)**

```
[x2,461] Progress: <*> [1176989316/1225436278, ...]
```

숫자 슬롯으로 수렴하지 못하고 분수 형태 문자열로 샘플링됨 → 개선 여지 있음.

**macOS 버전 히스토리 확인됨**

```
SU:macOS Sequoia 15.3.1 / 15.4 / 15.4.1 / 15.5 / 15.6 / 15.7.7
SU:macOS Tahoe 26.0.1 / 26.2 / 26.3 / 26.5.1
```

업데이트 경로가 descriptor 슬롯에서 읽힘.

---

## 관찰 사항

### 잘 동작한 것

- **프로세스 ID 통합**: `softwareupdated[303]`, `softwareupdated[305]` 등 PID가 다른 동일 프로세스를 하나의 `<*>` 슬롯으로 묶음
- **시간 슬롯 통계**: 스캔 소요 시간 1~27.8초, p50=1.7초 등 수치 범위 자동 계산
- **macOS syslog 포맷 자동 감지**: `2025-05-16 17:09:55 +09 MacBook-Pro` 형태 파싱 정상 동작

### 개선 여지

- **템플릿 수 7,275개**: install.log는 구조가 다양한 시스템 로그라 압축률 40x는 낮은 편. 앱 서버 로그처럼 패턴이 반복되는 로그에서는 훨씬 높은 압축률 기대 가능
- **`Progress: N/M` 패턴**: 분수 형태 문자열이 숫자로 파싱 안 됨 (슬롯이 문자열 샘플 목록으로 표시)
- **단일 토큰 라인**: `[x4,592] )`, `[x2,019] }` 등 JSON 멀티라인 파싱 잔재가 별도 템플릿으로 분류됨

---

## 결론

`/var/log/install.log` 기준 **287,746줄 → 7,275 템플릿 (40x 압축)** 달성. macOS 소프트웨어 업데이트 로그 특성상 프로세스 종류와 메시지 패턴이 다양해 압축률이 낮게 나왔으나, 반복 에러 패턴과 백그라운드 스캔 로그는 정상적으로 수렴됨. 앱 서버나 Kubernetes 로그처럼 균일한 패턴의 로그에서 더 높은 효율이 예상됨.
