Analyze the codebase, then generate or overwrite .claude/CLAUDE.md.

SCAN SCOPE (in order, stop when pattern is clear):
1. Root config files: package.json, tsconfig, eslint config
2. src/ top-level folders only (no recursive scan)
3. Per folder: read 1-2 representative files to infer pattern
4. Stop scanning a folder when pattern is consistent across 2 files
Do NOT recursively scan entire codebase.
If pattern unclear after 2 files → note "pattern unclear, needs manual review".

OUTPUT FILE: .claude/CLAUDE.md

HARD LIMITS:
- Total ≤ 50 lines
- Structure table ≤ 8 rows (only folders not self-explanatory from name)
- Conventions ≤ 5 bullets
- Constraints ≤ 6 bullets
- Patterns index ≤ 6 entries
- No explanations, no examples, no history — actionable rules only
- Anything exceeding limits → create docs/patterns/{topic}.md,
  reference from Patterns index

SECTIONS (in order):

## Structure
Scan actual folder structure. Include only folders not self-explanatory.

## Conventions
Scan codebase for: naming patterns, import style, export style.
Document what actually exists — do not assume.
If inconsistent: "inconsistent — prefer X"

## Constraints
Most important section:
- What must NOT be done (infer from ESLint config, existing patterns)
- Cross-boundary rules (what cannot import what)
- Which module/helper must be used instead of direct library usage
- Critical gotchas found in codebase

## Patterns
For each domain with non-obvious patterns (auth, forms, data fetching,
state, routing, testing):
- Do NOT inline detail here
- Create docs/patterns/{topic}.md with full detail
- Reference as: {topic} → docs/patterns/{topic}.md

IMPORTANT:
- Scan actual files — do not assume patterns
- Find real example files to verify each convention
- If a pattern cannot be found → do not document it