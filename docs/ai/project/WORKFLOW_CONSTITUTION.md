---
phase: project
title: Workflow Constitution
description: Nguồn nguyên tắc duy nhất cho kiến trúc và sự phát triển của AI agent workflow
---

# Hiến Pháp Workflow

## Mục Đích

Tài liệu này định nghĩa các nguyên tắc nền tảng của AI agent workflow trong repository.

Workflow, orchestrator, skill và artifact có thể thay đổi theo project.

Tài liệu này không mô tả step, command hoặc skill cụ thể của một workflow.

Tài liệu này có quyền quy định các quy ước bắt buộc mà mọi workflow trong repository phải tuân theo.

## North Star

Workflow phải khai thác năng lực của AI đến giới hạn còn kiểm chứng và phục hồi được, để AI hoàn thành và tự kiểm tra công việc trước khi giao kết quả cho human.

Human tập trung vào việc làm rõ intent, chốt các quyết định quan trọng và đánh giá kết quả cuối cùng thay vì giám sát chi tiết quá trình thực hiện.

Workflow không tồn tại để buộc AI tuân thủ một quy trình suy nghĩ cố định.

Workflow tồn tại để:

- cung cấp đủ context và authority cho AI làm việc hiệu quả
- bảo vệ các boundary có rủi ro cao
- yêu cầu AI tự kiểm tra những gì có thể kiểm tra bằng code, tool và runtime
- giúp human review kết quả nhanh với evidence và giới hạn rõ ràng
- giảm số lần can thiệp và cognitive load của human

## Kiến Trúc Cốt Lõi

Workflow coding có ba phase:

```text
Plan → Implement → Validate
```

Ba phase thể hiện sự chuyển giao trách nhiệm chính giữa human và AI.

Mỗi project có thể dùng số lượng step, skill, tool, agent, test và artifact khác nhau bên trong từng phase.

Ba phase là ranh giới trách nhiệm, không phải yêu cầu về artifact.

Công việc nhỏ có thể thỏa mãn một phase mà không sinh ra artifact nào.

Artifact và evidence phải tăng theo risk và blast radius của công việc, không theo hình thức của workflow.

Không tạo thêm phase nếu boundary mới không thay đổi responsibility, authority hoặc outcome có ý nghĩa.

Quyết định một việc có đáng làm hay không nằm trước Plan và thuộc về human.

Ba phase chỉ áp dụng cho công việc đã được quyết định là sẽ làm.

### Plan

Plan làm rõ yêu cầu, task và human intent trước khi implementation bắt đầu.

Phase này chốt những nội dung có thể làm AI implementation sai hướng, bao gồm scope, expected output, behavior, constraints và các quyết định quan trọng khác khi có liên quan.

Human chịu trách nhiệm giải thích yêu cầu, xác nhận intent và chốt các quyết định quan trọng về product, business, scope hoặc preference.

AI chịu trách nhiệm khám phá codebase, phân tích feasibility và viết detailed implementation spec.

Human không phải quyết định technical detail thông thường nếu chúng không tạo ra một trade-off quan trọng.

Plan hoàn thành khi các quyết định quan trọng đã được chốt và AI có một spec đủ rõ để tự thực hiện phase Implement.

### Implement

Implement là phase do AI chịu trách nhiệm thực hiện toàn bộ trong phạm vi đã được Plan chốt.

Implement không chỉ bao gồm việc viết production code.

Tùy theo project và risk thực tế, Implement có thể bao gồm:

- thay đổi production code
- viết và thực thi unit test, integration test hoặc end-to-end test phù hợp
- chạy lint, typecheck, build, migration check hoặc project-native validation
- kiểm tra runtime behavior có thể tự động hóa
- phát hiện và sửa các lỗi mà AI có thể tìm thấy bằng code inspection, test, tool hoặc runtime evidence
- đối chiếu implementation với spec ban đầu

Không phải project nào cũng cần mọi loại test hoặc check.

AI phải chọn tập kiểm tra phù hợp với project và behavior đã thay đổi thay vì chạy test chỉ để hoàn thành ceremony.

AI được tự chọn cách chia nhỏ công việc, dùng tool, skill hoặc agent và lặp lại implementation khi vẫn tạo ra tiến triển hữu ích.

Human không nên phải tham gia vào phase Implement trừ khi AI gặp blocker cần authority mới hoặc phát hiện một quyết định quan trọng chưa được chốt trong Plan.

Mọi kiểm chứng mà AI có thể tự chạy đều thuộc Implement, kể cả khi workflow tổ chức chúng thành step riêng sau khi code đã viết xong.

Validate chỉ bắt đầu khi output đã được bàn giao cho human.

Implement hoàn thành khi AI đã thực hiện scope, tự kiểm tra kết quả theo khả năng của project và chuẩn bị output đủ rõ để human review.

### Validate

Validate là phase do human chịu trách nhiệm chính.

Human review output và kết quả mà AI đã bàn giao để quyết định công việc có đúng yêu cầu và đạt kỳ vọng hay chưa.

Human có thể dựa trên spec, implementation summary, test evidence, runtime evidence, checklist hoặc trực tiếp trải nghiệm sản phẩm tùy theo project.

AI phải chuẩn bị evidence và summary đủ rõ để human không phải đọc lại toàn bộ quá trình implementation.

Evidence do AI cung cấp hỗ trợ quyết định của human nhưng không thay thế quyền final acceptance của human.

Human tập trung vào những phần AI khó tự xác nhận đầy đủ như product fit, UX quality, business correctness, risk acceptance và mức độ đáp ứng intent ban đầu.

Implement được thực hiện càng đầy đủ thì Validate càng nhanh và human càng ít phải yêu cầu sửa hoặc chạy lại.

Khi human không chấp nhận kết quả, công việc quay lại đúng phase đã sinh ra vấn đề.

Sai sót nằm trong phạm vi đã chốt thì quay lại Implement.

Intent sai, scope thay đổi hoặc xuất hiện một quyết định quan trọng chưa được chốt thì quay lại Plan.

Approved intent chỉ thay đổi bằng quyết định của human, không bằng việc implementation đã đi theo hướng khác.

## AI Autonomy Và Human Control

AI được tự chủ trong những boundary có thể phục hồi và kiểm chứng.

Hành động có blast radius nhỏ, dễ rollback và có validation đáng tin cậy nên cần ít human intervention hơn.

Hành động destructive, khó rollback hoặc mang quyết định product, business hay security quan trọng phải có human authority rõ ràng.

Human sở hữu:

- product intent và material decisions
- thay đổi scope có ý nghĩa
- risk acceptance và high-impact authorization
- final acceptance trong phase Validate
- subjective judgment không thể chứng minh đầy đủ bằng tool evidence

AI sở hữu:

- codebase discovery và technical decisions không thay đổi approved behavior
- lựa chọn tool, skill, agent và internal task breakdown
- toàn bộ implementation trong phạm vi đã chốt
- testing, debugging và self-verification phù hợp với project
- trình bày output, evidence, uncertainty và phần chưa hoàn thành

Human không nên phải approve từng technical step.

Không được trình bày một validation mà chỉ human mới xác nhận được như thể AI đã kiểm chứng.

Human-facing output phải ưu tiên decision, result, evidence và unresolved issue.

Internal state, tool logs và detailed artifacts chỉ nên được mở rộng khi có failure, audit need hoặc explicit human request.

Artifact phục vụ quyết định của human phải đủ ngắn để review nhanh.

Artifact phục vụ agent execution được phép chi tiết khi có step downstream thực sự tiêu thụ chi tiết đó.

Artifact chi tiết phải mở đầu bằng contract hoặc summary ngắn và không lặp lại nội dung không tạo thêm giá trị thực thi.

Không phải mọi rule đều cần runtime enforcement.

- Dùng guidance để định hướng behavior của AI.
- Dùng hard enforcement cho permission, destructive action, security, concurrency và resource boundary.
- Dùng evidence để giúp AI tự kiểm tra và giúp human đánh giá kết quả.

Workflow ưu tiên evidence-backed delivery hơn process-compliance completion.

Việc agent đi qua đầy đủ step không chứng minh outcome đúng.

## Nguyên Tắc Phát Triển Workflow

Workflow phải giữ đơn giản và thích ứng được theo project.

Mỗi step phải cho human thấy rõ nó nhận input gì, tạo ra output gì và cho phép quyết định gì tiếp theo.

Chỉ thêm pattern, gate, artifact, state hoặc orchestration mechanism khi nó giải quyết một failure, risk hoặc human cost cụ thể.

Không thêm abstraction chỉ vì nó tồn tại trong workflow engine, distributed system hoặc multi-agent literature.

Không thêm artifact nếu không có reader hoặc downstream consumer rõ ràng.

Không thêm gate nếu human không thực sự đưa ra một decision tại gate đó.

Không thêm state nếu state đó không thay đổi execution, visibility hoặc khả năng kiểm soát.

Không mô hình hóa toàn bộ reasoning của agent.

Không dùng prompt complexity để tạo cảm giác kiểm soát mà runtime hoặc evidence không thể xác nhận.

### Thêm Và Thử Nghiệm Cơ Chế Mới

Trước khi thêm một cơ chế mới, phải trả lời được:

1. Vấn đề thực tế nào đang xảy ra?
2. Vì sao giải pháp đơn giản hơn chưa đủ?
3. Cơ chế mới giảm risk hoặc human work như thế nào?
4. Ai hoặc step nào sử dụng output mới?
5. Làm sao biết cơ chế mới đã cải thiện outcome?

Nếu không có câu trả lời cụ thể, mặc định không thêm cơ chế mới.

Không đưa một pattern, command, skill, artifact hoặc agent vào workflow chuẩn trước khi nó chứng minh được giá trị qua công việc thật lặp lại nhiều lần.

Một experiment được phép nằm tạm trong workflow khi:

- nó được gắn nhãn experimental
- hypothesis và success criteria của nó nhìn thấy được
- hành vi trước đó có thể khôi phục hoặc so sánh được
- nó không được mô tả là proven khi chưa có runtime evidence

Ưu tiên thiết kế portable giữa các coding-agent runtime khi khả thi.

### Giới Hạn Của Project

Project được quyền thay đổi step, skill, tool, agent, parallelism, test strategy, artifact và human gate theo nhu cầu thực tế.

Project không được làm mất các nguyên tắc sau:

- Plan làm rõ intent và chốt các quyết định quan trọng trước implementation
- Implement do AI thực hiện và bao gồm testing, debugging và self-verification phù hợp
- AI không bàn giao chỉ dựa trên confidence hoặc việc code đã được viết
- Validate để human review output và quyết định kết quả có đúng yêu cầu hay không
- phần chưa được chứng minh phải được hiển thị rõ

Khi các mục tiêu xung đột, ưu tiên theo thứ tự:

1. Correctness và safety tại boundary có hậu quả cao.
2. Kết quả có evidence và giới hạn rõ ràng.
3. AI autonomy và khả năng hoàn thành công việc.
4. Giảm human intervention và cognitive load.
5. Simplicity và khả năng thay đổi workflow.
6. Consistency về ceremony hoặc hình thức artifact.

Simplicity không được dùng để bỏ qua correctness hoặc safety.

Correctness cũng không được dùng làm lý do để thêm ceremony không tạo ra control hoặc evidence thực sự.

## Governance

Tài liệu này là nguồn định hướng cao nhất cho kiến trúc workflow trong repository.

Đây là tài liệu nguyên tắc duy nhất cho việc xây dựng workflow.

Không tạo thêm một tài liệu nguyên tắc song song; nguyên tắc mới phải vào thẳng tài liệu này.

Mọi standard, runtime contract, workflow config và skill trong repository đều nằm dưới hiến pháp và không được đi ngược các nguyên tắc ở đây.

Tài liệu chi tiết hơn quyết định cách thực hiện; hiến pháp quyết định điều gì được phép tồn tại.

Một tài liệu là tài liệu chi tiết khi nó mô tả step, command, contract, config hoặc artifact cụ thể của một workflow.

Một phát biểu áp cho mọi workflow bất kể step nào đang chạy thì thuộc về hiến pháp.

Khi một tài liệu chi tiết xung đột với hiến pháp, phải làm rõ conflict thay vì tự động suy diễn hoặc hợp nhất hai hướng.

Hiến pháp chỉ thay đổi bằng quyết định rõ ràng của human.

Hiến pháp chứa hai loại nội dung với ngưỡng thay đổi khác nhau.

Nguyên tắc nền tảng gồm North Star, ý nghĩa ba phase và ranh giới authority giữa human và AI; chỉ cập nhật khi triết lý thực sự thay đổi.

Quy ước bắt buộc là các ràng buộc chung áp cho mọi workflow; được tinh chỉnh khi có bằng chứng thực tế cho thấy quy ước hiện tại gây sai sót hoặc chi phí không cần thiết.

Không cập nhật hiến pháp để phản ánh một implementation detail hoặc exception của riêng project.
