# Checklist kiểm thử thủ công — command-code-provider

## Tóm tắt xác minh
- Tổng số ca kiểm thử: 35
- 🟢 Đã xác minh đầy đủ: 31/35 (89%)
- 🟡 Có bằng chứng một phần: 4/35 (11%)
- 🔴 Cần người kiểm tra: 0/35 (0%)
- Tiêu chí chấp nhận được bao phủ đầy đủ: 8/10
- Bằng chứng chi tiết: docs/ai/verifications/command-code-provider.md

## Chú giải bằng chứng
- 🟢 Bằng chứng trực tiếp cho thấy ca kiểm thử đã đạt đầy đủ.
- 🟡 Bằng chứng gián tiếp, chưa đầy đủ, hoặc chưa bao phủ toàn bộ phạm vi.
- 🔴 Chưa chạy, không đạt, bị chặn, có bằng chứng mâu thuẫn, hoặc đặc tả chưa rõ.

## Các ca kiểm thử
- [ ] 🟢 TC-001 [AC1] — Cấu hình tối giản chỉ khai `enabled: true` + account command_code có `token_file` → `Validate()` đạt nhờ provider defaults. — AI: TestValidateCommandCodeDefaults đạt
- [ ] 🟢 TC-002 [AC1] — Cấu hình command_code thiếu `authorization_url` → `Validate()` trả lỗi nêu rõ tên provider. — AI: TestValidateCommandCodeMissingAuthorizationURL đạt
- [ ] 🟢 TC-003 [AC1] — Cấu hình command_code thiếu `redirect_port` (hoặc ≤ 0) → `Validate()` trả lỗi. — AI: TestValidateCommandCodeMissingRedirectPort đạt
- [ ] 🟢 TC-004 [AC1] — Account command_code thiếu `token_file` → `Validate()` trả lỗi. — AI: TestValidateCommandCodeMissingTokenFile đạt
- [ ] 🟢 TC-005 [AC2] — `config.example.yaml` chứa đủ 5 model alias theo D-003 với upstream đúng và alias không trùng provider khác. — AI: Đọc trực tiếp file, khớp D-003
- [ ] 🟢 TC-006 [AC3] — `auth command_code --account <name>` sinh URL dạng `https://commandcode.ai/studio/auth/cli?callback=http://localhost:<port>/callback&state=<hex 32 byte>`. — AI: TestBuildBrowserKeyURL đạt
- [ ] 🟡 TC-007 [AC3] — Mở browser thất bại vẫn in URL và tiếp tục chờ callback. — AI: Kiểm tra mã nguồn, chưa có test inject
- [ ] 🟡 TC-008 [AC4] — Callback có `state` khớp và apiKey hợp lệ `user_...` → ghi token file và `tokenstore.Load` đọc lại được. — AI: Handler test; luồng browser thật chưa chạy
- [ ] 🟢 TC-009 [AC4] — Callback thiếu `state` → từ chối, không ghi token. — AI: TestBrowserKeyCallbackStateMismatch đạt (missing → mismatch)
- [ ] 🟢 TC-010 [AC4] — Callback có `state` sai → từ chối, không ghi token. — AI: TestBrowserKeyCallbackStateMismatch đạt
- [ ] 🟢 TC-011 [AC4] — Callback thiếu apiKey hoặc apiKey không bắt đầu `user_` → từ chối, không ghi token. — AI: TestBrowserKeyCallbackInvalidKey/MissingKey đạt
- [ ] 🟡 TC-012 [AC4] — Callback hợp lệ lặp lại sau khi phiên đã hoàn tất → không ghi đè token, không block handler. — AI: Kiểm tra mã nguồn (send non-blocking), chưa test lặp
- [ ] 🟡 TC-013 [AC4] — Timeout hoặc huỷ luồng login → trả lỗi, không ghi token. — AI: Kiểm tra mã nguồn, chưa test luồng chờ
- [ ] 🟢 TC-014 [AC5] — `provider.Build("command_code", ...)` trả provider implement đủ interface; `Models()` trả alias theo config. — AI: factory + runtime GET /v1/models đạt
- [ ] 🟢 TC-015 [AC5] — Token file thiếu → typed auth error; qua proxy HTTP 401 và account `reauth_required`. — AI: TestCommandCodeCredentialErrors/missing_file đạt
- [ ] 🟢 TC-016 [AC5] — Token file lỗi parse → typed auth error; HTTP 401, `reauth_required`. — AI: TestCommandCodeCredentialErrors/parse_error đạt
- [ ] 🟢 TC-017 [AC5] — `AccessToken` rỗng hoặc sai prefix → typed auth error; HTTP 401, `reauth_required`. — AI: TestCommandCodeCredentialErrors đạt
- [ ] 🟢 TC-018 [AC5] — `accounts` hiển thị `reauth_required` cho token không hợp lệ, `ok`/`never` cho token hợp lệ. — AI: summary_test + runtime accounts đạt
- [ ] 🟢 TC-019 [AC6] — Body upstream có `params.stream=true` kể cả client gọi non-stream. — AI: TestCommandCodeBuildRequest đạt
- [ ] 🟢 TC-020 [AC6] — `threadId` bằng `x-session-id` và có đủ headers. — AI: TestCommandCodeBuildRequest đạt
- [ ] 🟢 TC-021 [AC6] — `params.system` nhận system/developer text theo thứ tự; `messages` là text blocks. — AI: TestCommandCodeBuildRequestSkipsSystemMessages đạt
- [ ] 🟢 TC-022 [AC6] — `config` khớp fixture; `stop`/`stop_sequences`/`metadata` không gửi upstream. — AI: TestCommandCodeBuildRequestDropsStopAndMetadata đạt
- [ ] 🟢 TC-023 [AC7] — NDJSON success phát đúng chuỗi Start → Delta → Stop kèm usage. — AI: TestCommandCodeStreamSuccess đạt
- [ ] 🟢 TC-024 [AC7] — Event `error` trước/sau delta → đúng một `StreamError`, không có stop/[DONE] sau lỗi. — AI: TestCommandCodeStreamErrorEvent đạt
- [ ] 🟢 TC-025 [AC7] — Known event sai shape, dòng > 1 MiB, read error, EOF thiếu terminal → một `StreamError` rồi đóng channel. — AI: TestStream EOF/LineTooLong/WrongShape đạt
- [ ] 🟢 TC-026 [AC7] — Dòng rỗng, JSON rác, unknown event type → bỏ qua. — AI: TestCommandCodeStreamIgnoresGarbageAndUnknown đạt
- [ ] 🟢 TC-027 [AC8] — Non-stream chỉ trả `ChatResponse` khi nhận success terminal, có content/stop reason/usage. — AI: TestCommandCodeChatCompletion đạt
- [ ] 🟢 TC-028 [AC8] — Non-stream error hoặc EOF thiếu terminal → trả error, không trả response dở dang. — AI: TestCommandCodeChatCompletionErrorDiscardsPartial đạt
- [ ] 🟢 TC-029 [AC9] — Upstream 401/403 → `AuthFailed=true, Retryable=false`; 429/503 → `Retryable=true`. — AI: TestCommandCodeUpstreamErrorMapping đạt
- [ ] 🟢 TC-030 [AC9] — Generic 400 → `invalid_request`; unknown-model 400 → `unsupported_model`. — AI: error mapping + integration test đạt
- [ ] 🟢 TC-031 [AC9] — Upstream echo apiKey/Bearer → không lộ credential trong error/response/log. — AI: redaction tests đạt
- [ ] 🟢 TC-032 [AC10] — Integration test + chạy end-to-end qua alias command_code (stream/non-stream, cả 2 giao thức) đúng protocol. — AI: integration test + runtime 200 đạt sau fix UUID
- [ ] 🟢 TC-033 [AC10] — Alias lạ → HTTP 404 `model_not_found`. — AI: integration test + runtime 404 đạt
- [ ] 🟢 TC-034 [AC10] — Unknown-model 400 → `unsupported_model`; generic 400 → `invalid_request`. — AI: integration test đạt
- [ ] 🟢 TC-035 [AC10] — In-stream error là terminal, không kèm success terminal. — AI: TestCCIntegrationStreamErrorTerminal đạt

## Lỗ hổng đặc tả / Sai lệch
- Đã giải quyết: `threadId`/`x-session-id` từng được sinh bằng `randomHex(16)` (32 hex không gạch nối) và bị upstream từ chối với 400 "Invalid UUID at threadId". Đã sửa bằng `randomUUID()` (RFC 4122 v4) và xác minh lại qua runtime.

## Xác nhận của người kiểm tra
- [ ] Đã hoàn thành các ca kiểm thử còn ô checkbox.
- [ ] Đã chấp nhận các ca kiểm thử màu xanh và không cần kiểm tra lại.
- [ ] Tính năng đạt yêu cầu hoặc đã ghi rõ các ca kiểm thử chưa đạt.

## Nguồn
- Đặc tả đã duyệt: docs/ai/specs/command-code-provider.md
- Bằng chứng xác minh: docs/ai/verifications/command-code-provider.md
