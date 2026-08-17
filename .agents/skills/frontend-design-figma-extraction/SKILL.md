---
name: extract-figma
description: Use when the user wants to extract a Figma frame into `docs/ai/requirements/DD-MM-YYYY-figma-{name}.md` with exact specs, partial-progress support, and large-frame handling.
---

# Extract Figma

This skill mirrors the `/extract-figma` workflow.
Use it to turn a Figma URL or frame into an implementation-ready design spec file that other workflows can reuse without re-reading Figma.

## Inputs

- Figma URL or frame URL
- Optional output slug in kebab-case

## Output

- `docs/ai/requirements/DD-MM-YYYY-figma-{name}.md`

The output must be detailed enough to:

1. implement the UI without reopening Figma
2. validate implementation against the design later

## Tool Mapping

- Figma access tools -> use any available Figma integration or repository-provided workflow when present
- File read/write/edit tools -> inspect supporting files with the runtime's file tools and update outputs with precise edits
- User clarification -> ask the user directly only when the file, frame, or next extraction target is unclear

## Workflow

### 1. Validate input and setup

Parse:

- the Figma URL
- the target output name
- the output path `docs/ai/requirements/DD-MM-YYYY-figma-{name}.md`

If the file already exists:

- if `status: complete`, ask whether to overwrite or stop
- if `status: partial`, resume from the unchecked extraction items

Do not continue if the Figma source cannot be accessed or if no suitable Figma integration is available in the current runtime.

### 2. Scan layout before detailed extraction

Always inspect the frame structure first to estimate complexity:

- frame name and path
- top-level sections
- approximate nesting depth
- approximate node count

Choose strategy:

- low complexity -> full extraction
- medium or high complexity -> progressive extraction

Report the chosen strategy briefly before continuing.

### 3. Use the right extraction strategy

#### Full extraction

Use this when the frame is small enough to capture in one pass.

Extract:

- frame overview
- layout hierarchy
- design tokens
- component specs with states and variants
- responsive specs
- assets
- interaction patterns
- validation notes

Set `status: complete` unless some sections were intentionally skipped.

#### Progressive extraction

Use this when the frame is large or complex.

First pass:

- extract only the layout structure and section inventory
- create placeholders for each top-level section
- set `status: partial`

Then iterate section by section:

- design tokens
- individual UI sections such as header, hero, form, sidebar
- responsive specs
- assets

After each extracted section:

- replace the placeholder with real content
- update extraction-status checkboxes
- ask which remaining section to extract next when needed

### 4. Extract exact specs only

Never guess or approximate values.
Capture exact values for:

- colors with hex codes and usage notes
- typography with family, size, weight, line-height, tracking when relevant
- spacing scale and repeated layout values
- shadows and border radius
- component dimensions
- padding, gaps, and margins
- states: default, hover, active, focus, disabled, loading, error, success when present
- variants: size, style, intent
- responsive differences by breakpoint
- icons, images, and export-sensitive assets:
  - For each icon: node ID, library (Heroicons/Lucide/Custom), container size, icon size, color, placeholder path (`{ICONS_PATH}/{name}.svg`), export format SVG
  - For each image: node ID, container size, object-fit, aspect ratio, border-radius, placeholder path (`{IMAGES_PATH}/{name}.{ext}`), export format
  - Icon spec example:
    ```
    Icon: chevron-right | Node ID: 123:456 | Container: 24×24px | Color: #374151 | Path: {ICONS_PATH}/chevron-right.svg
    ```
  - Image spec example:
    ```
    Image: hero-banner | Node ID: 789:012 | Container: 1440×480px | object-fit: cover | Path: {IMAGES_PATH}/hero-banner.jpg | Format: JPG 2x
    ```
  - Produce a consolidated Assets Export Table at the end of the Assets section:
    - First line defines the symbolic vars: `ICONS_PATH = (replace with project path)` and `IMAGES_PATH = (replace with project path)`
    - Table columns: Name | Node ID | Type | Dimensions | Color | Placeholder Path | Format
    - Dev replaces the two vars once per project — no other changes needed

### 5. Write the output file

Follow `docs/ai/requirements/figma-template.md` when it exists.
If the template is missing, use this fallback structure:

1. frontmatter
2. reference
3. frame overview
4. layout structure
5. design tokens
6. component specifications
7. responsive specifications
8. assets
9. interaction patterns
10. validation notes
11. extraction status

Frontmatter should include:

```yaml
---
frame_url: {url}
frame_name: {frame name}
file_name: {figma file name}
extracted: {YYYY-MM-DD}
status: complete
---
```

Use `status: partial` if any planned extraction sections remain unfinished.

## Completion Checklist

- all values are exact, not inferred
- all important components include states and variants
- responsive changes are explicit
- large frames keep resumable extraction status
- the output is sufficient for implementation without reopening Figma

## Error Handling

- If Figma access fails, stop and tell the user what is missing.
- If the URL is invalid or the frame cannot be found, ask for the correct frame.
- If a partial file exists, resume from it instead of starting over by default.
- If `docs/ai/requirements/figma-template.md` is missing, use the fallback structure and continue.

## Integration with Spec-Driven Workflow

`/spec` should read the generated `DD-MM-YYYY-figma-{name}.md` file when turning a design into a durable feature spec.
If that file does not exist yet, the spec creation flow should stop guessing and tell the user to run `/extract-figma` first.
