---
name: client-interview-prep
description: "Prepare a developer for a client-facing interview or project pitch by turning keywords into high-level visual topic briefs and, when client context is supplied, researching the company on the web, extracting evidence from the candidate's CV in PDF, DOCX, Markdown, or text format, analyzing job descriptions, and producing a tailored learning map, fit-gap analysis, likely questions, and truthful talk tracks. Use for requests such as học nhanh topic để phỏng vấn, chuẩn bị phỏng vấn khách hàng, nghiên cứu công ty khách, đọc CV để học topic phù hợp, client interview preparation, or tailor my project pitch to this company."
---

# Client Interview Prep

Prepare the candidate to discuss technology in the context of the client's domain, products, project needs, and hiring signals.
Optimize for a client-facing project interview rather than a generic HR interview or an exhaustive technical study guide.

## Select The Mode

Use `Rapid topic mode` when the user supplies keywords and wants a high-level explanation without client, company, CV, job, or project context.
Do not require company or CV inputs in this mode.

Use `Client-tailored mode` when the user supplies or mentions a client, company, CV, job description, role, or project context.
Use web research and CV evidence in this mode.

When the user explicitly requests client-tailored preparation but omits required inputs, ask for all missing items in one concise batch.
Do not silently fall back to generic advice when the user expects company-specific analysis.

## Collect Inputs For Client-Tailored Mode

Require the following inputs.

- Obtain the client company name or official website.
- Obtain the candidate CV path or attachment in `.pdf`, `.docx`, `.md`, or `.txt` format.
- Obtain the initial keywords or topics the candidate expects to discuss.

Accept the following optional inputs when available.

- Use the job description, careers page, or recruitment link.
- Use the project brief, expected role, seniority, interview language, interview date, and interview format.
- Use any known client questions or feedback from earlier interview rounds.

Ask for all missing required inputs in one concise batch.
Do not ask the user for information that can be discovered reliably from the supplied company identity or public sources.

## Extract CV Evidence

1. Read the supplied CV without modifying it.
2. Extract roles, responsibilities, projects, domains, technologies, delivery activities, scale, and stated outcomes.
3. Preserve the difference between direct experience, adjacent experience, listed knowledge, and unsupported inference.
4. Use only evidence present in the CV or explicitly supplied by the user when describing the candidate's experience.
5. Attempt OCR for a scanned PDF when an available tool supports it.
6. State the extraction limitation and request an OCR or text version only after available extraction methods fail.

Do not repeat phone numbers, email addresses, home addresses, identification numbers, or other irrelevant personal data in the output.
Do not include personal CV data in web search queries.
Do not search for additional personal information about the candidate unless the user explicitly requests it.

## Research The Client

Use current web research rather than relying only on model memory.
Record the research date and cite the source for company-specific, recruitment, and time-sensitive claims.

Research the following areas when relevant.

- Identify the company's domain, markets, customer segments, and business model at a high level.
- Identify major products, services, user journeys, and operational workflows related to the candidate's likely project.
- Identify public technology initiatives, engineering content, AI or digital transformation programs, and relevant recent announcements.
- Find current or recent software developer, engineering, architecture, data, AI, security, or adjacent job postings.
- Extract recurring skills, responsibilities, seniority expectations, delivery practices, and domain knowledge from those postings.
- Identify regulatory, privacy, reliability, or security concerns that materially affect the company's domain.

Prioritize sources in this order.

1. Use the official company website, product documentation, careers site, engineering blog, and official reports first.
2. Use official or verified company profiles and job posts second.
3. Use reputable industry publications, regulatory sources, and established recruitment platforms for additional context.
4. Use aggregators, cached pages, or informal sources only as secondary signals and label their limitations.

Mark each recruitment item as current, expired, or status unclear when that status can be determined.
Treat one job posting as a role signal, not proof of the entire company's technology stack or architecture.
Separate verified facts, recurring signals, and working assumptions.
Surface meaningful conflicts between sources instead of silently merging them.
If web access is unavailable, state that current company research could not be completed and do not simulate search results.

## Model The Client Need

Build a concise model connecting the following elements.

```text
Company domain -> products and workflows -> likely project needs
               -> developer hiring signals -> interview expectations
```

Focus on what a client typically evaluates in a project interview.

- Show understanding of the client's business problem and user impact.
- Connect technical decisions to delivery value, risk, cost, reliability, and security.
- Communicate trade-offs clearly without drowning the client in implementation detail.
- Demonstrate relevant experience with concrete evidence.
- Handle gaps honestly and show a credible learning or delivery approach.

Do not assume the interview is for a specific project when the project scope is not public or supplied.
Label plausible project relevance as a hypothesis.

## Build The Fit Map

Create a table with these columns.

| Client need or signal | CV evidence | Fit | Interview handling |
|---|---|---|---|
| A sourced need or hiring signal | A concrete CV claim or `No direct evidence` | Strong, adjacent, or gap | Emphasize, bridge honestly, learn, or ask |

Apply these fit labels consistently.

- Use `Strong` only when the CV contains direct, relevant evidence.
- Use `Adjacent` when the CV demonstrates a transferable pattern but not the exact domain or technology.
- Use `Gap` when neither the CV nor user-provided context demonstrates relevant experience.

Never manufacture a project story, metric, responsibility, technology, or level of ownership.
For an adjacent fit, explain the transferable principle and the boundary of the candidate's experience.
For a gap, propose a concise learning target or an honest question to ask the client.

## Prioritize The Learning Topics

Treat the user's keywords as starting points rather than a fixed order.
Reorder them using client evidence and CV fit, but do not silently omit a supplied keyword.

Group topics into three levels.

- Put concepts required to understand the client's domain, product, or likely project in `Must know`.
- Put concepts likely to strengthen technical discussion or demonstrate role fit in `Should know`.
- Put speculative, low-signal, or deep implementation topics in `Optional`.

For each `Must know` topic, provide the following high-level briefing.

1. Define the topic in one sentence.
2. Explain why this client or project may care about it.
3. Show the main components or workflow.
4. Define only the essential vocabulary needed to discuss it.
5. Explain the most important trade-off or risk.
6. Connect it to direct or adjacent CV evidence.
7. Provide one likely client question and a truthful answer direction.

Keep the default explanation high level.
Do not include code, detailed configuration, algorithms, or exhaustive edge cases unless the user requests a deep dive.

## Add A Visual Map

Include one compact Mermaid diagram when it improves understanding.
Use an ASCII diagram when Mermaid is unavailable or the user requests plain text.

Prefer one of these views.

- Show `domain -> product workflow -> project need -> topic -> CV evidence` for preparation strategy.
- Show a system or business workflow when the topic is architectural.
- Show relationships between client signals, candidate strengths, gaps, and learning priorities.

Keep the visual to roughly six to ten nodes with short labels in the user's language.
Do not invent dependencies, architecture, or project scope to complete the diagram.
Label inferred relationships as hypotheses when needed.

## Produce The Rapid Topic Brief

Use this output only in `Rapid topic mode`.
Keep it readable in about five minutes, normally 400 to 700 words.

Use this output order.

1. Define the combined topic in one sentence.
2. Explain why it matters using two to four bullets.
3. Build a big-picture mental model from three to six main parts or ideas.
4. Include one simple Mermaid or ASCII visual when it improves understanding.
5. Define only the essential keywords in plain language.
6. Summarize the main trade-offs, risks, and boundaries.
7. Write a natural 30-to-60-second interview or pitch talk track.
8. List three to five likely follow-up questions with one-sentence answer directions.
9. End with two to four optional deep-dive branches without expanding them.

Group related keywords into one coherent model instead of writing an isolated mini-essay for each keyword.
Do not perform broad company research in this mode unless the user asks or the topic depends on current facts.
Do not include code, commands, formulas, configuration, low-level architecture, or historical detail unless the user requests it.

## Produce The Preparation Brief

Write the output in the user's language unless another language is requested.
Start with a section that can be read in about five minutes, then provide focused supporting material.

Use this output order.

1. Present the client and interview context, including explicit assumptions.
2. Summarize the company domain, products, relevant workflows, and current hiring signals with citations.
3. Show the visual map.
4. Present the CV-to-client fit map.
5. Present the prioritized learning map.
6. Brief the `Must know` topics at a high level.
7. Write a natural 60-to-90-second client-facing introduction or project talk track.
8. List likely client questions with answer directions grounded in the CV.
9. List three to five high-value questions the candidate should ask the client.
10. Provide a preparation sequence ordered by interview impact and available time.
11. End with a source list containing page titles, URLs, source type, and access date.

Keep sourced company facts separate from recommendations and assumptions.
Prefer a small number of high-quality sources over a long list of weak results.

## Write Truthful Talk Tracks

Use the candidate's actual experience as the foundation.
Frame adjacent experience with language such as `The closest experience in my background is...` or its equivalent in the response language.
Frame gaps with a learning approach, validation plan, or clarifying question rather than false confidence.
Avoid claiming that the candidate designed, led, scaled, secured, or operated a system unless the CV or user explicitly confirms it.

## Completion Criteria

- In `Rapid topic mode`, produce one coherent high-level brief without demanding client inputs.
- In `Client-tailored mode`, base company and hiring claims on cited current research.
- In `Client-tailored mode`, connect every high-priority learning topic to a client signal, project context, or material CV gap.
- In `Client-tailored mode`, distinguish direct CV evidence from adjacent experience and gaps.
- Provide at least one visual overview when it adds explanatory value.
- Keep default topic explanations high level and interview-oriented.
- Produce talk tracks the candidate can use without misrepresenting their experience.

## Example Invocation

`Nghiên cứu công ty Example Insurance từ https://example.com và các tin tuyển developer gần đây. Đọc CV của tôi tại ./cv.pdf. Tôi cần chuẩn bị client interview về AI model, business workflow và security cho dự án chatbot.`

`Giúp tôi hiểu nhanh AI model, business workflow và security để chuẩn bị pitching một dự án chatbot.`
