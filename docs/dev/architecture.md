# Kiến Trúc Và Quy Tắc Code

## Trạng Thái

Tài liệu này là nguồn chuẩn cho kiến trúc ứng dụng và vị trí code của Unified AI Proxy.
AI agent phải đọc tài liệu này trước khi thêm package, provider, endpoint, service, helper hoặc model mới.
Runtime behavior, test và approved feature spec vẫn quyết định hành vi cụ thể của từng tính năng.
Khi code hiện tại lệch với tài liệu này, không được nhân bản lệch chuẩn đó vào code mới.
Hãy ghi nhận lệch chuẩn là technical debt và sửa trong một thay đổi riêng nếu việc sửa không thuộc scope hiện tại.

## Quyết Định Kiến Trúc

Dự án dùng **modular monolith với protocol adapters và provider adapters**.
Đây là cách gọi chính thức cho pattern đã tồn tại một phần trong code hiện tại.
Dự án không áp dụng Clean Architecture đầy đủ, không tạo repository/use-case/interface cho mọi type, và không chia package chỉ để đạt hình thức nhiều layer.
Lý do và phương án đã loại được ghi tại `DEC-001` trong `docs/ai/architecture/decisions.md`.
Chiến lược chuyển đổi tăng dần, không rewrite, được ghi tại `DEC-002`.

Luồng request chuẩn là:

```text
OpenAI / Anthropic HTTP
          |
          v
internal/server -> internal/proxy -> internal/provider -> Upstream API
          \              |                  /
           +------ internal/model --------+
```

`internal/model` là normalized contract được các layer sử dụng, không phải một runtime service mà request phải gọi qua.

CLI và TUI là hai entry adapter dùng chung composition root:

```text
main -> cli -----\
                 > internal/app.Build -> server -> proxy -> provider
       tui -----/
```

`internal/app` là composition owner của graph `config -> accounts -> proxy -> server`.
`internal/app` chỉ dựng object graph và không sở hữu process hoặc server lifecycle.
CLI sở hữu signal handling, terminal output và blocking server lifecycle.
TUI sở hữu background lifecycle, mutex, state và presentation.

## Nguyên Tắc Cốt Lõi

### 1. Normalize once at boundaries

Wire model của OpenAI, Anthropic, Gemini, Codex hoặc Command Code chỉ được tồn tại trong adapter tương ứng.
`internal/server` chuyển inbound protocol thành `model.ChatRequest` và chuyển normalized response/event thành outbound protocol.
`internal/provider` chuyển normalized request thành upstream wire format và chuyển upstream response/event về normalized model.
`internal/proxy` không được parse JSON, tạo SSE frame hoặc biết field riêng của một protocol.

### 2. Policy ở giữa, I/O ở biên

Routing model, chọn account, retry, failover và mapping lỗi upstream thuộc `internal/proxy`.
HTTP transport và parsing upstream thuộc `internal/provider`.
HTTP auth cục bộ, status code và response envelope thuộc `internal/server`.
Đọc/ghi YAML, token file và backup thuộc package hạ tầng hiện có.

### 3. Dependency đi vào normalized core

Dependency hợp lệ cho request path là:

```text
server -> proxy -> provider
   |         |         |
   +-------> model <---+
   +-------> apierr <--+
```

Dependency restrictions tối thiểu là:

| Importer | Không được import |
|---|---|
| `model` | mọi application package trong project |
| `provider` | `app`, `server`, `proxy`, `cli`, `tui` |
| `proxy` | `app`, `server`, `cli`, `tui` |
| `server` | `app`, `cli`, `tui` |
| `config` | `app`, `accounts`, `provider`, `proxy`, `server`, `cli`, `tui` |
| `accounts` | `app`, `provider`, `proxy`, `server`, `cli`, `tui` |
| `tokenstore` | `app`, `accounts`, `provider`, `proxy`, `server`, `cli`, `tui` |
| `app` | `cli`, `tui` |

Entry adapter được phụ thuộc vào application packages, nhưng application packages không được import ngược `cli` hoặc `tui`.
`app` được phụ thuộc vào các concrete application packages để dựng graph, nhưng không package nào trong graph được import ngược `app`.
Không được để provider gọi trực tiếp provider khác.
Không được để handler chọn account hoặc tự retry upstream.
Không được để UI gọi trực tiếp upstream API.

### 4. Interface chỉ ở nơi có biến thể thật

Giữ `provider.Provider` vì có nhiều upstream implementation và proxy cần thay thế implementation trong test.
Không tạo interface cho struct chỉ có một implementation.
Interface mới phải được consumer sở hữu hoặc nằm ở boundary ổn định, và phải có ít nhất hai implementation thực tế hoặc một test seam có giá trị rõ ràng.

### 5. Package theo trách nhiệm, file theo capability

Giữ package ở mức capability ổn định như `server`, `proxy`, `provider`, `accounts`, `config`, `tokenstore`.
Không tạo các package chung chung như `utils`, `common`, `shared`, `services`, `handlers` hoặc `helpers` ở root `internal`.
Helper chỉ dùng trong một package phải ở lại package đó và mang tên theo capability, ví dụ `sse.go`, `errors.go`, `oauth.go`.
Chỉ tách package khi có ownership rõ, dependency direction rõ và nhiều hơn một caller thực tế.

## Ownership Theo Package

| Package | Sở hữu | Được phụ thuộc vào | Không được sở hữu |
|---|---|---|---|
| `main` | process entry tối thiểu | `cli` | config, business logic, HTTP wiring |
| `cli` | parse command, terminal I/O, process lifecycle | application packages | provider wire format, routing policy |
| `tui` | Bubble Tea state và interaction | application packages | upstream HTTP, duplicate business policy |
| `app` | load config và dựng application object graph | `config`, `accounts`, `provider`, `proxy`, `server`, `logs` | signal handling, goroutine lifecycle, terminal output, TUI state, global singleton |
| `server` | route, local auth, protocol validation/translation, SSE output | `proxy`, `model`, `apierr` | account selection, upstream parsing |
| `proxy` | model resolution, account selection, retry/failover, canonical error mapping | `accounts`, `provider`, `model`, `apierr` | Gin, protocol JSON/SSE, token persistence |
| `provider` | upstream auth/HTTP/wire translation/stream parsing | `model`, `config`, credential storage | local API response shape, cross-provider routing |
| `model` | normalized request, response, stream event và shared value objects | standard library | config, transport, persistence, UI concerns |
| `accounts` | account registry, health state, round-robin selection | `model`, config input | upstream calls, HTTP errors |
| `config` | YAML schema, defaults, validation, file persistence | standard library và YAML library | runtime health, request routing |
| `tokenstore` | secure token-file persistence | token model | OAuth flow, account selection |
| `backup` | encrypted export/import | config and filesystem | CLI presentation, runtime routing |
| `apierr` | canonical application error code và HTTP intent | standard library | OpenAI/Anthropic response envelope |
| `logs` | bounded in-memory log data | standard library | formatting UI hoặc HTTP middleware policy |

## Placement Rules

Trước khi tạo file hoặc symbol mới, tìm implementation tương đương bằng tên capability và hành vi.
Nếu logic đã có một owner, mở rộng owner đó thay vì tạo bản sao ở caller.

| Thay đổi | Vị trí mặc định |
|---|---|
| Thêm OpenAI/Anthropic request field | file protocol tương ứng trong `internal/server` và normalized type chỉ khi provider cần field đó |
| Thêm public endpoint | route và adapter trong `internal/server`; policy dùng chung ở service owner |
| Thêm upstream provider | file riêng trong `internal/provider`, đăng ký tại `factory.go`, config tại `internal/config` |
| Thêm routing/failover rule | `internal/proxy` |
| Thêm account health/selection rule | `internal/accounts` |
| Thêm normalized chat capability | `internal/model`, sau khi xác nhận nhiều protocol/provider cần nó |
| Thêm credential persistence | `internal/tokenstore` hoặc package storage có ownership cụ thể |
| Thêm CLI command | dispatch/presentation trong `internal/cli`; logic tái sử dụng đặt ở package owner |
| Thêm TUI action | interaction trong `internal/tui`; logic tái sử dụng đặt ở package owner |
| Thêm unit test | `tests/unit/<capability>` |
| Thêm integration test | `tests/integration/<flow-or-boundary>` |
| Thêm architecture rule test | `tests/architecture` |

Không đặt business behavior vào `main.go`.
Không dùng `gin.H`, `http.Request`, `http.Response` hoặc protocol JSON tag ngoài adapter HTTP liên quan.
Ngoại lệ là provider adapter được dùng `net/http` và upstream wire structs vì đó chính là transport boundary của nó.

## Pattern Được Chấp Nhận

### Request Ownership

`internal/server` sở hữu inbound wire request và tạo `model.ChatRequest` normalized.
`proxy.Service` coi request của caller là immutable và tạo routed copy trước khi thay alias, provider hoặc stream flag.
Provider coi routed request là read-only.
Shallow copy chỉ hợp lệ khi routing và provider không mutate nội dung slice, map, metadata hoặc stop sequences.

### Protocol Adapter

Mỗi inbound protocol có ba bước rõ ràng: parse/validate, normalize, render response/stream.
OpenAI và Anthropic được phép khác response envelope nhưng phải dùng cùng normalized service contract.
Validation dành riêng cho protocol phải xảy ra trước khi gọi `proxy.Service`.

### Provider Adapter

Mỗi provider chịu trách nhiệm cho endpoint, auth header, request mapping, response parsing, stream parsing và upstream error extraction của chính nó.
Provider phải trả `model.ChatResponse`, `model.StreamEvent` hoặc typed `provider.UpstreamError` thay vì trả wire payload ra ngoài package.
Credential hoặc secret từ request/upstream không được xuất hiện trong error trả về, log hoặc test failure output.

### Routing Và Failover

`proxy.Service` sở hữu model resolution, account selection, health update, retry và pre-stream failover.
Retry chỉ áp dụng cho typed retryable error.
Auth failure phải cập nhật account state theo policy hiện tại và không được bị biến thành generic retry.
Server và provider không được tự cài đặt routing policy riêng.

### Streaming

Provider chuyển upstream stream thành `model.StreamEvent` normalized.
Proxy chỉ được failover khi provider chưa trả channel cho caller.
Sau khi channel được trả hoặc downstream đã nhận event hay byte, không retry hoặc failover sang provider khác.
Server sở hữu SSE framing, protocol event order và terminal event.
OpenAI Chat Completions, OpenAI Responses và Anthropic Messages không được dùng chung lifecycle abstraction làm mất semantics riêng.

### Application Service

`proxy.Service` là application service cho chat routing.
Service điều phối collaborator và policy nhưng không sở hữu transport details.
Service không được mutate input do caller sở hữu; hãy clone request trước khi thay model alias, provider hoặc stream flag.

### Composition Root

Khởi tạo concrete dependency chỉ diễn ra ở entry/composition boundary.
`internal/app.Build` là composition boundary dùng chung cho CLI và TUI.
`app.Build` phải trả một graph hoàn chỉnh hoặc error, không trả partial graph.
`app.Build` không mở listener, không chạy goroutine, không giữ process signal và không ghi terminal output.
TUI phải build graph mới thành công trước khi swap runtime state.
Factory provider được inject vào proxy để test không cần network thật.
Không dùng global service locator, mutable singleton hoặc `init()` để đăng ký behavior.

### Authentication

`provider.Provider` chỉ chứa model catalog và chat/stream routing contract.
OAuth access, refresh và token persistence thuộc concrete OAuth capability của Codex.
API-key và browser-key validation thuộc concrete auth mode tương ứng.
Shared provider transport không được sở hữu OAuth refresh hoặc token persistence.
Không tạo universal auth interface nếu chưa có nhiều consumer cần cùng một polymorphic contract.

### Configuration Và Persistence

`internal/config` sở hữu YAML schema, defaults, validation, path expansion và config file persistence.
Config package không sở hữu runtime health, account selection hoặc request routing.
`internal/tokenstore` sở hữu token file persistence nhưng không sở hữu OAuth flow.
`internal/backup` sở hữu encrypted export/import nhưng không sở hữu CLI presentation.
Thay đổi schema, default hoặc persistence format phải có compatibility decision và round-trip test rõ ràng.
File nhạy cảm phải giữ permission theo Security Rules.

### Entry Adapter Lifecycle

CLI sở hữu command dispatch, signal context, terminal output và blocking `Server.Run`.
TUI sở hữu Bubble Tea state, background start/stop/reload, logger ring buffer và synchronization.
CLI và TUI dùng chung application graph từ `app.Build` nhưng không chia sẻ presentation hoặc process lifecycle abstraction.
Business policy dùng chung phải nằm trong package owner, không copy giữa CLI và TUI.

### Typed Errors

Provider phân loại lỗi upstream bằng `provider.UpstreamError`.
Proxy ánh xạ lỗi đó thành `apierr.APIError` ổn định.
Server render canonical error theo OpenAI hoặc Anthropic envelope.
Không dò chuỗi lỗi bên ngoài adapter đã tạo ra chuỗi đó khi có thể dùng typed error hoặc sentinel error.

## Quy Tắc Chống Duplicate

Một đoạn code giống nhau chưa đủ để tạo abstraction.
Chỉ trích abstraction khi các đoạn code có cùng lý do thay đổi và cùng semantics.

Áp dụng thứ tự sau:

1. Tìm implementation hiện có trong package owner.
2. Dùng lại function/type hiện có nếu semantics trùng khớp.
3. Mở rộng function hiện có nếu thay đổi vẫn thuộc một responsibility.
4. Trích helper có tên theo capability khi logic lặp ít nhất hai nơi trong cùng package.
5. Trích package mới chỉ khi nhiều package cần cùng capability và dependency direction vẫn hợp lệ.

Không hợp nhất OpenAI SSE và Anthropic SSE chỉ vì cả hai cùng ghi `event:` và `data:`.
Hai protocol có lifecycle và compatibility contract khác nhau.
Có thể chia sẻ primitive framing nhỏ nếu test chứng minh không làm mất semantics của từng protocol.

## Quy Tắc Kích Thước Và Tách File

Line count là tín hiệu để review, không phải luật tự động.
Khi file vượt khoảng 400 dòng hoặc có hơn một lý do chính để thay đổi, phải xem xét tách theo capability trong cùng package trước.
Không tạo package mới chỉ để giảm line count.

Các hotspot hiện tại cần tránh phình thêm không kiểm soát:

- `internal/tui/app.go` đang trộn state transition, command, form và rendering.
- `internal/server/openai_chat.go` đang chứa cả Chat Completions và Responses protocol lifecycle.

Khi sửa đáng kể một hotspot, ưu tiên tách file trong cùng package theo capability đang thay đổi.
Ví dụ hợp lệ là `openai_chat.go`, `openai_responses.go`, `commandcode_stream.go` hoặc `config_validation.go`.
Không yêu cầu refactor các hotspot này trước mọi feature.

## Checklist Thêm Provider

Một provider mới hoàn thành khi có đủ các phần liên quan sau:

1. Xác nhận contract upstream từ tài liệu hoặc runtime evidence; không suy ra endpoint từ tên model.
2. Thêm config/default/validation tối thiểu trong `internal/config`.
3. Tạo adapter riêng trong `internal/provider` và implement `provider.Provider`.
4. Map normalized request sang wire request mà không thêm behavior ngoài contract.
5. Map non-stream response và stream event về normalized model.
6. Phân loại auth, retry, timeout, unsupported model và plan restriction bằng typed error khi upstream hỗ trợ.
7. Redact API key, token và authorization data khỏi mọi error/log.
8. Đăng ký provider ở `internal/provider/factory.go`.
9. Thêm unit test cho mapping và parsing, integration test qua server/proxy cho behavior quan trọng.

Không thêm `switch providerName` vào server, proxy hoặc UI nếu factory/config/provider adapter có thể sở hữu khác biệt đó.

## Checklist Thêm Hoặc Mở Rộng Protocol

1. Định nghĩa wire structs trong file của protocol tại `internal/server`.
2. Reject hoặc ignore field theo contract đã được duyệt.
3. Normalize về `model.ChatRequest` mà không để wire type rò sang proxy.
4. Dùng `proxy.Service` cho model routing, account selection và failover.
5. Render non-stream response đúng protocol.
6. Render đầy đủ stream lifecycle và terminal error đúng protocol.
7. Thêm handler/integration test cho success, invalid input, auth, unknown model và stream terminal behavior liên quan.

## Testing Rules

Tất cả test mới phải nằm dưới top-level `tests/`.
Không tạo file `*_test.go` mới trong `internal/*`, root package hoặc các production package khác.
Test phải kiểm tra qua exported contract hoặc stable boundary của package owner.
Không export production symbol chỉ để test private implementation detail.
Nếu behavior chỉ có thể test bằng private access, ưu tiên test qua public boundary gần nhất hoặc tách một capability contract có giá trị production thật.

Test structure chuẩn là:

```text
tests/
  unit/           deterministic package behavior through exported contracts
  integration/    multi-package, HTTP handler, provider adapter and persistence flows
  architecture/   dependency direction, placement and forbidden-package rules
  fixtures/       secrets-safe static test data
  testkit/        test-only builders and fakes reused by more than one test package
```

Test-only fake, builder hoặc helper chỉ dùng một test package phải nằm cùng folder test đó.
Chỉ đưa helper vào `tests/testkit` khi có ít nhất hai test package thật sự dùng chung cùng semantics.
Provider test dùng `httptest.Server`; không gọi upstream thật trong test suite mặc định.
Proxy policy phải được test bằng fake provider được inject qua builder hoặc interface, không thêm mock provider vào production factory.
Server integration test phải đi qua `http.Handler` và xác nhận protocol envelope/SSE observable.
Bug fix phải có regression test ở boundary thấp nhất tái hiện đúng lỗi.
Chạy focused test trong khi sửa, sau đó chạy `go test ./...` và `go vet ./...` trước handoff.
Final local gate là `./scripts/check.sh`, chạy test, vet và build theo thứ tự fail-fast.

## Security Rules

API key, OAuth token, authorization header, password và decrypted backup là secret.
Không log, format vào error, lưu trong fixture commit vào Git hoặc đưa vào snapshot output.
File chứa config/token/backup nhạy cảm phải giữ permission `0600`; directory chứa token phải giữ `0700`.
Callback auth phải bind loopback, validate state và có timeout.
Mọi body đọc từ network hoặc file không tin cậy phải có bound phù hợp.
Local `/v1` endpoint phải tiếp tục yêu cầu bearer auth trừ khi approved contract thay đổi rõ ràng.

## Known Debt Và Migration Triggers

Đây là debt inventory, không phải yêu cầu rewrite ngay:

| Debt hiện tại | Rủi ro | Trigger để xử lý |
|---|---|---|
| OpenAI Chat Completions và Responses còn chung `openai_chat.go` | protocol lifecycle dễ bị trộn | trước khi mở rộng một trong hai protocol |
| TUI state, update, forms và rendering còn chung `app.go` | AI dễ đặt logic sai capability | trước thay đổi TUI lớn tiếp theo |
| Test cho routing/failover còn mỏng | policy regression khó phát hiện | trước khi mở rộng retry, health hoặc multi-provider routing |
| Test hiện tại còn nằm trong production package | test layout không khớp rule `tests/` mới | di chuyển theo package trong thay đổi test-only riêng, không trộn với feature behavior |

Mỗi migration phải nhỏ, giữ behavior và có test bảo vệ.
Mặc định, chỉ migration boundary liên quan đến feature hiện tại.
Một approved architecture migration có thể bao phủ nhiều boundary, nhưng phải chia thành các slice hoặc checkpoint độc lập và giữ repository xanh sau mỗi checkpoint.
Không thực hiện big-bang rewrite không có characterization tests hoặc rollback boundary.

## Enforcement Map

| Rule | Enforcement chính |
|---|---|
| Dependency direction | architecture test dưới `tests/architecture` |
| Forbidden generic package names | architecture test dưới `tests/architecture` |
| Test chỉ nằm dưới `tests/` | architecture test quét file `*_test.go` |
| Test, vet và build | `./scripts/check.sh` |
| Request immutability | proxy unit/integration test |
| Error ownership và mapping | provider/proxy/server tests và review |
| SSE lifecycle | handler integration tests |
| Secret redaction | provider và integration tests |
| Không tạo abstraction hình thức | review theo package ownership và precedent |

## Checklist Cho AI Agent Trước Khi Code

Trước khi implement, trả lời được các câu sau:

1. Behavior này thuộc protocol adapter, routing policy, provider adapter hay supporting capability nào?
2. Package owner hiện tại là package nào?
3. Đã tìm implementation tương đương và test liên quan chưa?
4. Type mới là normalized model hay chỉ là wire model của một boundary?
5. Dependency mới có đi ngược sơ đồ không?
6. Logic này có đang bị copy giữa CLI, TUI, server hoặc provider không?
7. Error có được phân loại ở đúng layer và render ở protocol boundary không?
8. Secret có thể lọt vào error, log hoặc fixture không?
9. Test nào chứng minh behavior và compatibility sau thay đổi?

Nếu không xác định được owner mà phải tạo `utils` hoặc một interface chung chung, dừng lại và làm rõ thiết kế trước khi code.
