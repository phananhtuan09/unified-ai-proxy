# Skill Creation Process - Detailed Steps

This document contains the detailed step-by-step process for creating skills.

---

## Step 1: Understanding the Skill with Concrete Examples

Skip this step only when the skill's usage patterns are already clearly understood. It remains valuable even when working with an existing skill.

To create an effective skill, clearly understand concrete examples of how the skill will be used. This understanding can come from either direct user examples or generated examples that are validated with user feedback.

For example, when building an image-editor skill, relevant questions include:

- "What functionality should the image-editor skill support? Editing, rotating, anything else?"
- "Can you give some examples of how this skill would be used?"
- "I can imagine users asking for things like 'Remove the red-eye from this image' or 'Rotate this image'. Are there other ways you imagine this skill being used?"
- "What would a user say that should trigger this skill?"

To avoid overwhelming users, avoid asking too many questions in a single message. Start with the most important questions and follow up as needed for better effectiveness.

Conclude this step when there is a clear sense of the functionality the skill should support.

---

## Step 2: Planning the Reusable Skill Contents

To turn concrete examples into an effective skill, analyze each example by:

1. Considering how to execute on the example from scratch
2. Identifying what scripts, references, and assets would be helpful when executing these workflows repeatedly

**Example: PDF Editor Skill**
- Query: "Help me rotate this PDF"
- Analysis: Rotating a PDF requires re-writing the same code each time
- Solution: A `scripts/rotate_pdf.py` script would be helpful

**Example: Frontend Webapp Builder Skill**
- Query: "Build me a todo app" or "Build me a dashboard"
- Analysis: Writing a frontend webapp requires the same boilerplate each time
- Solution: An `assets/hello-world/` template containing boilerplate files

**Example: BigQuery Skill**
- Query: "How many users have logged in today?"
- Analysis: Querying BigQuery requires re-discovering table schemas each time
- Solution: A `references/schema.md` file documenting the table schemas

To establish the skill's contents, analyze each concrete example to create a list of the reusable resources to include: scripts, references, and assets.

---

## Step 3: Initializing the Skill

Skip this step only if the skill being developed already exists, and iteration or packaging is needed.

When creating a new skill from scratch, always run the `init_skill.py` script:

```bash
scripts/init_skill.py <skill-name> --path <output-directory>
```

The script:
- Creates the skill directory at the specified path
- Generates a SKILL.md template with proper frontmatter and TODO placeholders
- Creates example resource directories: `scripts/`, `references/`, and `assets/`
- Adds example files in each directory that can be customized or deleted

After initialization, customize or remove the generated SKILL.md and example files as needed.

---

## Step 4: Edit the Skill

When editing the skill, remember that it's being created for another instance of Claude to use. Include information that would be beneficial and non-obvious to Claude.

### Learn Proven Design Patterns

Consult these helpful guides based on your skill's needs:
- **Multi-step processes**: See `references/workflows.md`
- **Specific output formats**: See `references/output-patterns.md`

### Start with Reusable Skill Contents

Begin implementation with the reusable resources identified: `scripts/`, `references/`, and `assets/` files.

Note: This step may require user input. For example, when implementing a `brand-guidelines` skill, the user may need to provide brand assets or templates.

Added scripts must be tested by actually running them to ensure there are no bugs and that the output matches expectations.

Any example files and directories not needed for the skill should be deleted.

### Update SKILL.md

**Writing Guidelines:** Always use imperative/infinitive form.

#### Frontmatter

Write the YAML frontmatter with `name` and `description`:

- `name`: The skill name
- `description`: Primary triggering mechanism. Include:
  - What the skill does
  - Specific triggers/contexts for when to use it
  - All "when to use" information (body is only loaded after triggering)

**Example description for a `docx` skill:**
```
"Comprehensive document creation, editing, and analysis with support for tracked changes, comments, formatting preservation, and text extraction. Use when Claude needs to work with professional documents (.docx files) for: (1) Creating new documents, (2) Modifying or editing content, (3) Working with tracked changes, (4) Adding comments, or any other document tasks"
```

Do not include any other fields in YAML frontmatter.

#### Body

Write instructions for using the skill and its bundled resources.

---

## Step 5: Packaging a Skill

Once development is complete, package into a distributable .skill file:

```bash
scripts/package_skill.py <path/to/skill-folder>
```

Optional output directory:
```bash
scripts/package_skill.py <path/to/skill-folder> ./dist
```

The packaging script will:

1. **Validate** the skill automatically, checking:
   - YAML frontmatter format and required fields
   - Skill naming conventions and directory structure
   - Description completeness and quality
   - File organization and resource references

2. **Package** the skill if validation passes, creating a .skill file (zip with .skill extension)

If validation fails, fix errors and run again.

---

## Step 6: Iterate

After testing the skill, users may request improvements.

**Iteration workflow:**
1. Use the skill on real tasks
2. Notice struggles or inefficiencies
3. Identify how SKILL.md or bundled resources should be updated
4. Implement changes and test again
