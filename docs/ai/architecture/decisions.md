# Quyết Định Kiến Trúc Và Nợ Có Chủ Ý

## Quyết Định

### DEC-001 — Dùng modular monolith với protocol adapters và provider adapters
- Ngày: 2026-08-19 · Trạng thái: active
- Bối cảnh: Proxy có nhiều inbound protocol và upstream provider nhưng vẫn là một binary MVP, trong khi thiếu boundary được ghi rõ khiến code mới dễ đặt sai chỗ và lặp translation hoặc policy.
- Quyết định: Giữ một modular monolith, normalize dữ liệu tại protocol và provider boundaries, đặt routing policy ở application service, và chỉ tạo interface tại boundary có biến thể thật.
- Đã loại: microservices vì tăng deployment và distributed-system cost không cần thiết; Clean Architecture đầy đủ vì tạo nhiều interface và layer hình thức so với quy mô hiện tại; tiếp tục cấu trúc tự phát vì không cung cấp guardrail cho AI agent.
- Chấp nhận: Một số package hiện tại chưa khớp hoàn toàn với boundary mục tiêu và cần được chỉnh dần khi feature chạm tới.
- Xem lại khi: Binary có capability cần deploy hoặc scale độc lập, hoặc dependency graph không còn giữ được boundary bằng Go package.

### DEC-002 — Chuyển đổi kiến trúc tăng dần thay vì rewrite
- Ngày: 2026-08-19 · Trạng thái: active
- Bối cảnh: Runtime và test hiện tại đang hoạt động, còn vấn đề chính là ownership, duplicate wiring và hotspot chứ không phải một kiến trúc hỏng hoàn toàn.
- Quyết định: Giữ behavior hiện tại, áp dụng placement rules cho code mới, và chỉ refactor boundary liên quan trong thay đổi nhỏ có test bảo vệ.
- Đã loại: big-bang rewrite vì blast radius lớn, khó chứng minh compatibility OpenAI/Anthropic và dễ trộn refactor với feature behavior.
- Chấp nhận: Kiến trúc mục tiêu và code thực tế sẽ có giai đoạn chuyển tiếp được ghi rõ trong debt inventory.
- Xem lại khi: Một thay đổi bắt buộc chạm đồng thời phần lớn request path hoặc compatibility test đủ mạnh để bảo vệ một migration lớn hơn.

## Nợ Có Chủ Ý

Chưa có mục nào.
