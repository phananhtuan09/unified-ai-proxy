# Spec: Command Code Provider

## Cấp Độ
Extended

## Hợp Đồng Thực Thi

### Mục Tiêu
- Thêm provider `command_code` vào unified-ai-proxy: cấu hình, đăng nhập qua browser, gọi chat completion (stream + non-stream) qua endpoint `/alpha/generate` của Command Code.
- Client của proxy gọi chuẩn OpenAI/Anthropic hiện có với model alias của command_code mà không cần biết giao thức upstream.

### Nguồn Quyết Định Đã Duyệt
- Manifest quyết định: `docs/ai/design-decisions/command-code-provider.json` — revision 1, `approval_meaning: direction-approved` (sha256 `4281be…`).
- Bản xem trước đầu ra (kind `api`): client POST `/v1/chat/completions` với model alias command_code, nhận SSE `chat.completion.chunk` chuẩn OpenAI; các trạng thái quan sát được: Chưa đăng nhập / Đã đăng nhập / Lỗi upstream; các nhánh flow đã duyệt:
  - Login: timeout hoặc huỷ → lỗi, không ghi token; callback lỗi → in lỗi, không ghi token; apiKey rỗng/sai định dạng → báo lỗi đăng nhập thất bại.
  - Chat: alias chưa cấu hình → lỗi 400 model_not_found; upstream 401 → đánh dấu auth failed, không retry mù quáng; upstream 5xx/429 → failover theo `routing.failover`.
- D-001: Cách đăng nhập → browser flow `https://commandcode.ai/studio/auth/cli?callback=<localhost>&state=<state>`, callback trả apiKey `user_...`, lưu vào token file.
- D-002: Endpoint → `POST https://api.commandcode.ai/alpha/generate` với apiKey của phiên đăng nhập, kèm headers `x-session-id`, `x-command-code-version`, `x-cli-environment`.
- D-003: Model alias mặc định → DeepSeek V4 Pro/Flash, Qwen 3.6 Max Preview, GLM-5, MiniMax M2.7 (theo registry 9router).
- BR-001: Credential là apiKey `user_...` thay OAuth token; lưu qua tokenstore cùng schema `TokenSet` (`AccessToken` = apiKey, không có refresh_token).
- BR-002: Mọi request upstream phải `stream=true`; request non-stream của client được proxy gom từ NDJSON rồi trả response hoàn chỉnh.
- BR-003: Mỗi request upstream gắn một `x-session-id` mới (UUID ngẫu nhiên).
- BR-004: Model upstream giữ nguyên tên đầy đủ trong `models[].upstream`; alias do người dùng đặt trong `models[].id`.
- IC-001: Đăng nhập bằng browser cho command_code là yêu cầu của người dùng (`human-request`).
- IC-002: Provider tên cấu hình `command_code`, đăng ký qua `provider.Build` và `providerDefaults`.
- IC-003: Tái sử dụng `base.doJSON`, tokenstore, `UpstreamError`, cơ chế failover hiện có; không thêm framework mới.
- IC-004: Lệnh `auth command_code` đi qua nhánh CLI auth hiện có.
- IC-005: Không log apiKey, không ghi key vào lỗi trả về client.
- Ràng buộc human thêm khi duyệt: Không có.

### Bắt Buộc Xảy Ra
- `unified-ai-proxy auth command_code --account <name>` mở browser tới trang đăng nhập Command Code, chờ callback tại localhost, lưu apiKey hợp lệ vào token file 0600.
- Request với model alias command_code được dịch sang body `/alpha/generate` và chạy end-to-end cả stream lẫn non-stream.
- Tài khoản command_code xuất hiện trong `unified-ai-proxy accounts`.

### Không Được Xảy Ra
- Không gọi refresh token cho command_code (không có refresh_token).
- Không retry khi upstream trả 401/403 (auth failure).
- Không để apiKey xuất hiện trong log, thông báo lỗi, hoặc response gửi về client.
- Không hỗ trợ tool calling / image / reasoning content cho command_code trong slice này.

## Vấn Đề
Proxy hiện chỉ hỗ trợ `openai_codex` và `gemini`. Tài khoản Command Code (coding agent CLI) không dùng được qua proxy dù người dùng đã trả phí; cần thêm provider để gộp Command Code vào pool routing và dùng credential đăng nhập sẵn có (api key `user_...`).

## Phạm Vi
- Provider `command_code` trong config: defaults, validate, example accounts.
- Đăng nhập browser qua lệnh CLI auth hiện có (nhánh mới cho auth method `browser_key`).
- Provider client `internal/provider/commandcode.go`: chat + stream, dịch NDJSON → `StreamEvent`.
- Tests cho request translation, NDJSON parsing, config defaults/validation.

## Ngoài Phạm Vi
- Tool calling, image parts, reasoning content (chỉ text thuần).
- Anthropic-native endpoint `/v1/messages` của Command Code.
- Tự động đồng bộ token từ `~/.commandcode/auth.json` của cmd CLI.
- Provider API chính thức (`/provider/v1/chat/completions`) — đã bị loại ở D-002.

## Quyết Định Thiết Kế Đã Duyệt
- D-001 (human): Đăng nhập bằng browser flow studio/auth/cli; lý do: trải nghiệm một bấm giống `cmd login`, chấp nhận phụ thuộc endpoint chưa tài liệu chính thức.
- D-002 (human): Dùng `/alpha/generate` với credential phiên đăng nhập; lý do: dùng được ngay với apiKey nhận từ login, đúng cách 9router đã chạy.
- D-003 (human): 5 model alias mặc định (DeepSeek V4 Pro, DeepSeek V4 Flash, Qwen 3.6 Max Preview, GLM-5, MiniMax M2.7); lý do: đủ lựa chọn thực tế, ít nguy cơ trùng alias.

## Kiểm Tra Giả Định

### Từ Design Đã Duyệt
- A-001: Schema request `/alpha/generate` (threadId, config, params với `system` string + `messages` content blocks) theo mã nguồn 9router (`open-sse/translator/request/openai-to-commandcode.js`) — nếu sai: upstream từ chối request, phải sửa translator.
- A-002: Trang `studio/auth/cli` trả apiKey về listener localhost qua callback — nếu sai: đăng nhập browser thất bại, phải chuyển sang dán API key thủ công.
- A-003: apiKey `user_...` dùng dài hạn không cần refresh — nếu sai: người dùng phải đăng nhập lại khi key hết hạn.
- R-001: `/alpha/generate` không tài liệu chính thức, có thể đổi hoặc bị khoá — hệ quả: provider ngừng hoạt động đột ngột.
- R-002: Dùng credential phiên CLI cho traffic proxy có thể vi phạm điều khoản Command Code — hệ quả: tài khoản người dùng có thể bị giới hạn/khoá.

### Đã Xác Nhận
- `internal/provider/factory.go:12`: `Build()` switch theo tên, chỉ có `openai_codex`, `gemini`.
- `internal/config/config.go:195`: `providerDefaults` là nơi đặt hằng số auth mặc định theo tên provider.
- `internal/provider/oauth.go:37`: `RunOAuthLogin` chỉ làm authorization-code + PKCE; không phù hợp flow trả apiKey.
- `internal/cli/cli.go:171`: `cmdAuth` gọi cứng `RunOAuthLogin`.
- `internal/provider/base.go:44`: `accessToken` giả định OAuth token có thể refresh.
- `internal/provider/base.go:140`: `buildRefreshRequest` giả định có refresh_token.
- `internal/provider/sse.go:16`: chỉ có parser SSE chuẩn, chưa có NDJSON reader.
- `internal/model/model.go`: `Role` hiện chỉ có system/user/assistant/developer (text thuần).
- Máy dev có `~/.commandcode/auth.json` chứa `apiKey` dạng `user_...` — cùng định dạng credential với flow browser (bundle cmd CLI: `buildManualCommandAuthConfig`/`buildBrowserCommandAuthConfig` cùng shape `{apiKey, userId, userName, keyName}`).

### Suy Luận An Toàn
- NDJSON trả về là AI SDK v5 events (`text-delta`, `finish-step`, `error`, ...) theo translator 9router `commandcode-to-openai.js`; parser sẽ bỏ qua event không nhận diện.
- Callback của studio/auth/cli mang apiKey qua query param hoặc body; handler sẽ kiểm tra cả hai nơi.

### Cần Xác Nhận
- Không có câu hỏi sản phẩm nào đang blocking.

### Chi Tiết Kỹ Thuật Do Agent Chọn
- (ai_discretion) Cách parse NDJSON theo dòng dùng `bufio.Scanner` và ánh xạ event AI SDK v5 → `StreamEvent` (chỉ `start`, `text-delta`, `finish-step`, `finish`, `error`).
- (ai_discretion) `max_tokens` mặc định khi client không gửi: dùng `cfg.Server.DefaultMaxTokens` đã truyền qua request.
- (ai_discretion) Nội dung tests và cấu trúc file test.
- (ai_discretion) Trạng thái `accounts` khi chưa đăng nhập: hiện `missing`/`reauth_required` theo logic `tokenstore.Load` hiện có — không thêm trạng thái mới.
- Agent chọn (không phải human): tên auth method mới trong config là `browser_key` để phân biệt với `oauth` và `api_key`; dùng chuẩn auth method để CLI phân nhánh. Agent chọn: `x-command-code-version` mặc định `0.25.7` (phiên bản 9router đã verify), overridable qua config nếu cần sau này.

## Bằng Chứng Hệ Thống Hiện Tại
- `internal/provider/provider.go:13`: interface `Provider` gồm `Name/Models/ValidateAccount/ChatCompletion/StreamChatCompletion/RefreshToken`.
- `internal/provider/base.go:27`: `newBase` dùng cho provider OAuth; `doJSON:180` dùng được chung.
- `internal/provider/codex.go:17`: mẫu provider con hoàn chỉnh (buildRequest, headers, ChatCompletion, StreamChatCompletion, upstreamError).
- `internal/config/config.go:295`: `validateProvider` yêu cầu OAuth đủ client_id/authorization_url/token_url/redirect_port khi `auth.method=oauth` — cần nhánh riêng cho `browser_key`.
- `internal/proxy/service.go:29`: `New()` build provider từ config và map alias → upstream; failover dựa trên `IsRetryable`/`IsAuthFailure` của `UpstreamError`.
- `internal/server/openai.go`, `internal/server/anthropic.go`: chuyển `StreamEvent` sang SSE chuẩn — không cần sửa nếu provider phát đúng event types.

## Tiền Lệ Trong Codebase
- Theo mẫu: `internal/provider/codex.go` (Codex) — provider OAuth con gần nhất về hình dạng: embed `base`, build request struct riêng, parse response riêng, `upstreamError` riêng.
- Phải giống: cấu trúc file (`NewCommandCode(cfg, timeout)` trả `*CommandCode`), đăng ký trong `factory.go`, trả `UpstreamError` có `Retryable`/`AuthFailed` đúng để failover hoạt động, tests đặt cùng package (`commandcode_test.go` cạnh `gemini_test.go`).
- Cố ý khác: không dùng `base.accessToken` (OAuth refresh) mà đọc apiKey trực tiếp từ `tokenstore.Load`; thêm login flow mới song song với `RunOAuthLogin` vì callback trả apiKey thay vì authorization code.

## Yêu Cầu Hành Vi

### Cấu Hình (AC group: config)
- `providerDefaults["command_code"]` cung cấp: `auth.method=browser_key`, `authorization_url=https://commandcode.ai/studio/auth/cli`, `api.base_url=https://api.commandcode.ai`, `redirect_host=localhost`, `redirect_port=1458` (mặc định, overridable), `redirect_path=/callback`.
- Config example thêm provider `command_code` với 5 model alias theo D-003 (upstream giữ tên đầy đủ BR-004) và 1 account dùng token file.
- `Validate()` chấp nhận `auth.method=browser_key`: yêu cầu `authorization_url`, `redirect_port > 0`, account có `token_file`; không yêu cầu client_id/token_url.

### Đăng Nhập (AC group: login)
- `auth command_code --account <name>`: mở browser tới `<authorization_url>?callback=http://<redirect_host>:<redirect_port><redirect_path>&state=<32-byte hex>`; chờ callback với timeout 5 phút như OAuth hiện hành.
- Callback hợp lệ khi mang apiKey không rỗng bắt đầu bằng `user_` (kiểm tra ở query param rồi tới body); lưu `TokenSet{AccessToken: apiKey, TokenType: "Bearer", ExpiresAt: zero}` qua `tokenstore.Save`.
- Timeout, huỷ, callback lỗi, hoặc apiKey sai định dạng → in lỗi, không ghi file token.
- Server callback bind `127.0.0.1` như `RunOAuthLogin`.

### Provider Client (AC group: chat)
- Request service xây cho command_code: `{threadId, memory:"", config:{...}, params:{model, messages, system, stream:true, max_tokens, temperature?, top_p?}}` theo schema 9router; `messages` role ∈ {user, assistant}, content là array `[{type:"text", text}]`; `system` đặt ở `params.system` nếu có, không đưa vào `messages`.
- Headers: `Authorization: Bearer <apiKey>`, `x-session-id: <uuid>`, `x-command-code-version: 0.25.7`, `x-cli-environment: cli`, `Accept: text/event-stream`.
- Stream: đọc NDJSON từng dòng; `text-delta` → `StreamContentDelta`; dòng đầu hợp lệ → phát `StreamMessageStart` (id `chatcmpl-<hex>` ngẫu nhiên, model = upstream model); `finish-step`/`finish` → `StreamMessageStop` với `StopReason` map (`stop` → `end_turn`, `length` → `max_tokens`, giữ nguyên nếu khác) và `Usage` (input_tokens/output_tokens); `error` → `StreamError`.
- Non-stream: gọi `StreamChatCompletion` nội bộ, gom `Content`, lấy usage từ event stop, trả `ChatResponse`.
- Non-200 → `UpstreamError` theo bảng: 401/403 AuthFailed, 429/5xx Retryable; body JSON `{error:{message}}` được trích message.

### Routing & Failover
- `UpstreamError` của command_code phải tương thích `IsRetryable`/`IsAuthFailure` để failover trong `internal/proxy/service.go` hoạt động không sửa đổi (IC-003).

## Thay Đổi Trạng Thái / Dữ Liệu / Giao Diện
- Token file mới cho account command_code: `TokenSet` với `AccessToken` là apiKey, `ExpiresAt` zero, không `RefreshToken`. Không thay đổi schema tokenstore.
- Config thêm provider `command_code` và accounts tương ứng.
- CLI: lệnh `auth` giữ nguyên cú pháp; thêm phân nhánh theo auth method.
- Bề mặt API của proxy không đổi (thêm alias mới qua config).

## Thiết Kế Kỹ Thuật Chi Tiết

### `internal/provider/commandcode.go`
- Trách nhiệm: provider `command_code` — dịch `ChatRequest` → body `/alpha/generate`, gửi request, dịch NDJSON → `StreamEvent`, dựng `UpstreamError`.
- Đầu vào: `config.ProviderConfig`, `model.Account` (TokenFile), `model.ChatRequest`.
- Đầu ra: `ChatResponse` hoặc channel `StreamEvent`.
- Đọc credential: load `tokenstore.Load(account.TokenFile)`; nếu nil hoặc `AccessToken == ""` → lỗi auth (dạng lỗi khiến account bị đánh dấu reauth theo flow hiện hành của proxy — dùng cùng kiểu lỗi `base` trả khi thiếu token).
- Override `RefreshToken`: trả token hiện tại (apiKey không refresh được) hoặc lỗi nếu thiếu file — để thoả interface `Provider` mà không gọi upstream.
- Override `ValidateAccount`: chỉ cần token file tồn tại và apiKey không rỗng (không gọi upstream khi chưa có nhu cầu), khác `base.ValidateAccount` vốn refresh.
- Hành vi lỗi: non-200 → `UpstreamError`; lỗi mạng → `networkError`.

### `internal/provider/commandauth.go` (hoặc ghép vào oauth.go)
- Trách nhiệm: `RunBrowserKeyLogin(ctx, provider, account, tokenFile, cfg)` cho auth method `browser_key`.
- Luồng: sinh `state` (dùng lại `randomHex`), bind callback server 127.0.0.1:redirect_port, dựng URL `<authorization_url>?callback=<redirectURI>&state=<state>` (không dùng các query OAuth chuẩn vì D-001 là flow riêng của studio), mở browser (`openBrowser` dùng lại), chờ callback/timeout/huỷ.
- Callback handler: đọc apiKey từ query (`api_key`, `apiKey`, `key` — kiểm tra lần lượt) rồi tới form/JSON body; validate prefix `user_`; đáp HTML xác nhận; gửi qua channel như `oauthSession`.
- Không yêu cầu `state` khớp trong callback vì studio/auth/cli chỉ nhận state để định tuyến lại; nếu callback không kèm state vẫn chấp nhận nhưng log cảnh báo (agent-chosen).

### `internal/config/config.go`
- Thêm `providerDefaults["command_code"]`; mở rộng `validateProvider` với case `browser_key` (yêu cầu `authorization_url`, `redirect_port`, account token_file).
- Agent chọn: hằng số method `browser_key` thêm vào danh sách chấp nhận của `normalizeMethod` (giữ nguyên hành vi mặc định `oauth` khi rỗng).

### `internal/cli/cli.go`
- `cmdAuth`: phân nhánh theo `pc.Auth.Method`: `oauth` → `RunOAuthLogin` (hiện hành), `browser_key` → login mới; method khác → lỗi rõ ràng. `normalizeProviderName` không cần đổi (tên `command_code` đã hợp lệ).

## Bản Đồ Thay Đổi Theo File
| Bề mặt | Thay đổi dự kiến | Lý do | Quyết định / AC |
|---|---|---|---|
| `internal/provider/commandcode.go` (mới) | Provider client: buildRequest, ChatCompletion, StreamChatCompletion, NDJSON parse, upstreamError, RefreshToken/ValidateAccount override | Gọi /alpha/generate và dịch NDJSON | D-002, BR-002, BR-003 / AC5-AC9 |
| `internal/provider/oauth.go` hoặc `commandauth.go` (mới) | Login browser cho method `browser_key` | Flow studio/auth/cli trả apiKey, không phải OAuth code | D-001, IC-001 / AC3, AC4 |
| `internal/provider/factory.go` | Thêm case `command_code` | Đăng ký provider | IC-002 / AC5 |
| `internal/config/config.go` | providerDefaults + validate `browser_key` | Config chạy được tối giản và validate đúng | BR-004, D-003 / AC1, AC2 |
| `internal/cli/cli.go` | Phân nhánh auth theo method | Lệnh auth dùng được cho command_code | IC-004 / AC3 |
| `internal/provider/commandcode_test.go` (mới) | Test buildRequest, NDJSON parse, error mapping | Yêu cầu tests của scope | / AC6-AC9 |
| `internal/config/config_test.go` | Test defaults/validate command_code | Bảo vệ hành vi config | / AC1, AC2 |
| `config.example.yaml` (mới — repo chưa có config mẫu, `config.yaml` hiện là config riêng của người dùng) | Thêm provider command_code | Người dùng cấu hình được ngay | D-003 / AC2 |

## Kiểm Tra Hợp Lệ / Lỗi / Trường Hợp Biên
- Alias chưa cấu hình → `apierr.ModelNotFound` (400) như hiện tại.
- Token file thiếu/rỗng khi gọi chat → lỗi auth; account bị đánh dấu reauth theo cơ chế sẵn có.
- Upstream trả 401 giữa stream → phát `StreamError` với message không chứa apiKey.
- NDJSON dòng rỗng, dòng không parse được, event lạ → bỏ qua, không làm sập stream.
- Callback nhận apiKey không bắt đầu `user_` → từ chối, báo lỗi.
- Port callback bị chiếm → lỗi khởi động login rõ ràng (hành vi server bind hiện hành).

## Cân Nhắc Bảo Mật / Phân Quyền
- apiKey lưu qua `tokenstore.Save` (0600) như token OAuth; IC-005: không log, không đưa vào error message.
- Callback server bind `127.0.0.1` như `RunOAuthLogin` (internal/provider/oauth.go:67).
- Response login HTML trả về browser không chứa apiKey.
- R-002 được ghi nhận: người dùng tự chịu trách nhiệm về điều khoản Command Code.

## Tương Thích / Di Chuyển Dữ Liệu
- Không có migration: provider mới thêm, config cũ không đổi hành vi. `Validate()` cũ không biết `browser_key` — chỉ ảnh hưởng khi người dùng thêm provider mới.
- Endpoint `/v1/chat/completions` và `/v1/messages` của proxy giữ nguyên; thêm alias không phá vỡ client cũ.

## Trình Tự Triển Khai
1. Config: thêm `providerDefaults["command_code"]`, validate `browser_key`, tests config.
2. Provider client: `commandcode.go` + tests buildRequest/NDJSON (httptest server giả lập NDJSON).
3. Factory: đăng ký command_code.
4. Login: hàm login browser mới + phân nhánh `cmdAuth`; test handler callback bằng httptest.
5. Cập nhật config example + chạy `go test ./...` và `go vet ./...`.

## Tiêu Chí Chấp Nhận

### Cấu Hình
- [ ] AC1: Config tối giản chỉ khai `enabled: true` + account command_code vẫn pass `Validate()` nhờ defaults; thiếu `authorization_url`/`redirect_port`/token file → lỗi validate nêu rõ tên provider.
- [ ] AC2: Config example chứa đủ 5 model alias theo D-003 với upstream đúng (`deepseek/deepseek-v4-pro`, `deepseek/deepseek-v4-flash`, `Qwen/Qwen3.6-Max-Preview`, `zai-org/GLM-5`, `MiniMaxAI/MiniMax-M2.7`) và alias không trùng provider khác.

### Đăng Nhập
- [ ] AC3: `auth command_code --account <name>` sinh URL theo dạng `https://commandcode.ai/studio/auth/cli?callback=http://localhost:<port>/callback&state=<hex>` và mở browser (hoặc in URL nếu mở thất bại).
- [ ] AC4: Callback mang apiKey hợp lệ `user_...` → token file được ghi với `AccessToken` là apiKey, `tokenstore.Load` đọc lại được; callback thiếu/sai apiKey, timeout, hoặc huỷ → không ghi file, lệnh trả lỗi.

### Provider Client
- [ ] AC5: `provider.Build("command_code", ...)` trả provider implement đủ interface; `Models()` trả alias theo config (BR-004).
- [ ] AC6: Body gửi upstream khớp schema: `params.stream=true` luôn đúng (kể cả khi client non-stream), `params.system` nhận nội dung system, `messages` content dạng array text blocks; có đủ 4 headers theo D-002/BR-003 (kiểm tra qua httptest).
- [ ] AC7: Stream NDJSON gồm `text-delta` và `finish-step` được phát thành `StreamMessageStart` → các `StreamContentDelta` (ghép đúng thứ tự) → `StreamMessageStop` kèm usage; dòng rác/event lạ bị bỏ qua.
- [ ] AC8: Non-stream trả `ChatResponse` với content ghép đầy đủ và usage từ event finish.
- [ ] AC9: Upstream 401 → `UpstreamError` có `AuthFailed=true, Retryable=false`; 429/503 → `Retryable=true`; message lỗi trích từ body `{error:{message}}` nếu có.

### Tích Hợp
- [ ] AC10: Với config chứa command_code, `go run . start` + request tới alias qua `/v1/chat/completions` trả SSE chuẩn OpenAI chống lại upstream giả lập (httptest) end-to-end; alias lạ trả 400 model_not_found.

## Ma Trận Xác Minh
| AC | Chiến lược bằng chứng | Bề mặt chính |
|---|---|---|
| AC1 | Test tập trung | `internal/config` |
| AC2 | Test tập trung + xem config example | `internal/config` |
| AC3 | Test tập trung (URL dựng) + thủ công 1 lần với browser thật | `internal/provider`, `internal/cli` |
| AC4 | Test httptest callback | `internal/provider` |
| AC5 | Test tập trung | `internal/provider/factory.go` |
| AC6 | Test httptest kiểm tra body/header | `internal/provider/commandcode.go` |
| AC7 | Test tập trung NDJSON | `internal/provider/commandcode.go` |
| AC8 | Test tập trung | `internal/provider/commandcode.go` |
| AC9 | Test tập trung | `internal/provider/commandcode.go` |
| AC10 | Runtime thủ công với stub server | `go run . start` |

## Câu Hỏi Mở
- Không có câu hỏi sản phẩm nào đang blocking.

## Nhật Ký Quyết Định
- 2026-08-17: Duyệt thiết kế revision 1 qua local-runner; các quyết định D-001 (browser studio flow), D-002 (/alpha/generate), D-003 (5 model alias) do human chọn; phần còn lại `agent-proposed-not-objected`.
- Agent chọn (chưa qua human): auth method name `browser_key`, `redirect_port` mặc định 1458, giá trị `x-command-code-version: 0.25.7`, cách tìm apiKey trong callback (query rồi tới body), ValidateAccount không gọi upstream.
