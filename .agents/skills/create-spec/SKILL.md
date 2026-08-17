---
name: create-spec
description: Use when the user asks to create a detailed implementation specification or when an approved design decision manifest must be converted into the durable source of truth for implementation and verification. Creates docs/ai/specs/{feature}.md with approved intent, codebase evidence, concrete technical design, implementation mapping, acceptance criteria, and verification strategy.
---

# Create Spec

Create the detailed AI-facing implementation and verification contract for an approved feature slice.

## Process

1. Determine the authority source.
   - Under `feature-standard`, require the orchestrator-provided `design_decisions_path` and read the matching `design_path` from that manifest.
   - Validate the decision manifest with the validator bundled in the matching runtime's `design-spec` skill when available.
   - In standalone use, gather material human decisions directly in chat before writing.
   - A standalone invocation must not pretend that chat assumptions were approved through `design-spec`.
2. Inspect the codebase deeply enough to produce an implementation-ready design.
   - Confirm current behavior, affected files and symbols, existing boundaries, data and state flow, interfaces, dependencies, validation, failure behavior, security constraints, and verification capabilities.
   - Start from the manifest `evidence` entries, which were already verified during design, and deepen them instead of rediscovering the same ground.
   - Record concrete evidence paths in the spec.
   - Find the nearest existing precedent: one or two features, files, or modules closest in shape to what will be built.
   - Record what the implementation must mirror from that precedent, and any deliberate deviation with its reason.
   - When no precedent exists, say so explicitly and state why this is a new kind of work; do not leave the question unanswered.
   - Do not copy repository-wide context that does not affect this slice.
3. Check the approved slice before writing.
   - Classify important assumptions as confirmed, inferred but safe, needs confirmation, or agent-chosen technical detail.
   - Preserve the semantics of `output_preview`: its summary, the screen, exchange, or shape block, the observable states, and the flow branches.
   - Translate every answered decision and every `business_rules` statement into behavior requirements and acceptance criteria without changing their meaning.
   - Treat `implementation_constraints` as binding and record them where the executor will see them.
   - Treat `ai_discretion` as the agent's own latitude; never restate it as a human commitment.
   - Map the approved output preview into the existing execution contract, behavior requirements, interface or state changes, and acceptance criteria; do not create a second product-design section.
   - A human-approved decision may be elaborated technically but must not be changed in meaning.
   - If implementation discovery exposes a conflict with an approved decision, stop and escalate it instead of silently rewriting the behavior.
4. Choose exactly one result:
   - `write-spec`
   - `ask-human`
   - `split-slices`
   - `run-spike`
   - `escalate-conflict`
   - If the result is not `write-spec`, stop without creating a spec file
5. If the result is `ask-human`:
   - Ask only the missing questions (max 5), batch them into one block, then wait
   - After answers arrive, re-evaluate the request with the new information
6. If the result is `split-slices`:
   - Do not write an epic spec
   - Propose the smallest valuable slice and stop
7. If the result is `run-spike`:
   - Explain what feasibility or architecture question must be answered first
   - Stop without writing the spec
8. If the result is `escalate-conflict`:
   - State the specific conflict with the codebase or business rules
   - Stop without writing the spec
9. If the result is `write-spec`, classify implementation depth.
   - Score +1 for each true condition:
     - More than 1 primary user flow
     - Persistence across session or restart
     - Validation rules or hard limits
     - Fallback or degraded behavior
     - Empty or error states that must be defined
     - Reset or default-selection behavior
     - Migration or backward compatibility expectations
     - More than 2 affected surfaces
     - Quota, permission, or gating behavior
     - External integration or asynchronous boundary
     - Security-sensitive or destructive behavior
     - More than 8 expected acceptance criteria
   - Tier by score:
     - 0-2 = Lite
     - 3-5 = Standard
     - 6+ = Extended
   - Migration, fallback, quota, security-sensitive behavior, or more than 3 distinct behavior areas cannot be Lite.
   - Tier controls required analysis depth, not file length or acceptance-criteria count.
10. Write `docs/ai/specs/{feature-name}.md`.
    - `{feature-name}` must be the approved kebab-case feature slug when available.
    - Include every section that is relevant to execution or verification.
    - Write `Không áp dụng` with a short reason for a required risk section that is not relevant.
    - Do not omit detail only to keep the file short.
    - Split only when the request contains independently valuable outcomes or cannot remain one executable slice.

## Spec Format

Generated specs must use the Vietnamese headings below.
Do not keep the English section names from older examples.
Keep only code symbols, file paths, API names, schema names, JSON keys, decision IDs, AC IDs, and literal enum values in English.

```markdown
## Cấp Độ
[Lite | Standard | Extended]

## Hợp Đồng Thực Thi
### Mục Tiêu
- ...

### Nguồn Quyết Định Đã Duyệt
- Manifest quyết định: `docs/ai/design-decisions/{feature}.json` — revision [n], `approval_meaning: direction-approved`
- Bản xem trước đầu ra: [kind, summary, trạng thái quan sát được, và các nhánh của flow đã duyệt]
- D-001: [câu hỏi] → [phương án human đã chọn]
- BR-001: [quy tắc nghiệp vụ đã duyệt]
- IC-001: [ràng buộc bắt buộc tuân thủ]
- Ràng buộc human thêm khi duyệt: [từ `constraints`, hoặc `Không có`]

### Bắt Buộc Xảy Ra
- ...

### Không Được Xảy Ra
- ...

## Vấn Đề
...

## Phạm Vi
- ...

## Ngoài Phạm Vi
- ...

## Quyết Định Thiết Kế Đã Duyệt
- D-001: [quyết định và lý do]

## Kiểm Tra Giả Định
### Từ Design Đã Duyệt
- A-001: [giả định] — nếu sai: [tác động]
- R-001: [rủi ro] — hệ quả: [hệ quả]

### Đã Xác Nhận
- ...

### Suy Luận An Toàn
- ...

### Cần Xác Nhận
- Không có câu hỏi sản phẩm nào đang blocking.

### Chi Tiết Kỹ Thuật Do Agent Chọn
- [gồm cả các mục `ai_discretion` đã được duyệt là agent tự quyết]
- ...

## Bằng Chứng Hệ Thống Hiện Tại
- `path/to/file`: [hành vi hiện tại, symbol, hoặc ràng buộc]

## Tiền Lệ Trong Codebase
- Theo mẫu: `path/to/feature` — [vì sao đây là cái gần nhất về hình dạng]
- Phải giống: [cấu trúc file, hình dạng output, quy ước đặt tên, cách xử lý lỗi]
- Cố ý khác: [điểm khác và lý do]

## Yêu Cầu Hành Vi
### {Khu Vực Hành Vi}
- ...

## Thay Đổi Trạng Thái / Dữ Liệu / Giao Diện
- ...

## Thiết Kế Kỹ Thuật Chi Tiết
### {Thành Phần hoặc Ranh Giới}
- Trách nhiệm: ...
- Đầu vào và đầu ra: ...
- Chuyển trạng thái hoặc luồng dữ liệu: ...
- Hành vi lỗi: ...

## Bản Đồ Thay Đổi Theo File
| Bề mặt | Thay đổi dự kiến | Lý do | Quyết định / AC |
|---|---|---|---|
| `path/to/file` | ... | ... | D-001 / AC1 |

## Kiểm Tra Hợp Lệ / Lỗi / Trường Hợp Biên
- ...

## Cân Nhắc Bảo Mật / Phân Quyền
- ...

## Tương Thích / Di Chuyển Dữ Liệu
- ...

## Trình Tự Triển Khai
1. ...

## Tiêu Chí Chấp Nhận
### {Khu Vực Hành Vi}
- [ ] AC1: ...

## Ma Trận Xác Minh
| AC | Chiến lược bằng chứng | Bề mặt chính |
|---|---|---|
| AC1 | Runtime / test tập trung / kiểm tra thủ công | ... |

## Câu Hỏi Mở
- Không có câu hỏi sản phẩm nào đang blocking.

## Nhật Ký Quyết Định
- ...
```

## Rules

- Write assistant responses, questions, generated spec prose, Markdown headings, table headers, and recurring labels in Vietnamese.
- Keep only code symbols, file paths, API names, schema names, JSON keys, decision IDs, AC IDs, literal enum values, and command names in English.
- Treat approved design decisions as human authority and never reinterpret them silently.
- Treat an approved `output_preview` as human authority for the expected surfaces, observable states, and high-level operating flow.
- Treat `business_rules` as approved product behavior even though the manifest records their source as `agent-proposed-not-objected`.
- Carry every `D-xxx`, `BR-xxx`, and `IC-xxx` id into the spec so verification can trace each one back to what the human saw.
- Do not reopen, re-ask, or turn into a requirement anything the manifest lists under `ai_discretion`; it is settled latitude, not an open question.
- Carry `assumptions` and `risks` across with their ids and their stated impact, because they are what an executor must watch while implementing.
- `approval_meaning: direction-approved` means the human chose the decisions and accepted the rest as shown; it does not license changing a statement they never objected to.
- Under `feature-standard`, fail closed when `design_decisions_path` is missing, invalid, or does not match the design plan checksum.
- Cite the nearest codebase precedent and name what the implementation must mirror from it; when none exists, state that explicitly with a reason instead of leaving the section empty.
- Prefer mirroring an existing precedent over introducing a new structure, output shape, naming scheme, or error-handling style; record every deliberate deviation with its reason.
- Do not treat a precedent marked as intentional debt in `docs/ai/architecture/decisions.md` as a pattern to mirror.
- Include concrete file paths, symbols, interfaces, schema changes, migration mechanisms, storage keys, implementation order, and test surfaces when they are grounded in inspected code and useful to the executor.
- Do not invent low-level detail when codebase evidence is insufficient.
- Label agent-chosen technical decisions separately from human-approved behavior.
- Do not add product behavior, exclusions, thresholds, or visible defaults that are absent from the approved decisions or explicit standalone human input.
- Do not use line count or acceptance-criteria count as a completeness target or failure rule.
- Prefer information density and traceability over repetition.
- Keep one executable slice; split by independent outcome or dependency boundary, not document length.
- Every acceptance criterion must be observable or have a concrete verification strategy.
- Map every acceptance criterion to implementation and evidence surfaces.
- Blocking product questions are incompatible with `write-spec`; use `ask-human` instead.
- A spec is valid only when an executor can implement and a verifier can evaluate it without inventing behavior.

## Allowed Outcomes

The command does not always produce a spec file.

Valid outcomes are:
- `Spec written`
- `Questions needed`
- `Slice proposed`
- `Spike required`
- `Conflict escalated`
- `Blocked`

## Orchestrator Contract

When this skill is run under `/orchestrator`, append exactly one HTML comment as the final output line:

- Spec written:
  `<!-- orchestrator: outcome=continue provides=spec_path spec_path=docs/ai/specs/{feature-name}.md -->`
- Questions needed:
  `<!-- orchestrator: outcome=stop-ask-human -->`
- Slice proposed:
  `<!-- orchestrator: outcome=stop-split-slices -->`
- Spike required:
  `<!-- orchestrator: outcome=stop-run-spike -->`
- Conflict escalated:
  `<!-- orchestrator: outcome=stop-escalate-conflict -->`
- Missing or invalid required authority artifact:
  `<!-- orchestrator: outcome=stop-blocked -->`

Rules:
- Emit the comment only after the main human-readable response is complete
- `spec_path` must match the file actually written
- Under `feature-standard`, emit `continue` only after the design decision manifest passes validation and every approved decision is represented in the spec
- If this skill runs standalone, the comment is optional

## Self-Check Before Writing The File

- Is the authority source explicit and valid?
- Does every approved `D-xxx` decision appear without semantic drift?
- Does every approved output-preview item appear in the relevant execution, behavior, interface, acceptance, and verification sections without semantic drift?
- Does every `BR-xxx` appear as behavior or acceptance criteria, and every `IC-xxx` where the executor will actually see it?
- Is every `ai_discretion` item left as agent latitude rather than restated as a human commitment?
- Are goal, scope, must-happen, and must-not-happen behavior easy to locate?
- Is every current-system claim backed by a concrete codebase path or direct evidence?
- Does the spec name a concrete precedent to mirror, or state explicitly that none exists and why?
- Are affected files, symbols, interfaces, state, data, validation, and failure paths detailed when relevant?
- Are security, permission, compatibility, migration, and rollback concerns addressed or marked not applicable with a reason?
- Is the implementation sequence feasible and dependency-aware?
- Does every acceptance criterion map to implementation and verification surfaces?
- Are agent-chosen technical details clearly separated from human decisions?
- Are there zero blocking product questions in a spec that will return `continue`?
- Can a weaker executor implement the slice without rediscovering the design or inventing behavior?
