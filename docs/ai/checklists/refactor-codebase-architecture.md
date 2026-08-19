# Checklist kiểm thử thủ công — refactor-codebase-architecture

## Tóm tắt xác minh
- Tổng số ca kiểm thử: 28
- 🟢 Đã xác minh đầy đủ: 8/28 (29%)
- 🟡 Có bằng chứng một phần: 16/28 (57%)
- 🔴 Cần người kiểm tra: 4/28 (14%)
- Tiêu chí chấp nhận được bao phủ đầy đủ: 4/28
- Bằng chứng chi tiết: docs/ai/verifications/refactor-codebase-architecture.md

## Chú giải bằng chứng
- 🟢 Bằng chứng trực tiếp cho thấy ca kiểm thử đã đạt đầy đủ.
- 🟡 Bằng chứng gián tiếp, chưa đầy đủ, hoặc chưa bao phủ toàn bộ phạm vi.
- 🔴 Chưa chạy, không đạt, bị chặn, có bằng chứng mâu thuẫn, hoặc đặc tả chưa rõ.

## Các ca kiểm thử
- [ ] 🟡 TC-001 [AC1] Chạy characterization tests cho OpenAI Chat Completions, Responses, Anthropic Messages, non-stream, stream và terminal stream error → tất cả path liên quan được bảo vệ. — AI: Test hiện có pass, ma trận đầy đủ chưa độc lập kiểm tra
- [ ] 🟡 TC-002 [AC2] Chạy full regression suite và kiểm tra endpoint, envelope, status, SSE, CLI, TUI, config → không có regression observable. — AI: Gate pass, runtime chưa chạy
- [ ] 🟢 TC-003 [AC3] Kiểm tra CLI và TUI khởi tạo graph qua cùng composition owner → không còn duplicate graph wiring. — AI: Code inspection xác nhận `app.Build`
- [ ] 🟡 TC-004 [AC4] Gọi TUI `Runtime.Load` với config hợp lệ và lỗi build → graph cũ chỉ được thay sau success. — AI: Code đúng, thiếu focused state test
- [ ] 🟡 TC-005 [AC5] Kiểm tra composition build success/error với logger nil/non-nil → không mở listener và logger được gắn đúng. — AI: Có test lỗi config, thiếu logger parity
- [ ] 🟢 TC-006 [AC6] Gọi `Service.Chat` với request snapshot trước/sau trên success và error → request caller không đổi. — AI: Unit test immutability pass
- [ ] 🟢 TC-007 [AC7] Gọi `Service.Stream` với request snapshot trước/sau → chỉ internal copy có `Stream=true`. — AI: Unit test stream copy pass
- [ ] 🟢 TC-008 [AC8] Dùng fake provider capture request → provider nhận upstream model/provider, caller giữ alias. — AI: Capture test pass
- [ ] 🟡 TC-009 [AC9] Chạy các nhánh no account, auth/reauth, retryable, non-retryable, timeout/rate-limit và unknown model → mapping đúng. — AI: Full tests pass, matrix chưa đủ
- [ ] 🟡 TC-010 [AC10] Chạy stream setup lỗi trước channel và sau event → failover chỉ xảy ra trước downstream bytes. — AI: Setup test hẹp, thiếu event matrix
- [ ] 🟢 TC-011 [AC11] Kiểm tra compile contract của `provider.Provider` và provider API-key → không còn dummy OAuth refresh. — AI: Interface compile và code inspection pass
- [ ] 🔴 TC-012 [AC12] Kiểm tra shared transport → không sở hữu tokenstore/OAuth refresh lifecycle. — AI: Ownership split chưa hoàn tất
- [ ] 🟡 TC-013 [AC13] Chạy focused tests Codex OAuth, Gemini API-key và Command Code browser-key → behavior giữ nguyên. — AI: Provider tests pass, coverage một phần
- [ ] 🔴 TC-014 [AC14] Kiểm tra Command Code request, NDJSON, error/redaction files → capability nằm ở owner riêng. — AI: File split chưa hoàn tất
- [ ] 🟢 TC-015 [AC15] Chạy toàn bộ Command Code parser, terminal error, oversized line, classification và redaction tests → pass. — AI: Provider test suite pass
- [ ] 🔴 TC-016 [AC16] Kiểm tra file Chat Completions, Responses và shared helper → lifecycle không bị trộn. — AI: OpenAI split chưa hoàn tất
- [ ] 🟡 TC-017 [AC17] Chạy OpenAI và Anthropic integration tests cho non-stream/stream success → pass. — AI: Test hiện có pass, matrix chưa đầy đủ
- [ ] 🟡 TC-018 [AC18] Chạy invalid field, unknown model, auth error và terminal stream error → envelope/terminal behavior đúng. — AI: Test hiện có pass, matrix chưa đầy đủ
- [ ] 🟡 TC-019 [AC19] Kiểm tra config schema/duration, I/O/path, defaults và validation files → exported symbols/YAML tags giữ nguyên. — AI: Behavior pass, file split thiếu
- [ ] 🟡 TC-020 [AC20] Chạy config validation, defaults, round-trip và permission tests → pass và permission không đổi. — AI: Config tests pass, permission chưa riêng
- [ ] 🟡 TC-021 [AC21] Kiểm tra TUI model/update, commands/forms, rendering → key, label và message flow giữ nguyên. — AI: Unit tests pass, runtime bị chặn
- [ ] 🟡 TC-022 [AC22] Chạy TUI unit tests và smoke load/start-stop/navigation → không có regression quan sát được. — AI: Unit tests pass, smoke bị chặn
- [ ] 🟡 TC-023 [AC23] Đưa forbidden dependency vào architecture fixture → test fail và báo importer/imported rõ ràng. — AI: Current graph pass, fixture fail chưa chạy
- [ ] 🟡 TC-024 [AC24] Đưa forbidden generic package trực tiếp dưới `internal` vào fixture → test fail, không áp line-count threshold. — AI: Rule hiện có, fixture fail chưa chạy
- [ ] 🟢 TC-025 [AC25] Chạy architecture test trên production import graph hiện tại → pass. — AI: Architecture test pass
- [ ] 🟢 TC-026 [AC26] Chạy `./scripts/check.sh` → fail-fast theo thứ tự test, vet, build và command có trong `CLAUDE.md`. — AI: Final gate pass
- [ ] 🟡 TC-027 [AC27] Đối chiếu `docs/dev/architecture.md` với composition path, provider boundary và gate command → khớp implementation. — AI: Composition/gate khớp, migration còn thiếu
- [ ] 🔴 TC-028 [AC28] Chạy final gate, `git diff --check`, security inspection và public-contract regression review → toàn bộ scope hoàn tất. — AI: Gate pass nhưng scope chưa hoàn tất

## Lỗ hổng đặc tả / Sai lệch
- Implementation drift: các AC về split hotspot/provider ownership chưa hoàn tất.
- Runtime smoke đã chạy với `go run . start --config config.yaml` tại `127.0.0.1:18080`; TUI interactive smoke vẫn cần người kiểm tra.

## Xác nhận của người kiểm tra
- [ ] Đã hoàn thành các ca kiểm thử còn ô checkbox.
- [ ] Đã chấp nhận các ca kiểm thử màu xanh và không cần kiểm tra lại.
- [ ] Tính năng đạt yêu cầu hoặc đã ghi rõ các ca kiểm thử chưa đạt.

## Nguồn
- Đặc tả đã duyệt: docs/ai/specs/refactor-codebase-architecture.md
- Bằng chứng xác minh: docs/ai/verifications/refactor-codebase-architecture.md
