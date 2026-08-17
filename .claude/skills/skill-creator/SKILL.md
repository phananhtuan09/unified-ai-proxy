---
name: skill-creator
description: Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.
---

# Skill Creator

Guide for creating effective skills that extend Claude's capabilities.

For detailed step-by-step process, see `references/creation-process.md`.

---

## What Skills Provide

1. **Specialized workflows** - Multi-step procedures for specific domains
2. **Tool integrations** - Instructions for working with specific file formats or APIs
3. **Domain expertise** - Company-specific knowledge, schemas, business logic
4. **Bundled resources** - Scripts, references, and assets for complex tasks

---

## Core Principles

### Concise is Key

The context window is a public good. Only add context Claude doesn't already have.

Challenge each piece: "Does Claude really need this?" and "Does this justify its token cost?"

Prefer concise examples over verbose explanations.

### Degrees of Freedom

Match specificity to task fragility:

| Freedom | When to Use | Example |
|---------|-------------|---------|
| **High** (text instructions) | Multiple valid approaches | Heuristics, guidelines |
| **Medium** (pseudocode) | Preferred pattern exists | Configurable workflows |
| **Low** (specific scripts) | Fragile operations | Critical sequences |

---

## Skill Anatomy

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter (name, description)
│   └── Markdown instructions
└── Bundled Resources (optional)
    ├── scripts/     - Executable code (deterministic, reusable)
    ├── references/  - Documentation loaded on demand
    └── assets/      - Files used in output (templates, images)
```

### SKILL.md

- **Frontmatter**: `name` and `description` only. Description is the trigger mechanism.
- **Body**: Instructions loaded after skill triggers. Keep under 500 lines.

### Bundled Resources

| Type | Purpose | Load Behavior |
|------|---------|---------------|
| `scripts/` | Deterministic code | Execute without reading |
| `references/` | Documentation | Load when needed |
| `assets/` | Output resources | Use without loading |

**Don't include**: README.md, CHANGELOG.md, installation guides, etc.

---

## Progressive Disclosure

Three-level loading system:

1. **Metadata** (~100 tokens) - Always in context
2. **SKILL.md body** (<5k tokens) - When skill triggers
3. **References** (unlimited) - As needed

### Key Patterns

**Pattern 1: High-level guide with references**
```markdown
## Quick start
[Core example]

## Advanced features
- **Form filling**: See references/forms.md
- **API reference**: See references/api.md
```

**Pattern 2: Domain-specific organization**
```
bigquery-skill/
├── SKILL.md (overview)
└── references/
    ├── finance.md
    ├── sales.md
    └── product.md
```

**Pattern 3: Framework variants**
```
cloud-deploy/
├── SKILL.md (workflow + selection)
└── references/
    ├── aws.md
    ├── gcp.md
    └── azure.md
```

**Guidelines:**
- Keep references one level deep from SKILL.md
- Include TOC for files >100 lines
- Avoid duplication between SKILL.md and references

---

## Creation Process

1. **Understand** - Gather concrete usage examples
2. **Plan** - Identify reusable scripts, references, assets
3. **Initialize** - Run `scripts/init_skill.py <name> --path <dir>`
4. **Edit** - Implement resources and write SKILL.md
5. **Package** - Run `scripts/package_skill.py <path>`
6. **Iterate** - Improve based on real usage

For detailed instructions on each step, see `references/creation-process.md`.

---

## Quick Checklist

- [ ] Description includes triggers and "when to use"
- [ ] SKILL.md under 500 lines
- [ ] Large content moved to references/
- [ ] Scripts tested and working
- [ ] No unnecessary documentation files
- [ ] References linked from SKILL.md with clear guidance
