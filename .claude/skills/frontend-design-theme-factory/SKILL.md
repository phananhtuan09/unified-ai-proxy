---
name: frontend-design-theme-factory
description: |
  Theme/color selection when user has no aesthetic direction and wants agent to decide.
  Use when: user is uncertain — "suggest colors", "what theme", "help me choose", "you pick".
  Keywords: suggest theme, suggest colors, what theme, what colors, help choose, help pick, no direction, you choose, recommend colors
  Do NOT use for: user has colors/style in mind, applying existing theme.
---

# Theme Factory

## Purpose
Generate cohesive, beautiful UI themes when no design specifications provided, preventing generic monotone interfaces.

---

## Core Principle

**Brand personality drives design decisions.**

Every UI communicates personality through color, typography, and style. Rather than defaulting to arbitrary colors, ask about brand personality first, then generate cohesive themes that match.

**Good theme** = Intentional, cohesive, personality-aligned
**Bad theme** = Random colors, inconsistent, no personality

---

## When to Use Theme Factory

**Use when user EXPLICITLY asks for theme help:**
- "What theme should I use?"
- "Help me pick colors"
- "What colors work well together?"
- User uncertain about aesthetic choices

**Don't use if:**
- User has clear aesthetic in mind → design-fundamentals handles it
- Figma file/URL provided → use figma-design-extraction
- User already chose colors/fonts → just apply them

---

## Theme Selection Process

### Step 1: Understand Brand Personality

**Ask user (concise, 1-2 questions max):**

**Question 1: "What personality should this UI have?"**
- **Professional** - Corporate, trustworthy, business-focused
- **Creative** - Bold, artistic, unique, expressive
- **Minimal** - Clean, sophisticated, simple, timeless
- **Bold/Tech** - Modern, eye-catching, innovative
- **Warm/Organic** - Natural, welcoming, friendly
- **Playful** - Fun, colorful, approachable, energetic

**Question 2: "Any color preferences or inspirations?"**
- Specific colors they like
- Competitor/inspiration URLs
- Or: "Auto-generate based on personality"

### Step 2: Present Theme Options

**Show 2-3 pre-defined themes from `.claude/themes/` matching personality**

Example:
```
1. Professional Blue - Corporate, trustworthy, clean
2. Minimal Monochrome - Sophisticated, timeless
3. Generate custom theme based on your preferences
```

**If user wants custom:** Generate based on personality + color preferences

---

## Theme Anatomy

### 1. Colors

**Primary Palette (shades 50-900):**
- Used for: Main actions, links, buttons, key UI elements
- Scale: 50 (lightest) → 900 (darkest)

**Neutral Palette (50-900):**
- Used for: Text, backgrounds, borders, subtle UI
- Principle: True grays or slightly tinted toward primary

**Semantic Colors:**
- Success: Green (growth, positive, confirmation)
- Warning: Yellow/orange (caution, attention)
- Error: Red (danger, mistakes, destructive)
- Info: Blue (information, neutral)

**Why this structure:**
- Full palette enables light/dark modes
- Semantic colors ensure accessibility
- Neutrals provide visual breathing room

### Dark Mode Variant

**Derive dark palette from the same hue — don't invent new colors.**

| Role | Light mode | Dark mode |
|------|-----------|-----------|
| Page background | neutral-50 | neutral-950 |
| Surface (card/panel) | neutral-100 | neutral-900 |
| Surface hover | neutral-200 | neutral-800 |
| Body text | neutral-700 | neutral-200 |
| Heading text | neutral-900 | neutral-50 |
| Border | neutral-200 | neutral-700 |
| Primary action | primary-600 | primary-400 |
| Primary text on surface | primary-700 | primary-300 |

**Key rules:**
- Primary hue stays the same — only shift lightness (600→400 for dark)
- Semantic colors (success/error/warning) lighten ~100-200 steps in dark mode
- Never use the same shade for both modes — always invert the scale direction

---

### 2. Typography

**See design-fundamentals for typography pairing principles.**

Theme should define:
- Font families (heading + body)
- Type scale values
- Font weights available
- Line heights

### 3. Spacing System

**See design-fundamentals for spacing scale principles.**

Theme should define:
- Base unit (4px or 8px)
- Scale values

### 4. Visual Properties

**Border Radius:**
- **Minimal**: Small or none (sharp, clean)
- **Friendly**: Medium roundness (approachable, soft)
- **Playful**: Large roundness or full pill shapes (fun, organic)

**Shadows:**
- Subtle shadows for depth (professional, minimal)
- Stronger shadows for cards/modals (modern, bold)
- No shadows for flat design (minimalist)
- Define elevation system (subtle/medium/strong)

---

## Contrast Check

**WCAG AA minimums (required):**
- Normal text (< 18px): **4.5:1**
- Large text (≥ 18px bold / ≥ 24px): **3:1**
- UI components (buttons, inputs, icons): **3:1**

**Must-check pairs for every theme:**

| Pair | Target | Common failure |
|------|--------|---------------|
| Body text on page background | 4.5:1 | Light gray text on white |
| Heading on page background | 4.5:1 | Medium neutral on light neutral |
| White text on primary button | 4.5:1 | Light primaries (yellow, lime) |
| Primary text on white | 4.5:1 | Mid-range primaries (400-500) |
| Placeholder text on input | 3:1 | neutral-400 on white |

**Quick mental check — the 4.5:1 rule of thumb:**
- Dark text on light bg: text must be ≤ neutral-600 on neutral-50
- Light text on dark bg: text must be ≥ neutral-200 on neutral-900
- Primary-500 on white usually **fails** — use primary-600 or darker for text/icons

**If primary color is light (yellow, lime, teal-300):**
- Never use as button background with white text → use dark text instead
- Or shift to primary-700 for text, primary-600 for buttons

---

## Color Theory Foundations

### Color Harmony Methods

**Complementary** - Opposites on color wheel:
- High contrast, energetic, bold
- Example: Blue + Orange, Red + Green
- Use for: Bold, tech, creative personalities

**Analogous** - Adjacent on color wheel:
- Harmonious, calm, cohesive
- Example: Blue + Purple + Teal
- Use for: Professional, minimal, warm personalities

**Triadic** - Three evenly spaced:
- Balanced, vibrant, dynamic
- Example: Red + Yellow + Blue
- Use for: Playful, creative personalities

**Monochromatic** - Single hue, varying lightness:
- Clean, sophisticated, minimal
- Use for: Minimal, professional personalities

### Color Psychology

**Quick reference:**
- Blue = Trust, professional
- Green = Growth, health
- Purple = Creativity, luxury
- Red = Energy, urgency
- Orange = Friendly, energetic
- Yellow = Optimism, playfulness
- Neutral = Sophisticated, minimal

**When choosing colors:**
- Consider brand personality
- Use color harmony methods for cohesive palettes
- Verify contrast ratios meet WCAG AA

---

## Theme Documentation Format

**In planning doc:**

```markdown
## Theme Specification

### Selected Theme
- Name: [Theme Name]
- Source: .claude/themes/[filename].theme.json OR "Custom generated"
- Personality: [personality traits]

### Color Palette
**Primary ([Color Name]):**
- Base: [hex] (buttons, links, actions)
- Hover: [hex]
- Active: [hex]
- Full scale: 50-900 shades

**Neutral:**
- Lightest: [hex] (backgrounds)
- Medium: [hex] (body text)
- Darkest: [hex] (headings)
- Full scale: 50-900 shades

**Semantic:**
- Success: [hex] / dark: [hex]
- Error: [hex] / dark: [hex]
- Warning: [hex] / dark: [hex]
- Info: [hex] / dark: [hex]

### Dark Mode Palette
- Background: [hex] (neutral-950)
- Surface: [hex] (neutral-900)
- Body text: [hex] (neutral-200)
- Heading: [hex] (neutral-50)
- Border: [hex] (neutral-700)
- Primary action: [hex] (primary-400)

### Contrast Verification
- Body text / background: [ratio]:1 ✓/✗
- White / primary button: [ratio]:1 ✓/✗
- Primary text / white: [ratio]:1 ✓/✗

### Typography
- Heading: [Font family], [weight], [line-height]
- Body: [Font family], [weight], [line-height]
- Scale: [values]

### Spacing
- Scale: [values - base-4 or base-8]

### Visual Style
- Border radius: [small/medium/large]
- Shadows: [subtle/medium/strong]
```

---

## Theme Application Guidelines

**Consistency Rules:**

**Do:**
- Use only colors from theme palette
- Apply spacing scale consistently
- Follow typography scale
- Use semantic colors for their intended purpose

**Don't:**
- Mix colors from multiple themes
- Use arbitrary values
- Override theme for one-off styles
- Ignore semantic color meanings

**Component Theming:**

**Buttons:**
- Primary: Primary color background
- Secondary: Secondary/neutral background
- Outline: Transparent background, primary border
- Ghost: Transparent, primary text only

**States:**
- Hover: Slightly darker/more saturated
- Active: Noticeably darker than hover
- Disabled: Muted neutral shades
- Focus: Primary color ring

---

## Common Mistakes

1. **No personality** - Generic theme
   → Fix: Choose personality first
2. **Arbitrary tweaks** - Modifying theme colors slightly
   → Fix: Use theme palette exactly as defined
3. **Semantic color abuse** - Green button for delete
   → Fix: Red (error) for destructive, green (success) for positive
4. **Ignoring color harmony** - Random combinations
   → Fix: Use color harmony methods
5. **Theme mixing** - Using colors from multiple themes
   → Fix: Commit to one theme
6. **Copying examples literally** - Using exact values from docs
   → Fix: Generate unique values matching personality

---

## Validation Checklist

- [ ] Brand personality identified
- [ ] Color harmony method used
- [ ] Primary palette complete (50-900 shades)
- [ ] Neutral palette defined (50-900 shades)
- [ ] Semantic colors defined
- [ ] Typography pairing chosen
- [ ] Type scale defined
- [ ] Spacing scale defined
- [ ] Border radius values chosen
- [ ] Shadow system defined
- [ ] Theme documented in planning doc
- [ ] All colors pass WCAG AA contrast
- [ ] Key contrast pairs verified (text/bg, white/primary, primary/white)
- [ ] Dark mode palette derived (neutral scale inverted, primary shifted)
- [ ] Dark mode semantic colors adjusted (+100-200 steps lighter)

---

## Key Takeaway

**Color harmony + brand personality = cohesive themes.**

When users need help choosing colors/fonts:
1. Ask brand personality
2. Apply color harmony methods
3. Generate complete theme specification
4. Apply using design-fundamentals principles

Theme-factory provides **interactive selection** and **color theory methods**.
Design-fundamentals provides **application principles** and **technical foundation**.

Together: Distinctive, cohesive, accessible UI themes.
