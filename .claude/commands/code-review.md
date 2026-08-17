---
name: code-review
description: Performs a local code review strictly for standards conformance.
---

You are helping me perform a local code review **before** I push changes.

## Workflow Alignment

- Provide brief status updates (1–3 sentences) before/after important actions.
- For medium/large reviews, create todos (≤14 words, verb-led). Keep only one `in_progress` item.
- Update todos immediately after progress; mark completed upon finish.
- Provide a high-signal summary at completion highlighting key findings and impact.

---

## Step 1: Determine Review Scope

**Ask user for scope:**

Use AskUserQuestion to determine what files to review:
```
AskUserQuestion(questions=[{
  question: "Which files would you like to review?",
  header: "Review Scope",
  options: [
    { label: "Review against a base branch (PR Style)", description: "Compare current branch against main/master or specified base branch" },
    { label: "Review uncommitted changes (Working directory)", description: "Review staged and unstaged changes in working directory" }
  ],
  multiSelect: false
}])
```

**Based on selection:**

1. **PR Style (against base branch):**
   - Ask for base branch using AskUserQuestion:
   ```
   AskUserQuestion(questions=[{
     question: "Which base branch should we compare against?",
     header: "Base Branch",
     options: [
       { label: "main", description: "Compare against main branch (Recommended)" },
       { label: "develop", description: "Compare against develop branch" },
       { label: "master", description: "Compare against master branch" },
       { label: "Other", description: "Enter a custom branch name" }
     ],
     multiSelect: false
   }])
   ```
   - If user selects "Other", prompt them to enter the branch name
   - `Bash(command="git diff <base-branch>...HEAD --name-only")` to get changed files
   - `Bash(command="git diff <base-branch>...HEAD")` to get full diff

2. **Working directory changes:**
   - `Bash(command="git diff --cached --name-only")` to get staged file list
   - `Bash(command="git diff --cached")` to get staged diff content
   - Use **only the staged diff** as review scope — do NOT read files directly (unstaged changes are excluded)

**Error handling:**
- Git not available: Ask user for file paths directly
- No files to review: Notify user and exit
- Base branch not found: Ask user to specify correct branch name

---

## Step 2: Execute Both Reviews

Run **both** review types automatically. Results are reported **independently** in separate sections.

---

## Part A: Standards Conformance

> **Mode: STRICT** - Only report violations explicitly defined in standards docs. Do NOT infer or suggest beyond what rules state.

### A1. Load Standards

**Tools:**
- Read(file_path="docs/ai/project/CODE_CONVENTIONS.md")
- Read(file_path="docs/ai/project/PROJECT_STRUCTURE.md")

**Error handling:**
- Standards docs not found: Notify user, cannot proceed with Standards Conformance review

### A2. Scan for Violations

**Tool:** Task(
  subagent_type='Explore',
  thoroughness='medium',
  prompt="Scan the provided diff for violations against CODE_CONVENTIONS and PROJECT_STRUCTURE ONLY.

    IMPORTANT: Review ONLY the diff content provided — do NOT read files directly. This ensures only staged changes are reviewed, not unstaged modifications.

    STRICT RULES:
    - Report ONLY violations that are EXPLICITLY stated in the standards docs
    - Do NOT infer additional rules
    - Do NOT suggest improvements beyond what standards require
    - Do NOT provide design opinions

    Check for:
    - Naming conventions (variables, functions, classes, constants)
    - Import order and grouping
    - Folder structure and module boundaries
    - Test placement and naming
    - File naming patterns
    - Export patterns

    Return violations with file:line, exact rule violated (quote from docs), and brief description."
)

**Fallback:** If Explore agent unavailable, manually Read each file and check against standards.

### A3. Standards Report

**Output format:**

```
## Standards Conformance Report

### Violations Found: X

#### [File: path/to/file.ext]

1. **Line XX** — Violates: "[exact rule from CODE_CONVENTIONS or PROJECT_STRUCTURE]"
   - Found: `actual_code`
   - Expected: `expected_code`

2. **Line YY** — Violates: "[exact rule]"
   ...
```

**Severity:**
- **Blocking**: Architectural violations, module boundary breaches
- **Important**: Naming conventions, import order
- **Minor**: Formatting, spacing (if explicitly in standards)

---

## Part B: Quality Review

> **Mode: REASONING** - Agent uses judgment to identify issues and provide recommendations.

### Guidelines for Quality Review

**Good Feedback is:**
- Specific and actionable
- Educational, not judgmental
- Focused on the code, not the person
- Balanced (praise good work too)
- Prioritized (critical vs nice-to-have)

**Examples:**

❌ Bad: "This is wrong."
✅ Good: "This could cause a race condition when multiple users access simultaneously. Consider using a mutex here."

❌ Bad: "Why didn't you use X pattern?"
✅ Good: "Have you considered the Repository pattern? It would make this easier to test."

❌ Bad: "Rename this variable."
✅ Good: "[nit] Consider `userCount` instead of `uc` for clarity. Not blocking."

### B1. Review Criteria

**What to Review:**
- Logic correctness and edge cases
- Security vulnerabilities
- Performance implications
- Error handling
- Code readability and maintainability
- API design and naming
- Test coverage gaps

**What NOT to Review (leave to linters):**
- Code formatting
- Import organization
- Simple typos

### B2. Perform Quality Review

**Tool:** Task(
  subagent_type='Explore',
  thoroughness='medium',
  prompt="Review the provided diff for quality issues using your reasoning.

    IMPORTANT: Review ONLY the diff content provided — do NOT read files directly. This ensures only staged changes are reviewed, not unstaged modifications.

    REASONING MODE - Use judgment to identify:
    - Logic bugs and edge cases
    - Security vulnerabilities (injection, auth issues, data exposure)
    - Performance problems (N+1 queries, memory leaks, unnecessary loops)
    - Poor error handling (silent failures, swallowed exceptions)
    - Code smells and maintainability issues
    - Missing or weak test coverage

    For each issue:
    - Explain WHY it's a problem
    - Provide actionable recommendation
    - Mark severity (Critical/Important/Nit)

    Also note GOOD patterns worth praising."
)

### B3. Quality Report

**Output format:**

```
## Quality Review Report

### Overview
- 🔴 Critical: X
- 🟡 Important: Y
- 💬 Nits: Z

### What's Good
- [Positive observations about the code]

### Issues Found

#### Critical

**1. path/to/file.ext:45**
- **Category**: Security / Bug / Performance
- **Issue**: [Clear description of the problem]
- **Why it matters**: [Explanation]
- **Recommendation**: [Specific fix]

#### Important

**2. path/to/file.ext:78**
...

#### Nits

**3. path/to/file.ext:15**
- [nit] [Minor suggestion - not blocking]
```

---

## Step 3: Final Summary

Present results in **two separate sections**:

### Section 1: Standards Conformance Checklist

```
## Standards Conformance
- [ ] Naming conventions: [pass/X violations]
- [ ] Import order: [pass/X violations]
- [ ] Folder structure: [pass/X violations]
- [ ] Module boundaries: [pass/X violations]
```

### Section 2: Quality Review Checklist

```
## Quality Review
- [ ] Logic correctness verified
- [ ] Security vulnerabilities checked
- [ ] Error handling adequate
- [ ] Performance considerations reviewed
- [ ] Code is maintainable
```

---

## Notes

### Report Example (Both Reviews)

```markdown
# Code Review Summary

---

## Part 1: Standards Conformance

### Violations: 3

1. **src/utils/helper.ts:12** — Violates: "Functions must use camelCase"
   - Found: `get_user_data`
   - Expected: `getUserData`

2. **src/services/auth.ts:1** — Violates: "Imports must be grouped: external, internal, relative"
   - Found: Mixed import order
   - Expected: Group by category with blank lines

3. **tests/user.spec.ts** — Violates: "Tests must be colocated with source"
   - Found: `tests/user.spec.ts`
   - Expected: `src/services/user.spec.ts`

---

## Part 2: Quality Review

### Overview
- 🔴 Critical: 1
- 🟡 Important: 2
- 💬 Nits: 1

### What's Good
- Clean separation of concerns in service layer
- Good TypeScript usage with proper generics

### Critical

**1. src/api/auth.ts:78**
- **Category**: Bug
- **Issue**: Missing `await` on async password comparison
- **Why**: Function returns Promise<boolean> but caller expects boolean, causing auth bypass
- **Recommendation**: Add `await` before `bcrypt.compare()`

### Important

**2. src/controllers/order.ts:120**
- **Category**: Performance
- **Issue**: N+1 query pattern
- **Recommendation**: Use eager loading

---

## Next Steps
1. [ ] Fix critical bug (auth bypass)
2. [ ] Fix 3 standards violations
3. [ ] Address performance issue
```

---

Let me know when you're ready to begin the review.
