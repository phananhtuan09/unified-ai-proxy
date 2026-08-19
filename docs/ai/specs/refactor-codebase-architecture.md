# Chuẩn Hoá Kiến Trúc Codebase

## Cấp Độ

Extended.

Điểm độ sâu là 7 vì thay đổi có nhiều hơn hai bề mặt, yêu cầu tương thích ngược, có failure states ở từng checkpoint, đi qua HTTP/streaming boundary, chạm dữ liệu bảo mật và có hơn tám tiêu chí chấp nhận.

## Hợp Đồng Thực Thi

### Mục Tiêu

- Đưa toàn bộ boundary chính của codebase về kiến trúc modular monolith đã duyệt trước khi phát triển feature tiếp.
- Loại duplicate composition giữa CLI và TUI.
- Làm cho normalized request immutable tại routing boundary và bổ sung test riêng cho routing/failover.
- Làm rõ provider interface, shared transport và auth ownership mà không tạo abstraction hình thức.
- Tách cả bốn hotspot đã ghi nhận theo capability trong cùng package.
- Thêm architecture gate tự động để AI và CI phát hiện dependency hoặc placement drift.

### Nguồn Quyết Định Đã Duyệt

- Manifest quyết định: `docs/ai/design-decisions/refactor-codebase-architecture.json` — revision `1`, `approval_meaning: direction-approved`.
- Bản xem trước đầu ra: workflow chuyển từ duplicate composition, mutable routing request, provider boundary rộng và file hotspot sang composition dùng chung, request immutable, provider ownership rõ, file theo capability và architecture gate tự động.
- Trạng thái đã duyệt: baseline xanh; mỗi slice ở trạng thái đang chuyển đổi nhưng vẫn build được; gate thất bại thì dừng; chỉ đạt trạng thái sẵn sàng khi toàn bộ scope và quality gate xanh.
- Flow đã duyệt: characterization tests → composition dùng chung → normalized core/provider cleanup → tách hotspot → architecture quality gate; mọi nhánh lỗi quay lại slice gây lỗi thay vì tiếp tục.
- D-001: hoàn tất composition, routing immutability và tests, provider boundary, shared transport/auth ownership, đồng thời tách cả bốn hotspot trước khi phát triển feature tiếp.
- D-002: giữ architecture docs tự nạp và thêm check tự động cho dependency direction, forbidden generic packages và project quality commands.
- D-003: chỉ mở lại feature development khi mọi slice D-001 hoàn tất, `go test ./...`, `go vet ./...`, `go build ./...` và architecture gate đều pass, không còn known regression trong public contracts.
- BR-001: không thay đổi observable behavior của OpenAI-compatible API, Anthropic-compatible API, streaming SSE, CLI command, TUI flow hoặc config hiện có.
- BR-002: mỗi migration slice phải giữ repository build được và có test bảo vệ boundary vừa thay đổi trước khi sang slice tiếp theo.
- IC-001: giữ modular monolith với protocol adapters, normalized model, routing service và provider adapters theo DEC-001.
- IC-002: migration tăng dần bằng thay đổi nhỏ có test, không big-bang rewrite, theo DEC-002.
- IC-003: folder structure, dependency direction, placement và anti-duplicate rules bám `docs/dev/architecture.md`.
- IC-004: không tạo package `utils`, `common`, `shared`, `services`, `handlers` hoặc abstraction không có owner và consumer rõ.
- IC-005: không log hoặc làm rò API key, OAuth token, authorization header, password hay decrypted backup.
- IC-006: test nằm cạnh package owner; upstream test dùng local test server hoặc fake injected ở test, không thêm mock provider vào production factory.
- Ràng buộc human thêm khi duyệt: Không có.

### Bắt Buộc Xảy Ra

- Baseline behavior phải được khóa bằng test trước khi di chuyển code tại boundary tương ứng.
- CLI và TUI phải dùng chung một composition owner cho object graph `config -> accounts -> proxy -> server`, trong khi vẫn giữ lifecycle/presentation riêng của từng entry adapter.
- `proxy.Service.Chat` và `proxy.Service.Stream` không được mutate `model.ChatRequest` do caller sở hữu.
- Routing/failover phải có unit test riêng cho model resolution, account selection, retry, reauth, timeout/rate-limit mapping, stream setup và input immutability.
- `provider.Provider` chỉ giữ contract mà `proxy.Service` thực sự tiêu thụ cho request routing.
- Account validation/token refresh phải thuộc capability riêng hoặc concrete auth owner; CLI import/auth flow không được ép mọi chat provider implement token refresh giả.
- Shared HTTP transport không được đồng thời giả định OAuth lifecycle.
- Bốn hotspot phải được tách trong package hiện tại theo capability: OpenAI Chat Completions/Responses, Command Code transport/stream/error, config schema/defaults/validation/persistence và TUI update/forms/rendering.
- Architecture gate phải chạy được bằng một command repository-local, kiểm tra dependency direction, forbidden generic package names và các quality commands.
- `CLAUDE.md` phải chỉ tới command gate để AI biết cách tự kiểm tra trước handoff.

### Không Được Xảy Ra

- Không đổi endpoint, request field support, response envelope, HTTP status mapping hoặc SSE event lifecycle hiện tại.
- Không đổi YAML schema/default, CLI command/flag, TUI interaction hoặc persisted token/backup schema.
- Không chuyển sang microservices, full Clean Architecture, dependency injection framework, service locator, global mutable singleton hoặc registration bằng `init()`.
- Không tạo package mới chỉ để giảm line count.
- Không thêm mock provider vào `provider.Build` hoặc production code.
- Không gom OpenAI và Anthropic SSE lifecycle thành một abstraction làm mất semantics riêng của từng protocol.
- Không sửa unrelated behavior hoặc khôi phục `spec.md` đang bị xóa ngoài scope.

## Vấn Đề

Codebase đã có một request pipeline hợp lý nhưng ownership chưa được enforce đồng đều.
CLI và TUI lắp cùng object graph ở hai nơi.
Routing service sửa trực tiếp request đầu vào.
Provider interface trộn chat transport với auth lifecycle, còn `base` trộn shared HTTP state với OAuth token refresh.
Bốn file lớn có nhiều hơn một lý do thay đổi, khiến AI dễ đặt thêm logic vào file gần nhất thay vì đúng capability.
Test hiện bảo vệ tốt một phần provider và Command Code end-to-end flow nhưng chưa có test riêng cho routing/failover hay architecture dependency rules.

## Phạm Vi

- Chuẩn hoá composition, request path và provider/auth boundary.
- Tách `internal/server/openai.go`, `internal/provider/commandcode.go`, `internal/config/config.go` và `internal/tui/app.go` theo capability trong package hiện tại.
- Bổ sung characterization tests, proxy unit tests, composition tests và architecture checks.
- Cập nhật `docs/dev/architecture.md` nếu implementation discovery làm rõ technical detail nhưng không đổi DEC-001, DEC-002 hoặc quyết định design đã duyệt.

## Ngoài Phạm Vi

- Endpoint, provider, auth mode hoặc product behavior mới.
- Thay đổi public API, config schema, CLI/TUI behavior hoặc persisted data.
- Tối ưu performance không liên quan đến refactor.
- Đổi framework Gin, Bubble Tea hoặc storage format.
- Refactor package hỗ trợ không nằm trên dependency path của scope, trừ thay đổi tối thiểu để compile và giữ contract.

## Quyết Định Thiết Kế Đã Duyệt

- D-001 chọn baseline refactor toàn bộ boundary chính thay vì chỉ core; vì vậy cả bốn hotspot là deliverable bắt buộc, không phải debt để hoãn theo feature sau.
- D-002 chọn docs cộng automated gate; implementation phải có executable check, không được kết thúc bằng checklist thủ công.
- D-003 chọn completion gate toàn scope; không được báo sẵn sàng phát triển feature khi còn slice hoặc gate chưa hoàn tất.
- BR-001 biến mọi observable behavior change thành regression; nếu code hiện tại xung đột nguồn contract đã duyệt, dừng và escalate thay vì “sửa tiện”.
- BR-002 yêu cầu từng checkpoint giữ build xanh và test bảo vệ boundary vừa đổi.

## Kiểm Tra Giả Định

### Từ Design Đã Duyệt

- A-001: có thể tạm dừng merge feature mới vào package đang migration — nếu sai, cần ownership/worktree strategy riêng để tránh conflict và feature dựa trên structure trung gian.
- A-002: không có external Go module import trực tiếp package `internal` hoặc phụ thuộc symbol private — nếu sai, di chuyển type/package có thể cần compatibility shim.
- R-001: refactor có thể đổi error mapping hoặc SSE event order dù compile pass — hệ quả là client runtime regression; characterization test phải có trước di chuyển.
- R-002: architecture gate quá cứng có thể chặn dependency hợp lệ — hệ quả là bypass/workaround; gate phải nhỏ, deterministic và thay đổi cùng architecture decision.
- R-003: tách provider interface/base quá sớm có thể tạo nhiều abstraction mỏng — hệ quả là code khó đọc hơn; chỉ tách theo consumer và capability thực tế.

### Đã Xác Nhận

- `go test ./...`, `go vet ./...` và `go build ./...` đều pass trước khi viết spec; đây là baseline kỹ thuật hiện tại.
- `internal/cli/cli.go:cmdStart` và `internal/tui/runtime.go:Runtime.Load` đang lặp object graph.
- `proxy.Service.Chat` và `proxy.Service.Stream` đang mutate `req.Model`, `req.Provider` và `req.Stream`.
- `provider.Provider` đang bắt mọi provider implement `ValidateAccount` và `RefreshToken`, kể cả Gemini API-key provider.
- `base` đang được embed bởi Codex, Gemini và Command Code nhưng sở hữu OAuth token lifecycle chỉ có Codex cần.
- Repository chưa có Makefile/Taskfile/CI architecture check; quality commands hiện nằm trong `CLAUDE.md`.
- Existing integration tests có local upstream và bảo vệ một phần OpenAI/Anthropic envelope, stream terminal behavior, unknown model mapping và secret redaction.

### Suy Luận An Toàn

- Package composition mới có thể nằm tại `internal/app` vì nó sở hữu application object graph, không phải UI/runtime state; tên này không thuộc forbidden generic packages và dependency đi từ entry adapters vào application composition.
- `provider.Provider` có thể thu hẹp về `Name`, `Models`, `ChatCompletion`, `StreamChatCompletion` vì chỉ `proxy.Service` tiêu thụ interface này.
- Auth validation/refresh nên là concrete capability trong `provider` thay vì một interface chung mới nếu chỉ CLI/TUI gọi theo `auth.method`.
- Architecture gate có thể là Go test tại package riêng `internal/architecture` để dùng `go list -json`/source inspection và tự chạy trong `go test ./...`; một script `scripts/check.sh` chỉ orchestration quality commands, không chứa business logic.
- Tách file trong cùng package không làm đổi import path hoặc public symbol contract, phù hợp migration tăng dần.

### Cần Xác Nhận

- Không có câu hỏi sản phẩm nào đang blocking.
- Trước implementation, executor phải kiểm tra worktree/concurrency để xác nhận A-001; nếu có feature khác đang sửa cùng package, dừng và báo conflict thay vì ghi đè.

### Chi Tiết Kỹ Thuật Do Agent Chọn

- Dùng `internal/app` làm composition package, với một aggregate nhỏ chứa `Config`, `Accounts`, `Proxy` và `Server`; lifecycle `Run`, signal handling và TUI mutex/state vẫn ở entry adapter.
- Dùng shallow value copy `routed := *req` trong proxy trước khi thay alias/provider/stream; slices/maps giữ read-only semantics vì routing không mutate nội dung của chúng.
- Tách file nhưng giữ nguyên package và private symbol names khi khả thi để giảm diff và giữ test access.
- Dùng `internal/architecture/architecture_test.go` cho import/forbidden-package rules và `scripts/check.sh` cho `go test`, `go vet`, `go build`; script fail-fast và không tự sửa code.
- Không đặt line-count threshold vào automated gate; line count là review signal, không phải invariant kiến trúc.
- Không thêm interface mới cho composition, config loader, token store hoặc auth flow nếu chưa có consumer cần polymorphism.

## Bằng Chứng Hệ Thống Hiện Tại

- `CLAUDE.md`: đã tự nạp `docs/dev/architecture.md` và định nghĩa quality commands hiện tại.
- `docs/dev/architecture.md`: định nghĩa request pipeline, dependency direction, package ownership, placement, anti-duplicate rules và bốn hotspot.
- `docs/ai/architecture/decisions.md`: DEC-001 chọn modular monolith/adapters; DEC-002 chọn migration tăng dần và loại big-bang rewrite.
- `internal/cli/cli.go:cmdStart`: load config, tạo `accounts.Manager`, `proxy.Service`, `server.Server` và chạy server.
- `internal/tui/runtime.go:Runtime.Load`: lặp đúng graph trên rồi lưu vào TUI runtime.
- `internal/proxy/service.go:Service.Chat` và `Service.Stream`: model resolution, account failover, error mapping và mutation request nằm cùng flow.
- `internal/provider/provider.go:Provider`: chat contract và auth lifecycle đang gộp trong một interface.
- `internal/provider/base.go:base`: giữ provider metadata, HTTP client, config, OAuth access token và refresh behavior.
- `internal/provider/gemini.go:Gemini.RefreshToken`: implementation “not supported” tồn tại chỉ để thỏa interface.
- `internal/provider/commandcode.go`: credential, request shape, transport, NDJSON parser, error classifier và redaction cùng một file.
- `internal/server/openai.go`: OpenAI Responses và Chat Completions parsing/rendering/streaming cùng một file.
- `internal/config/config.go`: schema, path utility, load/save, defaults và validation cùng một file.
- `internal/tui/app.go`: model/state, update, commands, forms và rendering cùng một file.
- `internal/server/commandcode_integration_test.go`: precedent cho local upstream + full handler path + protocol compatibility assertions.
- `internal/provider/gemini_test.go` và `internal/provider/commandcode_test.go`: precedent cho provider adapter mapping, local upstream, stream parser và auth error tests.

## Tiền Lệ Trong Codebase

- Theo mẫu: `internal/server/commandcode_integration_test.go` — precedent gần nhất cho characterization test đi qua config, account manager, provider factory, proxy và HTTP handler mà không gọi upstream thật.
- Phải giống: dùng `httptest.Server`, test observable HTTP/SSE output, kiểm tra terminal success/error và secret redaction; fixture phải dùng fake secret rõ ràng và không log secret.
- Theo mẫu: `internal/provider/gemini_test.go` và `internal/provider/commandcode_test.go` — precedent cho test adapter ở package owner, request mapping và upstream parser bằng local test server.
- Phải giống: test nằm cạnh production package, tên theo behavior, không thêm production mock registry.
- Cố ý khác: proxy và architecture hiện chưa có precedent test; tạo `internal/proxy/service_test.go` với fake `provider.Provider` private trong `_test.go`, và `internal/architecture/architecture_test.go` làm executable boundary check vì đây là capability mới được D-002 yêu cầu.
- Không mirror: kích thước và mixed responsibilities của bốn hotspot; đây là vấn đề cần tách, không phải pattern để sao chép.

## Yêu Cầu Hành Vi

### Baseline Và Checkpoint

- Trước mỗi boundary migration, bổ sung focused test cho behavior chưa được cover.
- Sau mỗi checkpoint, chạy focused tests của package vừa đổi, `go test ./...` và `go build ./...`; chạy `go vet ./...` ít nhất ở final gate hoặc sớm hơn nếu thay interface/concurrency.
- Checkpoint thất bại phải được sửa trước khi tiếp tục; không để expected failure sang slice kế tiếp.

### Composition Dùng Chung

- Một owner duy nhất load config và lắp account manager, proxy service, server và optional logger.
- CLI vẫn sở hữu signal context, terminal output và blocking `Server.Run`.
- TUI vẫn sở hữu start/stop/reload state, logger ring buffer và mutex.
- Composition không mở goroutine, không giữ process signal và không ghi terminal output.
- TUI reload tạo graph mới hoàn chỉnh rồi swap state sau khi build thành công; lỗi build không được phá graph đang giữ.

### Routing Và Normalized Request

- Resolve alias sang upstream model trên một copy của request.
- Non-stream request giữ nguyên `Stream` caller truyền; stream request buộc `Stream=true` chỉ trên copy routed.
- Request gốc giữ nguyên `Model`, `Provider`, `Stream`, message slice, metadata map và stop sequences sau success hoặc error.
- Retry dùng cùng routed request read-only; provider không được mutate normalized request.
- Error mapping hiện tại phải giữ nguyên cho unknown alias, no account, reauth, retryable exhaustion, rate limit, timeout, invalid request, unsupported model và plan restriction.

### Provider Và Auth Boundary

- `provider.Provider` chỉ còn methods phục vụ model catalog và chat/stream routing.
- OAuth token access/refresh chỉ nằm trong OAuth capability dùng bởi Codex.
- Shared HTTP transport giữ provider name, config/models nếu cần và bounded HTTP request helpers, nhưng không sở hữu OAuth token persistence.
- Gemini và Command Code không implement dummy OAuth refresh để thỏa chat interface.
- CLI import vẫn có thể kiểm tra/refresh OAuth account theo `auth.method` mà không type-switch trên concrete provider nếu một explicit provider auth function hiện có đủ ownership; không tạo universal auth interface nếu chưa cần.
- `provider.Build` vẫn là production factory duy nhất cho chat providers và giữ supported provider names hiện tại.

### Tách Hotspot

- `internal/server/openai.go` được tách tối thiểu thành `openai_chat.go`, `openai_responses.go` và shared OpenAI content/error helper khi semantics thật sự chung; routes và response compatibility không đổi.
- `internal/provider/commandcode.go` được tách tối thiểu thành adapter/transport, request model, NDJSON stream parser và error/redaction files; test hiện tại được di chuyển hoặc bổ sung theo owner nhưng behavior không đổi.
- `internal/config/config.go` được tách tối thiểu thành schema/duration, load-save/path, defaults và validation files; exported type/function names và YAML tags không đổi.
- `internal/tui/app.go` được tách tối thiểu thành model/update, commands/forms và views/rendering files; Bubble Tea messages, keys, labels và flow không đổi.
- Không di chuyển package chỉ để tạo folder sâu hơn; file split trong package là mặc định.

### Architecture Gate

- Gate fail nếu package request path import ngược vào entry/adapter layer trái `docs/dev/architecture.md`.
- Tối thiểu enforce: `model` không import project package; `provider` không import `server`, `proxy`, `cli`, `tui`; `proxy` không import `server`, `cli`, `tui`; `server` không import `cli`, `tui`; supporting core package không import entry adapter.
- Gate fail nếu xuất hiện directory package tên `utils`, `common`, `shared`, `services` hoặc `handlers` trực tiếp dưới `internal`.
- Gate không enforce file line count hoặc cấm tên file `helpers.go` bên trong package hiện hữu; rule approved chỉ cấm generic package ownership.
- `scripts/check.sh` chạy theo thứ tự `go test ./...`, `go vet ./...`, `go build ./...` và trả exit code khác zero ngay khi một command fail.
- `CLAUDE.md` ghi command `./scripts/check.sh` là final local gate, vẫn giữ các command Go riêng để debug focused failure.

## Thay Đổi Trạng Thái / Dữ Liệu / Giao Diện

- Không đổi persisted data, YAML/JSON schema, endpoint hoặc public Go module API ngoài package `internal`.
- Thêm in-memory composition aggregate; đây không phải persisted state.
- `model.ChatRequest` không đổi field shape; chỉ đổi ownership semantics thành caller-owned immutable input tại proxy boundary.
- Provider chat interface thu hẹp; đây là internal compile-time contract migration và phải hoàn tất atomically trong checkpoint provider boundary.
- Thêm architecture test/check files; không có runtime production behavior.
- File source được tách nhưng package name và import path hiện tại giữ nguyên.

## Thiết Kế Kỹ Thuật Chi Tiết

### Composition Package

- Trách nhiệm: xây application graph từ config path/config object và optional logger.
- Đầu vào và đầu ra: `Build(configPath string, logger *logs.Store) (*App, error)` hoặc shape tương đương; `App` expose concrete `Config`, `Accounts`, `Proxy`, `Server` cần cho CLI/TUI.
- Chuyển trạng thái hoặc luồng dữ liệu: load/validate config → tạo manager → gọi `proxy.New` với `provider.Build` → gọi `server.New` → trả graph hoàn chỉnh.
- Hành vi lỗi: trả nguyên error có context hiện tại; không partially mutate TUI runtime và không start server.
- Constraint: không chứa signal handling, goroutine lifecycle, terminal/TUI presentation hoặc global state.

### Proxy Service

- Trách nhiệm: alias resolution, account selection, pre-stream retry/failover và canonical API error mapping.
- Đầu vào và đầu ra: giữ signature `Chat(ctx, *model.ChatRequest)` và `Stream(ctx, *model.ChatRequest)` để tránh ripple không cần thiết; lập copy nội bộ trước mutation.
- Chuyển trạng thái hoặc luồng dữ liệu: resolve từ alias gốc → copy request → set upstream/provider/stream trên copy → chọn account → gọi provider → cập nhật health state theo typed error.
- Hành vi lỗi: giữ exact `apierr` code/status/message policy hiện tại; request gốc không đổi ở mọi return path.
- Test seam: builder injection đã có ở `proxy.New`; fake provider trong test ghi lại request và trả scripted errors/events.

### Provider Contract Và Shared Transport

- Trách nhiệm chat interface: `Name`, `Models`, `ChatCompletion`, `StreamChatCompletion`.
- Trách nhiệm transport: bounded HTTP client, JSON request helper, response body limit và status helpers dùng thật sự chung.
- Trách nhiệm OAuth: token load, expiry check, refresh request, persist token, Codex account validation.
- Trách nhiệm API key/browser key: validate credential theo concrete mode; không giả lập refresh semantics.
- Hành vi lỗi: tiếp tục trả `UpstreamError`; secret redaction ở adapter tạo message; proxy chỉ map typed fields.
- Migration: đổi compile-time assertions và call sites cùng checkpoint; không để interface trung gian vừa cũ vừa mới.

### Protocol Files

- Trách nhiệm Chat Completions: allowed fields, request wire types, parse/normalize, non-stream response, finish reason và SSE lifecycle.
- Trách nhiệm Responses: request wire types, input normalization, non-stream response và Responses SSE lifecycle.
- Shared OpenAI helper chỉ chứa content decoding hoặc primitive không làm hai protocol phụ thuộc lifecycle của nhau.
- Hành vi lỗi: tiếp tục render OpenAI-compatible envelope qua `writeOpenAIError`.

### Command Code Files

- Adapter/transport: constructor, endpoint, credential access, headers, Chat/Stream entry methods.
- Request model: wire structs, `buildRequest`, environment config.
- Stream parser: NDJSON types, bounded scanner, event-to-normalized mapping và terminal-state handling.
- Error/redaction: unsupported-model/plan detection, upstream error extraction, secret sanitization.
- Hành vi lỗi: malformed known event, oversized line, EOF trước terminal, upstream auth và stream error giữ behavior test hiện có.

### Config Files

- Schema/duration: exported config structs, YAML tags, `Duration` marshal/unmarshal.
- I/O/path: `DefaultPath`, `ExpandPath`, `Load`, `Save` và permission behavior.
- Defaults: provider defaults, merge và normalization.
- Validation: root/provider/account/model validation và duplicate alias detection.
- Hành vi lỗi: giữ message hiện tại nơi tests hoặc CLI contract đang phụ thuộc; không nới validation.

### TUI Files

- Model/update: state types, Bubble Tea messages, `Init`, `Update`, key dispatch.
- Commands/forms: async commands, form lifecycle, config mutation calls.
- Views/rendering: dashboard/accounts/models/test/log/form render functions và footer/status.
- Runtime lifecycle tiếp tục ở `runtime.go`; không nhập composition lifecycle vào Bubble Tea model.
- Hành vi lỗi: giữ status messages và form data behavior hiện tại.

### Architecture Check

- Dùng Go test để lấy package/import graph bằng `go list -json ./internal/...` hoặc parser tương đương không phụ thuộc tool ngoài Go/Python đã có.
- Rules được khai báo bằng map/table nhỏ với failure message nêu importer, forbidden import và rule trong `docs/dev/architecture.md`.
- Package directory scan chỉ xét direct child package của `internal` cho forbidden generic names.
- Check phải chạy ổn định offline và không sửa file.

## Bản Đồ Thay Đổi Theo File

| Bề mặt | Thay đổi dự kiến | Lý do | Quyết định / AC |
|---|---|---|---|
| `internal/app/app.go` | Tạo composition aggregate và builder dùng chung | Loại duplicate graph | D-001 / AC3-AC5 |
| `internal/app/app_test.go` | Test build success/error và optional logger | Bảo vệ composition | BR-002 / AC4-AC5 |
| `internal/cli/cli.go` | Dùng composition builder trong `cmdStart`; giữ signal/output | Entry adapter chỉ sở hữu lifecycle | D-001 / AC3 |
| `internal/tui/runtime.go` | Dùng composition builder trong `Load`; swap graph sau success | Đồng nhất graph, giữ TUI state | D-001 / AC4 |
| `internal/proxy/service.go` | Route trên request copy, giữ error/failover policy | Loại caller side effect | D-001 / AC6-AC8 |
| `internal/proxy/service_test.go` | Fake provider và table tests cho routing/failover/errors/immutability | Lấp test gap chính | BR-002 / AC6-AC10 |
| `internal/provider/provider.go` | Thu hẹp chat interface; giữ typed upstream errors | Interface theo consumer | D-001 / AC11 |
| `internal/provider/transport.go` | Tách shared HTTP state/helpers khỏi OAuth | Ownership rõ | D-001 / AC12 |
| `internal/provider/oauth_token.go` | Di chuyển access/refresh token capability từ `base.go` | OAuth-only behavior | D-001 / AC12-AC13 |
| `internal/provider/base.go` | Xóa hoặc thu nhỏ sau migration | Không trộn auth với transport | D-001 / AC12 |
| `internal/provider/codex.go` | Compose transport + OAuth capability | Giữ Codex behavior | IC-001 / AC13 |
| `internal/provider/gemini.go` | Dùng transport, bỏ dummy refresh implementation | API-key boundary rõ | D-001 / AC11-AC13 |
| `internal/provider/commandcode.go` | Giữ adapter entry/transport sau file split | Hotspot ownership | D-001 / AC14 |
| `internal/provider/commandcode_request.go` | Chứa wire request/config và mapping | Tách request capability | D-001 / AC14 |
| `internal/provider/commandcode_stream.go` | Chứa NDJSON parser/terminal mapping | Tách streaming capability | D-001 / AC14 |
| `internal/provider/commandcode_errors.go` | Chứa classification/redaction | Tách security/error capability | IC-005 / AC14-AC15 |
| `internal/provider/*_test.go` | Điều chỉnh compile assertions và test theo file owner | Giữ provider behavior | BR-001 / AC13-AC15 |
| `internal/server/openai.go` | Thu nhỏ hoặc xóa sau khi di chuyển shared helper | Loại mixed protocol hotspot | D-001 / AC16 |
| `internal/server/openai_chat.go` | Chat Completions adapter/response/SSE | Capability ownership | BR-001 / AC16-AC18 |
| `internal/server/openai_responses.go` | Responses adapter/response/SSE | Capability ownership | BR-001 / AC16-AC18 |
| `internal/server/*_test.go` | Characterization cho OpenAI/Anthropic/Responses và stream errors còn thiếu | Compatibility gate | BR-001 / AC1-AC2, AC17-AC18 |
| `internal/config/config.go` | Thu nhỏ/xóa sau file split | Loại mixed responsibility | D-001 / AC19 |
| `internal/config/types.go` | Schema và `Duration` | Capability ownership | BR-001 / AC19-AC20 |
| `internal/config/io.go` | Path/load/save | Capability ownership | BR-001 / AC19-AC20 |
| `internal/config/defaults.go` | Provider defaults/merge | Capability ownership | BR-001 / AC19-AC20 |
| `internal/config/validation.go` | Validation rules | Capability ownership | BR-001 / AC19-AC20 |
| `internal/config/config_test.go` | Giữ và mở rộng parity tests khi cần | Config compatibility | BR-001 / AC20 |
| `internal/tui/app.go` | Giữ core model/update hoặc thu nhỏ sau split | Loại hotspot | D-001 / AC21 |
| `internal/tui/commands.go` | Commands và async actions | Capability ownership | BR-001 / AC21-AC22 |
| `internal/tui/forms.go` | Form state/update/submit | Capability ownership | BR-001 / AC21-AC22 |
| `internal/tui/views.go` | View/render helpers | Capability ownership | BR-001 / AC21-AC22 |
| `internal/tui/runtime_test.go` | Thêm Load/composition parity test, giữ config mutation tests | TUI compatibility | BR-002 / AC4, AC22 |
| `internal/architecture/architecture_test.go` | Enforce import direction và forbidden package names | Automated gate | D-002 / AC23-AC25 |
| `scripts/check.sh` | Chạy test/vet/build fail-fast | Unified final gate | D-002 / AC26 |
| `CLAUDE.md` | Thêm final gate command và giữ link architecture | AI tự kiểm tra | D-002 / AC26 |
| `docs/dev/architecture.md` | Đồng bộ exact composition/gate paths nếu implementation chọn khác | Durable guardrail | IC-003 / AC27 |

Tên file chi tiết có thể đổi trong lúc implementation nếu cùng capability và package ownership; thay đổi đó là agent discretion, không được làm giảm scope đã duyệt.

## Kiểm Tra Hợp Lệ / Lỗi / Trường Hợp Biên

- Nil request: giữ behavior hiện tại nếu có test chứng minh; nếu hiện tại panic và không có approved behavior, không tự thêm public validation trong refactor này, nhưng private guard có thể được thêm chỉ khi không đổi observable contract.
- Unknown model phải trả `model_not_found` trước khi chọn account.
- Không có account, tất cả reauth, tất cả unhealthy/disabled phải giữ mapping hiện tại.
- Retry chỉ áp dụng lỗi `Retryable`; auth failure đánh dấu reauth và dừng; non-retryable dừng ngay.
- Streaming chỉ failover trước khi provider trả channel; không retry sau downstream bytes.
- Request gốc phải immutable cả khi resolve lỗi, provider success, auth failure, retry exhaustion và stream setup.
- Composition build error không được để TUI giữ graph partially initialized.
- Logger nil phải tiếp tục disable request logging; logger non-nil phải được gắn vào server.
- OAuth refresh phải giữ timeout, bounded body, token rotation và `0600` persistence.
- Architecture check phải báo nhiều violation trong một run khi khả thi, nhưng exit non-zero nếu có ít nhất một violation.
- `scripts/check.sh` không được phụ thuộc network hoặc config thật.
- File split không được tạo import cycle.
- Test fixture không được chứa token/key có thể nhầm là credential thật; dùng giá trị rõ ràng như `sk-test`, `AIza-test`, `user_integration`.

## Cân Nhắc Bảo Mật / Phân Quyền

- Local `/v1` bearer auth không đổi và phải có characterization test cho missing/invalid key ở OpenAI lẫn Anthropic error envelope khi cần.
- Secret redaction tests của Command Code phải tiếp tục pass sau khi tách error file.
- Không đưa token content vào fake provider call log, architecture check output hoặc test failure message.
- Token/config/backup permission behavior không đổi: file nhạy cảm `0600`, token directory `0700`.
- OAuth/browser callback binding, state validation và timeout nằm ngoài structural split trừ khi cần di chuyển nguyên trạng; test hiện tại phải giữ.
- Không thêm runtime endpoint hoặc debug dump cho architecture verification.

## Tương Thích / Di Chuyển Dữ Liệu

- Không có data migration.
- Giữ nguyên YAML keys/defaults và JSON token/backup schema.
- Giữ endpoint `/health`, `/v1/models`, `/v1/chat/completions`, `/v1/responses`, `/v1/messages` và local auth behavior.
- Giữ OpenAI/Anthropic non-stream và SSE output order/terminal behavior theo characterization tests.
- Giữ CLI syntax và TUI key/label/flow hiện tại.
- Internal interface migration được compile-enforce trong cùng checkpoint; không cần compatibility shim vì Go `internal` không export ra module ngoài theo giả định A-002.
- Rollback theo checkpoint là revert focused commit/slice; không có persisted state cần rollback.

## Trình Tự Triển Khai

1. Ghi nhận baseline: chạy full checks và bổ sung characterization tests còn thiếu cho protocol/auth/SSE behavior trước khi di chuyển implementation.
2. Tạo `internal/proxy/service_test.go` để khóa routing, failover, error mapping và chứng minh request mutation hiện tại; test immutability ban đầu phải fail đúng nguyên nhân trước fix.
3. Sửa proxy route trên request copy; chạy proxy tests và full tests.
4. Tạo `internal/app` composition builder và tests; chuyển CLI `cmdStart`, sau đó TUI `Runtime.Load`; mỗi chuyển đổi là một checkpoint xanh.
5. Thu hẹp `provider.Provider`; tách shared transport khỏi OAuth token capability; migrate Codex, Gemini, Command Code và CLI import/auth call sites atomically; chạy provider/proxy/CLI/TUI tests.
6. Tách Command Code hotspot thành request, stream và error/redaction capabilities; giữ toàn bộ provider/integration tests xanh.
7. Tách OpenAI Chat Completions và Responses protocol files; bổ sung/giữ handler integration tests cho non-stream, stream, invalid field và terminal errors.
8. Tách config schema/I/O/defaults/validation; chạy config round-trip/default/validation tests và full tests.
9. Tách TUI model/update, commands/forms và views; chạy TUI tests, build và thực hiện smoke check thủ công nếu terminal hỗ trợ.
10. Thêm `internal/architecture/architecture_test.go` và `scripts/check.sh`; cố ý tạo violation tạm trong test fixture hoặc unit-test helper để chứng minh gate fail mà không sửa production tree.
11. Cập nhật `CLAUDE.md` và đồng bộ `docs/dev/architecture.md` với path/command thực tế.
12. Chạy final gate `./scripts/check.sh`, rà `git diff --check`, kiểm tra không có secret, không có known public-contract regression và đối chiếu toàn bộ AC trước handoff.

## Tiêu Chí Chấp Nhận

### Baseline Và Compatibility

- [ ] AC1: Trước khi di chuyển protocol/provider behavior, characterization tests cover tối thiểu OpenAI Chat Completions, OpenAI Responses, Anthropic Messages, non-stream success, stream success và terminal stream error ở các path liên quan.
- [ ] AC2: Existing observable endpoint, error envelope, status mapping, SSE terminal behavior, CLI command, TUI flow và config round-trip tests đều pass sau refactor, đáp ứng BR-001.

### Composition

- [ ] AC3: CLI và TUI không còn tự lắp duplicate graph `config -> accounts -> proxy -> server`; cả hai dùng cùng composition owner.
- [ ] AC4: TUI `Runtime.Load` chỉ swap graph sau khi composition build thành công và vẫn giữ logger/lifecycle behavior hiện tại.
- [ ] AC5: Composition tests chứng minh success, config/build error và nil/non-nil logger behavior mà không start network listener.

### Routing Và Failover

- [ ] AC6: `Service.Chat` không mutate bất kỳ field nào của caller `ChatRequest` trên success hoặc error.
- [ ] AC7: `Service.Stream` chỉ đặt upstream model/provider/`Stream=true` trên internal copy và không mutate caller request.
- [ ] AC8: Provider nhận đúng upstream model/provider, trong khi caller vẫn giữ alias ban đầu.
- [ ] AC9: Proxy unit tests cover no account, auth failure/reauth, retryable failover, non-retryable error, timeout/rate-limit exhaustion và unknown model mapping.
- [ ] AC10: Streaming tests chứng minh failover chỉ xảy ra trước khi channel được trả và không thay đổi normalized event contract.

### Provider Boundary

- [ ] AC11: `provider.Provider` chỉ còn methods phục vụ chat routing/catalog; Gemini và Command Code không còn dummy `RefreshToken` chỉ để thỏa interface.
- [ ] AC12: Shared HTTP transport không import hoặc sở hữu tokenstore/OAuth refresh lifecycle.
- [ ] AC13: Codex OAuth refresh, Gemini API-key validation và Command Code browser-key credential behavior giữ nguyên theo focused tests.
- [ ] AC14: Command Code request mapping, NDJSON parser và error/redaction nằm ở file capability riêng trong `provider` package.
- [ ] AC15: Toàn bộ Command Code parser, terminal error, oversized line, upstream classification và secret-redaction tests pass.

### Protocol Và Hotspot Split

- [ ] AC16: OpenAI Chat Completions và Responses nằm ở file capability riêng; shared helper không trộn SSE lifecycle của hai protocol.
- [ ] AC17: OpenAI Chat Completions/Responses và Anthropic compatibility tests pass cho non-stream và stream success.
- [ ] AC18: Invalid field, unknown model, auth error và terminal stream error vẫn tạo đúng protocol envelope/terminal behavior.
- [ ] AC19: Config schema/duration, I/O/path, defaults và validation nằm ở file capability riêng trong cùng package, giữ exported symbols/YAML tags.
- [ ] AC20: Config validation, defaults và save/load round-trip tests pass, file permission behavior không đổi.
- [ ] AC21: TUI model/update, commands/forms và rendering được tách theo capability trong cùng package mà không đổi key bindings, labels hoặc message flow.
- [ ] AC22: TUI unit tests pass và smoke check xác nhận load/start-stop navigation cơ bản không có regression quan sát được, hoặc ghi rõ constraint nếu terminal smoke không khả thi.

### Architecture Gate Và Hoàn Thành

- [ ] AC23: Architecture test fail với thông báo importer/imported rõ ràng khi một forbidden dependency được đưa vào test fixture.
- [ ] AC24: Architecture test fail khi xuất hiện forbidden generic direct package dưới `internal` và không áp line-count threshold.
- [ ] AC25: Current production import graph pass architecture test và không có package mới vi phạm `docs/dev/architecture.md`.
- [ ] AC26: `./scripts/check.sh` chạy fail-fast `go test ./...`, `go vet ./...`, `go build ./...`; command được ghi trong `CLAUDE.md`.
- [ ] AC27: `docs/dev/architecture.md` khớp composition path, provider boundary và gate command thực tế sau implementation.
- [ ] AC28: Toàn bộ scope D-001 hoàn tất, final gate pass, `git diff --check` pass và không còn known regression trong public contracts trước khi báo sẵn sàng phát triển feature tiếp, đáp ứng D-003.

## Ma Trận Xác Minh

| AC | Chiến lược bằng chứng | Bề mặt chính |
|---|---|---|
| AC1 | Test tập trung/integration được thêm trước file move | `internal/server/*_test.go`, `internal/provider/*_test.go` |
| AC2 | Full regression suite và protocol assertions | `go test ./...` |
| AC3 | Code inspection + composition tests | `internal/app`, `internal/cli`, `internal/tui/runtime.go` |
| AC4 | Unit test Load success/failure + code inspection | `internal/tui/runtime_test.go` |
| AC5 | Unit test không mở listener | `internal/app/app_test.go` |
| AC6 | Table test snapshot request trước/sau Chat | `internal/proxy/service_test.go` |
| AC7 | Table test snapshot request trước/sau Stream | `internal/proxy/service_test.go` |
| AC8 | Fake provider capture routed request | `internal/proxy/service_test.go` |
| AC9 | Scripted fake provider/account states | `internal/proxy/service_test.go` |
| AC10 | Stream setup/failure tests | `internal/proxy/service_test.go` |
| AC11 | Compile assertions + code inspection | `internal/provider/provider.go`, providers |
| AC12 | Import/code inspection + architecture test | `internal/provider/transport.go` |
| AC13 | Existing và focused auth/provider tests | `internal/provider/*_test.go` |
| AC14 | File/symbol inspection + provider tests | `internal/provider/commandcode*.go` |
| AC15 | Existing Command Code unit/integration tests | `internal/provider/commandcode_test.go`, `internal/server/commandcode_integration_test.go` |
| AC16 | File/symbol inspection | `internal/server/openai_*.go` |
| AC17 | Handler integration tests | `internal/server/*_test.go` |
| AC18 | Error/stream integration tests | `internal/server/*_test.go` |
| AC19 | File/symbol inspection | `internal/config/*.go` |
| AC20 | Config unit tests + permission assertion | `internal/config/config_test.go` |
| AC21 | File/symbol inspection + TUI tests | `internal/tui/*.go` |
| AC22 | Unit tests + manual terminal smoke evidence | `internal/tui`, execution summary |
| AC23 | Architecture test self-fixture | `internal/architecture/architecture_test.go` |
| AC24 | Architecture test self-fixture | `internal/architecture/architecture_test.go` |
| AC25 | Architecture test trên current graph | `go test ./internal/architecture` |
| AC26 | Chạy command thật | `./scripts/check.sh`, `CLAUDE.md` |
| AC27 | Artifact/code comparison | `docs/dev/architecture.md` |
| AC28 | Full gate + diff/security inspection | `./scripts/check.sh`, `git diff --check` |

## Câu Hỏi Mở

- Không có câu hỏi sản phẩm nào đang blocking.
- A-001 phải được kiểm tra lại ở đầu execution để tránh đụng thay đổi đang diễn ra trong cùng package.

## Nhật Ký Quyết Định

- 2026-08-19: Dùng design manifest revision `1` làm authority; validator xác nhận checksum và provenance hợp lệ.
- 2026-08-19: Phân loại spec là Extended do compatibility, security, streaming boundary và số bề mặt/AC.
- 2026-08-19: Chọn `internal/app` làm technical composition owner theo agent discretion; lifecycle vẫn ở CLI/TUI.
- 2026-08-19: Chọn thu hẹp `provider.Provider` theo consumer và tách OAuth capability khỏi shared transport; không tạo universal auth abstraction mới nếu concrete auth functions đủ dùng.
- 2026-08-19: Chọn architecture test bằng Go và quality orchestration bằng `scripts/check.sh`; không enforce line count.
- 2026-08-19: Giữ một executable spec vì mọi slice cùng phục vụ một outcome không độc lập: baseline kiến trúc phải hoàn tất toàn scope D-001 trước khi mở lại feature development.
