---
name: extract-figma
description: Extracts Figma frame design specs into docs/ai/requirements/DD-MM-YYYY-figma-{name}.md with smart large-frame handling.
---

## Goal

Extract complete design specifications from a Figma frame and save to `docs/ai/requirements/DD-MM-YYYY-figma-{name}.md`.
Output is detailed enough to:
1. **Implement** pixel-perfect UI without accessing Figma again.
2. **Validate** that code matches design (used by `/check-implementation`).

---

## Input

```
/extract-figma [figma-url] [name?]
```

- `figma-url`: Full Figma URL or frame URL (required).
- `name`: File name slug (kebab-case, optional — derived from frame name if omitted).

**Examples:**
```
/extract-figma https://figma.com/file/abc/design?node-id=123
/extract-figma https://figma.com/file/abc/design?node-id=123 login-page
```

---

## Workflow

### Step 1: Validate Input & Setup

**Parse arguments:**
- Extract Figma URL from user input (or ask if not provided)
- Derive `name` from frame name (kebab-case) if not explicitly given
- Output file path: `docs/ai/requirements/DD-MM-YYYY-figma-{name}.md`

**Check if file already exists:**
```
Read(file_path="docs/ai/requirements/DD-MM-YYYY-figma-{name}.md")
```
- If exists AND status is `complete`: Ask user:
  - Option A: "Re-extract and overwrite" (full re-extraction)
  - Option B: "Resume partial extraction" (continue from where left off)
  - Option C: "Cancel — file is already complete"
- If exists AND status is `partial`: Skip to Step 4 (resume from Extraction Status checkboxes)
- If not exists: Proceed to Step 2

**Verify Figma MCP connection:**
- Attempt to access the Figma file using the available Figma MCP tool
- If connection fails: Inform user — "Figma MCP not connected. Options: 1) Configure Figma MCP, 2) Provide design specs manually"
- Do NOT proceed if connection fails

---

### Step 2: Layout Scan (Always First)

**Purpose:** Understand the frame's structure and estimate complexity before deciding extraction strategy.

**What to fetch:**
- Frame name and path in the Figma file
- Top-level children: names, types (Frame/Group/Component/Text)
- Depth of nesting (how many levels deep)
- Total estimated node count

**Estimate complexity:**

| Complexity | Criteria                                                                    | Strategy       |
|------------|-----------------------------------------------------------------------------|----------------|
| Low        | ≤ 5 top-level sections, ≤ 40 total nodes, single viewport                  | Full extraction|
| Medium     | 6–12 top-level sections OR 41–100 nodes OR 2–3 viewports                   | Progressive    |
| High       | > 12 top-level sections OR > 100 nodes OR 4+ viewports OR multi-page frame | Progressive    |

**Report to user** (brief status):
```
Frame: "{Frame Name}"
Sections detected: {N}
Estimated nodes: {N}
Strategy: Full extraction / Progressive extraction (large frame detected)
```

---

### Step 3: Strategy Decision

#### Strategy A — Full Extraction (Low complexity)

Proceed directly to Step 5 (full extraction in one pass).

#### Strategy B — Progressive Extraction (Medium/High complexity)

**Explain to user:**
> "This frame is large ({N} sections, ~{N} nodes). To avoid missing details,
> I'll extract the layout structure first, then we'll go section by section.
> This ensures complete and accurate specs."

**Phase B1: Extract layout structure only**
- Frame overview (purpose, user type, product flow position)
- Complete hierarchy tree (all groups/frames to 2–3 levels deep)
- Layout type, container widths, grid system
- Section inventory: list all top-level sections with brief purpose

Do NOT extract component details yet.

**Write partial file:**
- Read template: `docs/ai/requirements/figma-template.md`
- Populate: Reference, Frame Overview, Layout Structure sections
- For each top-level section in hierarchy, add a placeholder under Component Specifications:
  ```markdown
  ### {Section Name}
  > ⏳ Not yet extracted. Run /extract-figma to detail this section.
  ```
- Set `status: partial` in frontmatter
- Fill Extraction Status checkboxes: check only "Frame overview and layout structure"

**Write the file:** `docs/ai/requirements/DD-MM-YYYY-figma-{name}.md`

**Ask user which section to detail next:**

```
AskUserQuestion:
  question: "Layout structure saved. Which section should I extract next?"
  options:
    - label: "{Section 1 name}" — {brief description}
    - label: "{Section 2 name}" — {brief description}
    - label: "{Section 3 name}" — {brief description}
    - label: "Extract design tokens first (colors, fonts, spacing)"
```

If more than 4 sections exist, present the 3 most important + "Extract design tokens first".
After user selects, proceed to Step 4 (section detail extraction).

---

### Step 4: Section Detail Extraction (Progressive Mode Loop)

**Triggered by:** User selecting a section, OR resume from partial file.

**For each selected section, extract:**

1. **If "Design Tokens" selected:**
   - All colors with hex codes and usage
   - All typography styles (font, size, weight, line-height, tracking)
   - Spacing scale (identify the base unit and all values used)
   - Shadows (exact CSS box-shadow values)
   - Border radius values
   - Update the "Design Tokens" section in the file
   - Check off "Design tokens" in Extraction Status

2. **If a UI section selected** (e.g., "Header", "Login Form", "Hero"):
   - Identify all unique components in this section
   - For each component, extract:
     - All states (default, hover, active, focus, disabled, loading, error)
     - All variants (size, style, intent)
     - Exact dimensions (width, height, min/max)
     - Padding and gap values
     - Colors per state/variant
     - Typography used
     - Border and border-radius
     - Shadow per state
     - **Icons and images** (see format below)
   - Document component hierarchy (which components nest inside others)
   - Update the placeholder for this section in the file
   - Check off this section in Extraction Status

   **Icon spec format** (within component spec):
   ```markdown
   **Icon: {icon-name}**
   - Figma Node ID: {node-id}
   - Library: {Heroicons / Lucide / Custom / etc.}
   - Container: {W}×{H}px, {flex center / inline / etc.}
   - Icon size: {W}×{H}px
   - Color: {hex} (apply via CSS `color` or SVG `fill`)
   - Placeholder path: `{ICONS_PATH}/{icon-name}.svg`
   - Export format: SVG
   ```

   **Image spec format** (within component spec):
   ```markdown
   **Image: {image-name}**
   - Figma Node ID: {node-id}
   - Container: {W}×{H}px
   - object-fit: {cover / contain / fill}
   - Aspect ratio: {W:H}
   - Border-radius: {value}
   - Placeholder path: `{IMAGES_PATH}/{image-name}.{ext}`
   - Export format: {JPG 2x / PNG / WebP}
   ```

   > `{ICONS_PATH}` and `{IMAGES_PATH}` are defined at the top of the Assets Export Table. Dev replaces them once per project.

**After each section extraction:**
- Update the file (replace placeholder with real content)
- Update Extraction Status checkboxes
- Check remaining uncompleted sections

**Ask if more sections to extract:**
```
AskUserQuestion:
  question: "'{Section}' extracted. What next?"
  options:
    - label: "{Next uncompleted section}" — continue extracting
    - label: "Extract responsive specs" — mobile/tablet/desktop differences
    - label: "Extract assets" — icons list, image dimensions
    - label: "Mark as complete" — done extracting
```

Repeat until user selects "Mark as complete" or all sections are done.

---

### Step 5: Full Extraction (Low Complexity)

Extract all sections in one pass without asking. Cover everything:

1. **Frame Overview**: Screen name, user type, purpose, product flow position
2. **Layout Structure**: Full hierarchy tree
3. **Design Tokens**: Complete colors, typography, spacing, shadows, border-radius
4. **Components**: Every unique component with all states and variants
5. **Responsive Specs**: For each breakpoint, document layout and component changes
6. **Assets**: For every icon and image:
   - Name, Figma node ID, type (icon/image), dimensions, color (icons), object-fit (images)
   - Placeholder path using symbolic vars: `{ICONS_PATH}/{name}.svg` or `{IMAGES_PATH}/{name}.{ext}`
   - Export format: SVG for icons, JPG 2x or PNG for images
   - Produce a consolidated export table at the end of the Assets section (see format below)
7. **Interaction Patterns**: Animation durations, easing, triggered properties
8. **Validation Notes**: Critical design decisions that are easy to miss

---

### Step 6: Finalize

**Read the template** to ensure output structure matches: `docs/ai/requirements/figma-template.md`

**Quality check before writing:**
- [ ] All colors documented with exact hex codes (no "blue" or "approximately")
- [ ] All typography styles captured with size, weight, line-height
- [ ] Spacing values are exact numbers (not "about 16px")
- [ ] All component states documented (not just default)
- [ ] All component variants documented
- [ ] Responsive layout changes explicitly noted per breakpoint
- [ ] No guessed or approximate values — exact only
- [ ] Every icon: node ID, container size, icon size, color, placeholder path documented
- [ ] Every image: node ID, container size, object-fit, aspect ratio, placeholder path documented
- [ ] Assets section contains a consolidated export table:
  ```markdown
  ## Assets Export Table
  > Replace `{ICONS_PATH}` and `{IMAGES_PATH}` with your project's actual asset folders.
  > Export each asset from Figma by node ID and place at the path. No layout changes needed.
  >
  > ICONS_PATH = (e.g. src/assets/icons, public/icons)
  > IMAGES_PATH = (e.g. src/assets/images, public/images)

  | Name | Node ID | Type | Dimensions | Color | Placeholder Path | Format |
  |------|---------|------|-----------|-------|-----------------|--------|
  | chevron-right | 123:456 | Icon | 24×24 | #374151 | {ICONS_PATH}/chevron-right.svg | SVG |
  | hero-banner | 789:012 | Image | 1440×480 | — | {IMAGES_PATH}/hero-banner.jpg | JPG 2x |
  ```

**Set frontmatter:**
```yaml
---
frame_url: {url}
frame_name: {frame name}
file_name: {figma file name}
extracted: {YYYY-MM-DD}
status: complete  # or partial if progressive mode not finished
---
```

**If any section was NOT extracted**, keep `status: partial`.

**Notify user:**
```
✓ Figma specs saved: docs/ai/requirements/DD-MM-YYYY-figma-{name}.md
  Status: complete / partial

Sections extracted:
  ✓ Design tokens
  ✓ Header components
  ✓ Form components
  ⏳ Responsive specs (not yet extracted)

Next steps:
  /extract-figma {url} {name}  → Resume and extract remaining sections
  /create-spec                 → Reference this file in a feature spec
```

---

## Output File

**Location:** `docs/ai/requirements/DD-MM-YYYY-figma-{name}.md`

**Structure** (follows `docs/ai/requirements/figma-template.md`):
1. Frontmatter (url, frame name, date, status)
2. Reference table
3. Frame Overview
4. Layout Structure (hierarchy tree)
5. Design Tokens (colors, typography, spacing, shadows, border-radius)
6. Component Specifications (each component: all states + variants)
7. Responsive Specifications (per breakpoint)
8. Assets (icons, images, illustrations)
9. Interaction Patterns
10. Validation Notes
11. Extraction Status (checkboxes)

---

## Error Handling

| Error                        | Action                                                     |
|------------------------------|------------------------------------------------------------|
| Figma MCP not connected      | Stop and tell user to configure MCP first                 |
| Invalid URL / no access      | Tell user to check URL and permissions                    |
| Frame not found              | List available frames and ask which to use                |
| Partial extraction left open | Resume from Extraction Status checkboxes in existing file |
| Template file missing        | Use inline fallback structure (Reference → Tokens → Components → Responsive) |

---

## Integration with create-spec

When `/create-spec` detects a Figma URL:
1. Check if `docs/ai/requirements/DD-MM-YYYY-figma-{name}.md` exists
2. If yes → Read it and include in Section 2a "Design Specifications"
3. If no → Tell user: "Run `/extract-figma {url}` first, then re-run `/create-spec`"

This keeps design extraction as a dedicated phase, separate from spec writing.
