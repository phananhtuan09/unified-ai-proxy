# Spec: Command Code Provider

## Cấp Độ
Extended

## Hợp Đồng Thực Thi

### Mục Tiêu
- Thêm provider `command_code` vào unified-ai-proxy: cấu hình, đăng nhập qua browser, gọi chat completion (stream + non-stream) qua endpoint `/alpha/generate` của Command Code.
- Client của proxy gọi các endpoint OpenAI-compatible và Anthropic-compatible hiện có với model alias của command_code mà không cần biết giao thức upstream.

### Nguồn Quyết Định Đã Duyệt
- Manifest quyết định: `docs/ai/design-decisions/command-code-provider.json` — revision 1, `approval_meaning: direction-approved` (sha256 `4281be…`).
- Bản xem trước đầu ra (kind `api`): client POST `/v1/chat/completions` với model alias command_code, nhận SSE `chat.completion.chunk` chuẩn OpenAI; các trạng thái quan sát được: Chưa đăng nhập / Đã đăng nhập / Lỗi upstream; các nhánh flow đã duyệt:
  - Login: timeout hoặc huỷ → lỗi, không ghi token; callback thiếu/sai `state`, callback lỗi, hoặc apiKey rỗng/sai định dạng → báo lỗi đăng nhập thất bại và không ghi token.
  - Chat: alias chưa cấu hình → lỗi model_not_found như hiện tại (HTTP 404, internal/apierr/apierr.go:32; manifest ghi "400 như hiện tại" nhưng hành vi hiện tại là 404 nên giữ 404); upstream 401 → đánh dấu auth failed, không retry mù quáng; upstream 5xx/429 → failover theo `routing.failover`.
- Lỗi client theo preview đã duyệt: 401 khi chưa đăng nhập hoặc apiKey hết hạn/bị thu hồi (proxy trả `provider_auth_failed`), 400 `unsupported_model` khi model upstream không có trong catalog Command Code, 429 khi upstream giới hạn tần suất.
- D-001: Cách đăng nhập → browser flow `https://commandcode.ai/studio/auth/cli?callback=<localhost>&state=<state>`, callback trả apiKey `user_...`, lưu vào token file.
- D-002: Endpoint → `POST https://api.commandcode.ai/alpha/generate` với apiKey của phiên đăng nhập, kèm headers `x-session-id`, `x-command-code-version`, `x-cli-environment`.
- D-003: Model alias mặc định → DeepSeek V4 Pro/Flash, Qwen 3.6 Max Preview, GLM-5, MiniMax M2.7 (theo registry 9router).
- BR-001: Credential là apiKey `user_...` thay OAuth token; lưu qua tokenstore cùng schema `TokenSet` (`AccessToken` = apiKey, không có refresh_token).
- BR-002: Mọi request upstream phải `stream=true`; request non-stream của client được proxy gom từ NDJSON rồi trả response hoàn chỉnh.
- BR-003: Mỗi request upstream gắn một `x-session-id` mới (UUID ngẫu nhiên).
- BR-004: Model upstream giữ nguyên tên đầy đủ trong `models[].upstream`; alias do người dùng đặt trong `models[].id`.
- IC-001: Đăng nhập bằng browser cho command_code là yêu cầu của người dùng (`human-request`).
- IC-002: Provider tên cấu hình `command_code`, đăng ký qua `provider.Build` và `providerDefaults`.
- IC-003: Tái sử dụng tokenstore, `UpstreamError`, `networkError`, cơ chế failover hiện có; không thêm framework mới. Ghi chú: `base.doJSON` (internal/provider/base.go:180) đọc toàn bộ body nên không dùng được cho đường streaming; request được dựng thủ công theo mẫu `Codex.StreamChatCompletion` (internal/provider/codex.go:167-188).
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
- Không chấp nhận callback login thiếu hoặc sai `state`.
- Không phát completion thành công hoặc `[DONE]` sau khi stream đã phát lỗi.
- Không hỗ trợ tool calling / image / reasoning content cho command_code trong slice này.

## Vấn Đề
Proxy hiện chỉ hỗ trợ `openai_codex` và `gemini`. Tài khoản Command Code (coding agent CLI) không dùng được qua proxy dù người dùng đã trả phí; cần thêm provider để gộp Command Code vào pool routing và dùng credential đăng nhập sẵn có (api key `user_...`).

## Phạm Vi
- Provider `command_code` trong config: defaults, validate, example accounts.
- Đăng nhập browser qua lệnh CLI auth hiện có (nhánh mới cho auth method `browser_key`).
- Provider client `internal/provider/commandcode.go`: chat + stream, dịch NDJSON → `StreamEvent`.
- Tests cho request translation, NDJSON parsing, config defaults/validation, login callback, redaction, account status và hai giao thức downstream OpenAI/Anthropic.

## Ngoài Phạm Vi
- Tool calling, image parts, reasoning content (chỉ text thuần).
- Anthropic-native endpoint upstream của Command Code; endpoint Anthropic-compatible `/v1/messages` hiện có của proxy vẫn thuộc phạm vi tương thích.
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
- NDJSON trả về là AI SDK v5 events (`text-delta`, `finish-step`, `finish`, `error`, ...) theo translator 9router `commandcode-to-openai.js`; parser sẽ bỏ qua event không nhận diện nhưng không được bỏ qua lỗi đọc hoặc lỗi vượt giới hạn dòng.
- Callback của studio/auth/cli mang apiKey và `state` qua query hoặc body. Implementation chỉ hỗ trợ các tên field và event shape có fixture trích từ bundle Command Code/9router được lưu trong testdata; không suy đoán thêm alias field khi chưa có bằng chứng.

### Cần Xác Nhận
- Không có câu hỏi sản phẩm nào đang blocking.
- Trước khi viết translator và callback parser, executor phải trích và lưu fixture đã lược bỏ credential từ nguồn được nêu ở A-001/A-002 để xác nhận: tên field apiKey, vị trí `state`, body request đầy đủ, và shape của `text-delta`/`finish-step`/`finish`/`error`/usage. Nếu fixture mâu thuẫn với A-001 hoặc A-002, dừng implementation và cập nhật spec thay vì đoán schema.

### Chi Tiết Kỹ Thuật Do Agent Chọn
- (ai_discretion) Cách parse NDJSON theo dòng dùng `bufio.Scanner`, tăng buffer từ mặc định và đặt giới hạn tối đa 1 MiB cho mỗi dòng; kiểm tra `Scanner.Err()`; ánh xạ event AI SDK v5 → `StreamEvent` (chỉ `start`, `text-delta`, `finish-step`, `finish`, `error`).
- (ai_discretion) `max_tokens` mặc định khi client không gửi: dùng `cfg.Server.DefaultMaxTokens` đã truyền qua request.
- (ai_discretion) Nội dung tests và cấu trúc file test.
- (ai_discretion) Trạng thái `accounts` khi token file thiếu, lỗi parse hoặc có `AccessToken` rỗng là `reauth_required`; token hợp lệ có expiry bằng zero hiển thị `ok`/`never`. Cần mở rộng `accounts.Summarize` vì code hiện tại chỉ đặt `Expiry=missing` nhưng vẫn trả `Status=ok`.
- Agent chọn (không phải human): tên auth method mới trong config là `browser_key` để phân biệt với `oauth` và `api_key`; dùng chuẩn auth method để CLI phân nhánh. Agent chọn: `x-command-code-version` mặc định `0.25.7` (phiên bản 9router đã verify), overridable qua config nếu cần sau này.

## Bằng Chứng Hệ Thống Hiện Tại
- `internal/provider/provider.go:13`: interface `Provider` gồm `Name/Models/ValidateAccount/ChatCompletion/StreamChatCompletion/RefreshToken`.
- `internal/provider/base.go:27`: `newBase` dùng cho provider OAuth; `doJSON:180` dùng được chung.
- `internal/provider/codex.go:17`: mẫu provider con hoàn chỉnh (buildRequest, headers, ChatCompletion, StreamChatCompletion, upstreamError).
- `internal/config/config.go:295`: `validateProvider` yêu cầu OAuth đủ client_id/authorization_url/token_url/redirect_port khi `auth.method=oauth` — cần nhánh riêng cho `browser_key`.
- `internal/proxy/service.go:29`: `New()` build provider từ config và map alias → upstream; failover dựa trên `IsRetryable`/`IsAuthFailure` của `UpstreamError`.
- `internal/server/openai.go`, `internal/server/anthropic.go`: chuyển `StreamEvent` sang SSE chuẩn; success path được tái sử dụng, nhưng error path cần sửa để lỗi là terminal.
- `internal/server/openai.go:268-281`: writer hiện phát error rồi vẫn có thể tự phát finish và `[DONE]`; phải sửa để `StreamError` là terminal và không phát completion thành công sau lỗi.
- `internal/server/anthropic.go:277-284`: writer hiện phát error nhưng không dừng đọc channel; phải sửa để error là terminal.
- `internal/accounts/summary.go:41-65`: token file thiếu chỉ làm `Expiry=missing`, còn `Status` mặc định vẫn là `ok`; cần sửa để trạng thái “Chưa đăng nhập” trong preview quan sát được.

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
- Callback chỉ hợp lệ khi mang `state` khớp giá trị đã sinh và apiKey không rỗng bắt đầu bằng `user_`; so sánh `state` bằng `crypto/subtle.ConstantTimeCompare`. Đọc `state` và apiKey từ query hoặc body theo đúng tên field đã xác nhận trong fixture A-002, rồi lưu `TokenSet{AccessToken: apiKey, TokenType: "Bearer", ExpiresAt: zero}` qua `tokenstore.Save`.
- Callback thiếu/sai `state`, timeout, huỷ, callback lỗi, hoặc apiKey sai định dạng → trả lỗi, không ghi file token. Callback hợp lệ thứ hai sau khi phiên đã hoàn tất không được ghi đè token hoặc block handler.
- Server callback bind `127.0.0.1` như `RunOAuthLogin`.

### Provider Client (AC group: chat)
- Request service xây cho command_code: `{threadId, memory:"", config:{...}, params:{model, messages, system, stream:true, max_tokens, temperature?, top_p?}}` theo fixture A-001; `messages` role ∈ {user, assistant}, content là array `[{type:"text", text}]`; `system` và developer message được ghép theo thứ tự vào `params.system`, không đưa vào `messages`.
- `threadId` là UUID mới mỗi request và dùng cùng giá trị với header `x-session-id`; `config` là object rỗng `{}` nếu fixture A-001 xác nhận. Không gửi `StopSequences` hoặc `Metadata` lên Command Code trong slice này vì schema upstream chưa xác nhận; proxy chấp nhận các field này theo contract downstream hiện có rồi bỏ qua nhất quán cho provider này, giống các provider không hỗ trợ tương ứng, và phải có test bảo vệ hành vi.
- Headers: `Authorization: Bearer <apiKey>`, `x-session-id: <uuid>`, `x-command-code-version: 0.25.7`, `x-cli-environment: cli`, `Accept: text/event-stream`.
- Stream: đọc NDJSON từng dòng với giới hạn 1 MiB; `text-delta` → `StreamContentDelta`; event đầu tiên được nhận diện → phát `StreamMessageStart` (id `chatcmpl-<hex>` ngẫu nhiên, model = upstream model); `finish-step`/`finish` terminal → đúng một `StreamMessageStop` với `StopReason` map (`stop` → `end_turn`, `length` → `max_tokens`, giữ nguyên nếu khác) và `Usage` (input_tokens/output_tokens). Event `error`, JSON event đã nhận diện nhưng sai shape, `Scanner.Err()`, dòng vượt giới hạn, hoặc EOF trước terminal event → phát đúng một `StreamError` rồi đóng channel; không phát `StreamMessageStop` sau lỗi.
- Non-stream: gọi cùng parser nội bộ, gom `Content`, lấy stop reason/usage từ event stop và chỉ trả `ChatResponse` khi nhận terminal success. Gặp `StreamError` ở bất kỳ vị trí nào phải trả error và bỏ response/content dở dang.
- Lỗi trước khi provider trả channel, gồm HTTP non-200 và credential cục bộ không hợp lệ, dùng `UpstreamError` để proxy phân loại/failover. Lỗi in-band sau HTTP 200 không được failover vì downstream có thể đã nhận bytes; chỉ phát protocol error terminal.
- Non-200 → `UpstreamError` theo bảng: 401/403 `AuthFailed=true, Retryable=false`; 429/5xx `Retryable=true`; 400 `Retryable=false`. Chỉ gắn loại `unsupported_model` khi structured error code hoặc message theo fixture xác nhận model không có trong catalog; generic 400 giữ `invalid_request`.
- Mọi message upstream phải qua hàm sanitization nhận apiKey hiện tại trước khi tạo `UpstreamError` hoặc `StreamError`. Nếu message chứa apiKey hoặc bearer credential thì thay credential bằng `[REDACTED]`; không bao giờ đưa raw body chưa sanitize vào error hoặc log.

### Routing & Failover
- `UpstreamError` của command_code phải tương thích `IsRetryable`/`IsAuthFailure` để failover trong `internal/proxy/service.go` hoạt động không sửa đổi (IC-003).
- Credential cục bộ thiếu, token file lỗi parse, `AccessToken` rỗng hoặc sai prefix phải trả `UpstreamError{StatusCode:401, AuthFailed:true, Retryable:false}` để cả `Chat` và `Stream` đánh dấu account reauth và trả HTTP 401 thay vì 503.
- `UpstreamError` cần mang phân loại model không hỗ trợ độc lập với status, hoặc provider phải trả typed sentinel tương đương. `mapUpstreamError` chỉ trả `apierr.UnsupportedModel` khi phân loại này đúng; command_code 400 khác và provider khác giữ `apierr.InvalidRequest`.

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
- Đọc credential: load `tokenstore.Load(account.TokenFile)`; token file thiếu/lỗi parse, `AccessToken` rỗng hoặc không bắt đầu bằng `user_` → `UpstreamError{StatusCode:401, AuthFailed:true, Retryable:false}` với message an toàn không chứa nội dung token file.
- Override `RefreshToken`: trả token hiện tại khi hợp lệ; nếu thiếu hoặc sai định dạng, trả cùng typed auth error. Không gọi upstream và không tạo refresh flow.
- Override `ValidateAccount`: chỉ kiểm tra token file và apiKey hợp lệ, không gọi upstream; trả cùng typed auth error khi không hợp lệ.
- Hành vi lỗi: non-200 → `UpstreamError` đã sanitize và phân loại; lỗi mạng → `networkError`; lỗi NDJSON → terminal `StreamError` đã sanitize.

### `internal/provider/commandauth.go` (hoặc ghép vào oauth.go)
- Trách nhiệm: `RunBrowserKeyLogin(ctx, provider, account, tokenFile, cfg)` cho auth method `browser_key`.
- Luồng: sinh `state` (dùng lại `randomHex`), bind callback server 127.0.0.1:redirect_port, dựng URL `<authorization_url>?callback=<redirectURI>&state=<state>` (không dùng các query OAuth chuẩn vì D-001 là flow riêng của studio), mở browser (`openBrowser` dùng lại), chờ callback/timeout/huỷ.
- Callback handler: đọc `state` và apiKey từ query/form/JSON body theo đúng fixture A-002; bắt buộc so khớp `state` bằng constant-time comparison trước khi xử lý apiKey; validate prefix `user_`; đáp HTML xác nhận không chứa credential; gửi kết quả qua buffered channel bằng non-blocking send để callback lặp lại không block.
- Tách hàm dựng URL, callback handler và browser opener thành seam nhỏ có thể test. Browser thật chỉ là smoke test thủ công; mở browser thất bại vẫn in URL và tiếp tục chờ callback như OAuth hiện tại.

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
| `internal/apierr/apierr.go` | Thêm `UnsupportedModel` (HTTP 400, code `unsupported_model`) | Bản xem trước đã duyệt yêu cầu mã lỗi này khi upstream từ chối model | preview errors, D-002 / AC9, AC10 |
| `internal/proxy/service.go` | `mapUpstreamError` nhận phân loại lỗi; chỉ unknown model → `UnsupportedModel` | Phân biệt model không hỗ trợ với generic validation 400 | preview errors, D-002 / AC9, AC10 |
| `internal/server/openai.go`, `internal/server/anthropic.go` | Dừng stream sau `StreamError`, không phát success terminal | Tránh response vừa lỗi vừa thành công | IC-005 / AC7, AC10 |
| `internal/accounts/summary.go` | Suy ra `reauth_required` từ token file command_code thiếu/không hợp lệ | Hiển thị đúng trạng thái Chưa đăng nhập | preview states / AC5 |
| `internal/provider/testdata/commandcode/` (mới) | Fixture request/callback/NDJSON đã lược credential từ nguồn A-001/A-002 | Không implement schema upstream bằng suy đoán | A-001, A-002 / AC3, AC6, AC7 |
| `internal/provider/commandcode_test.go` (mới) | Test buildRequest, credential, NDJSON terminal/error/limit, sanitization, error mapping | Yêu cầu tests của scope | / AC5-AC9 |
| `internal/config/config_test.go` | Test defaults/validate command_code | Bảo vệ hành vi config | / AC1, AC2 |
| `internal/accounts/summary_test.go` | Test trạng thái token thiếu, lỗi parse, rỗng, hợp lệ và runtime reauth | Bảo vệ trạng thái account | preview states / AC5 |
| `internal/server/openai_test.go`, `internal/server/anthropic_test.go` | Test stream error terminal và tích hợp hai giao thức | Bảo vệ compatibility downstream | / AC7, AC10 |
| `config.example.yaml` (mới — repo chưa có config mẫu, `config.yaml` hiện là config riêng của người dùng) | Thêm provider command_code | Người dùng cấu hình được ngay | D-003 / AC2 |

## Kiểm Tra Hợp Lệ / Lỗi / Trường Hợp Biên
- Alias chưa cấu hình → `apierr.ModelNotFound` (HTTP 404, code `model_not_found`) như hiện tại — spec.md:285 và internal/apierr/apierr.go:32 (manifest ghi "400" nhưng hành vi hiện tại và MVP contract là 404; giữ nguyên).
- Model upstream hợp lệ về alias nhưng structured error xác nhận không có trong catalog → proxy trả HTTP 400 code `unsupported_model`; generic upstream 400 → HTTP 400 code `invalid_request`.
- Token file thiếu, lỗi parse, rỗng hoặc sai prefix khi gọi `Chat`/`Stream` → lỗi auth HTTP 401; account bị đánh dấu reauth.
- HTTP 401/403 trước stream → typed auth error đồng bộ. Event error sau HTTP 200 → phát một `StreamError` terminal với message đã sanitize; không failover và không phát success terminal.
- NDJSON dòng rỗng, JSON không parse được và event type lạ → bỏ qua; event type đã nhận diện nhưng thiếu field bắt buộc, dòng trên 1 MiB, read error hoặc EOF trước terminal → terminal error.
- Callback thiếu/sai `state` hoặc apiKey không bắt đầu `user_` → từ chối HTTP 400, không ghi token.
- Port callback bị chiếm → lỗi khởi động login rõ ràng (hành vi server bind hiện hành).

## Cân Nhắc Bảo Mật / Phân Quyền
- apiKey lưu qua `tokenstore.Save` (0600) như token OAuth; IC-005: không log, không đưa vào error message.
- Callback server bind `127.0.0.1` như `RunOAuthLogin` (internal/provider/oauth.go:67).
- Callback bắt buộc `state` khớp bằng constant-time comparison; thiếu/sai state không được xử lý credential.
- Response login HTML trả về browser không chứa apiKey.
- Raw upstream body không được ghi log hoặc gắn trực tiếp vào error. Error sanitizer phải thay apiKey hiện tại và bearer credential bằng `[REDACTED]` trước mọi `UpstreamError`, `StreamError`, terminal output hoặc downstream response.
- R-002 được ghi nhận: người dùng tự chịu trách nhiệm về điều khoản Command Code.

## Tương Thích / Di Chuyển Dữ Liệu
- Không có migration: provider mới thêm, config cũ không đổi hành vi. `Validate()` cũ không biết `browser_key` — chỉ ảnh hưởng khi người dùng thêm provider mới.
- Endpoint `/v1/chat/completions` và `/v1/messages` của proxy giữ nguyên; thêm alias không phá vỡ client cũ.
- `stop`/`stop_sequences` và `metadata` tiếp tục được endpoint downstream chấp nhận; command_code bỏ qua các field này nhất quán cho đến khi schema upstream được xác nhận. Đây không phải cam kết hỗ trợ stop sequence ở upstream.

## Trình Tự Triển Khai
1. Config: thêm `providerDefaults["command_code"]`, validate `browser_key`, tests config.
2. Trích fixture A-001/A-002 đã lược credential vào testdata; nếu fixture mâu thuẫn spec thì dừng và cập nhật contract.
3. Provider client: `commandcode.go` + tests credential, buildRequest, NDJSON terminal/error/limit và sanitization bằng `httptest.Server`.
4. Factory và account summary: đăng ký command_code, hiển thị trạng thái chưa login đúng, thêm tests.
5. Error mapping và stream writers: thêm `apierr.UnsupportedModel`, phân loại unknown model, bảo đảm `StreamError` terminal; tests OpenAI/Anthropic.
6. Login: hàm login browser mới + phân nhánh `cmdAuth`; test URL/callback/open failure/timeout/cancel bằng seam cùng process.
7. Cập nhật config example, chạy integration server + upstream stub trong cùng process, sau đó `go test ./...` và `go vet ./...`.

## Tiêu Chí Chấp Nhận

### Cấu Hình
- [ ] AC1: Config tối giản chỉ khai `enabled: true` + account command_code vẫn pass `Validate()` nhờ defaults; thiếu `authorization_url`/`redirect_port`/token file → lỗi validate nêu rõ tên provider.
- [ ] AC2: Config example chứa đủ 5 model alias theo D-003 với upstream đúng (`deepseek/deepseek-v4-pro`, `deepseek/deepseek-v4-flash`, `Qwen/Qwen3.6-Max-Preview`, `zai-org/GLM-5`, `MiniMaxAI/MiniMax-M2.7`) và alias không trùng provider khác.

### Đăng Nhập
- [ ] AC3: Fixture đã lược credential xác nhận tên field callback; `auth command_code --account <name>` sinh URL theo dạng `https://commandcode.ai/studio/auth/cli?callback=http://localhost:<port>/callback&state=<hex>`. Browser opener được inject trong test; mở browser thất bại vẫn in URL và tiếp tục chờ callback.
- [ ] AC4: Callback chỉ khi `state` khớp và apiKey hợp lệ `user_...` mới ghi token file với `AccessToken` là apiKey; `tokenstore.Load` đọc lại được. Callback thiếu/sai `state`, thiếu/sai apiKey, callback lỗi, callback lặp lại, timeout hoặc huỷ không ghi/ghi đè token và lệnh trả lỗi phù hợp.

### Provider Client
- [ ] AC5: `provider.Build("command_code", ...)` trả provider implement đủ interface; `Models()` trả alias theo config (BR-004). Token file thiếu/lỗi parse/rỗng/sai prefix làm `ValidateAccount`, `RefreshToken`, `Chat` và `Stream` trả typed auth error; qua proxy nhận HTTP 401 và account chuyển `reauth_required`. `accounts` hiển thị `reauth_required` cho token không hợp lệ, `ok`/`never` cho token hợp lệ.
- [ ] AC6: Fixture A-001 xác nhận schema; body gửi upstream có `params.stream=true` kể cả client non-stream, `threadId` bằng `x-session-id`, `params.system` nhận system/developer text theo thứ tự, `messages` là text blocks, `config` đúng fixture và có đủ headers D-002/BR-003. `stop`/`stop_sequences` và `metadata` được chấp nhận ở downstream nhưng không gửi upstream.
- [ ] AC7: NDJSON success phát đúng một chuỗi `StreamMessageStart` → `StreamContentDelta` theo thứ tự → `StreamMessageStop` kèm usage. Event `error` trước delta/sau delta, known event sai shape, dòng trên 1 MiB, read error và EOF thiếu terminal đều phát đúng một `StreamError` rồi đóng channel; không có stop thành công hoặc `[DONE]` sau lỗi. Dòng rỗng, JSON rác và unknown event được bỏ qua.
- [ ] AC8: Non-stream chỉ trả `ChatResponse` khi nhận success terminal, với content, stop reason và usage đầy đủ; error trước/sau content hoặc EOF thiếu terminal trả error và không trả response dở dang.
- [ ] AC9: Upstream 401/403 → `AuthFailed=true, Retryable=false`; 429/503 → `Retryable=true`; generic 400 → HTTP 400 `invalid_request`; chỉ fixture unknown-model 400 → HTTP 400 `unsupported_model`. Upstream JSON và NDJSON cố tình echo đúng apiKey/Bearer token không làm credential xuất hiện trong provider error, OpenAI/Anthropic response, terminal output hoặc captured log.

### Tích Hợp
- [ ] AC10: Integration test trong cùng process dựng upstream `httptest.Server`, `proxy.Service` và HTTP router thật. Cả `/v1/chat/completions` và `/v1/messages`, stream và non-stream, chạy qua alias command_code với system text, content, usage và stop reason đúng protocol; alias lạ trả HTTP 404 `model_not_found`; unknown-model 400 trả `unsupported_model`; generic 400 trả `invalid_request`; in-stream error là terminal protocol error và không kèm success terminal.

## Ma Trận Xác Minh
| AC | Chiến lược bằng chứng | Bề mặt chính |
|---|---|---|
| AC1 | Test tập trung | `internal/config` |
| AC2 | Test tập trung + xem config example | `internal/config` |
| AC3 | Fixture + test URL/browser opener; browser thật là smoke test không blocking CI | `internal/provider`, `internal/cli` |
| AC4 | Test callback handler cùng process | `internal/provider` |
| AC5 | Test provider + proxy + account summary | `internal/provider`, `internal/proxy`, `internal/accounts` |
| AC6 | Fixture + `httptest.Server` kiểm tra body/header | `internal/provider/commandcode.go` |
| AC7 | Test NDJSON parser và downstream writers | `internal/provider`, `internal/server` |
| AC8 | Test tập trung non-stream aggregation | `internal/provider/commandcode.go` |
| AC9 | Test provider/proxy/server với captured logs | `internal/provider`, `internal/proxy`, `internal/server` |
| AC10 | Integration test toàn bộ trong cùng process | `internal/server` |

## Câu Hỏi Mở
- Không có câu hỏi sản phẩm nào đang blocking.
- Các chi tiết schema upstream được quản lý như cổng bằng chứng A-001/A-002: fixture phải được xác nhận trước implementation; mismatch yêu cầu cập nhật spec, không cho phép executor tự chọn shape.

## Nhật Ký Quyết Định
- 2026-08-17: Duyệt thiết kế revision 1 qua local-runner; các quyết định D-001 (browser studio flow), D-002 (/alpha/generate), D-003 (5 model alias) do human chọn; phần còn lại `agent-proposed-not-objected`.
- Agent chọn (chưa qua human): auth method name `browser_key`, `redirect_port` mặc định 1458, giá trị `x-command-code-version: 0.25.7`, cách tìm apiKey trong callback (query rồi tới body), ValidateAccount không gọi upstream.
- 2026-08-18: Sau review spec, siết `state` callback, typed auth error, sanitization credential, terminal stream error, phân loại 400, account status, giới hạn NDJSON, coverage `/v1/messages` và integration test cùng process; không thay đổi D-001/D-002/D-003.
