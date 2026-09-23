---
name: markdown-writer
description: Enforces professional Markdown formatting standards and Markdownlint compliance (e.g., MD031 blank lines around code fences, heading spacing, list padding, table formatting, and clean typography).
---

# Markdown Writer & Formatter Skill

This skill defines rules and best practices for authoring, editing, and auditing Markdown (`.md`) documents across the workspace (documentation, skills, workflows, and READMEs) to ensure consistent, lint-clean formatting.

## Core Markdownlint Compliance Rules

### 1. Fenced Code Blocks (MD031, MD040)

- **Surround Fences with Blank Lines (MD031)**: Always leave at least one empty line before the opening fence (```) and after the closing fence (```).
- **Language Identifiers (MD040)**: Always specify an explicit language identifier (e.g., ```` ```go ````, ```` ```bash ````, ```` ```latex ````, ```` ```markdown ````). Use `text` or `plaintext` for unhighlighted output.
- **Indentation in Lists**: If a fenced block belongs to a list item, indent the fence by 2 or 4 spaces matching the list item's indentation, and preserve blank lines before and after the fence.

### 2. Headings (MD018, MD022, MD025)

- **Surround Headings with Blank Lines (MD022)**: Insert one blank line before and after all headings (`#`, `##`, `###`).
- **Space After Hash (MD018)**: Always place exactly one space between `#` symbols and the heading text (e.g., `## Title`, never `##Title`).
- **Single Top-Level Heading (MD025)**: Use exactly one top-level `# Heading` per markdown file (excluding YAML frontmatter).

### 3. Lists and Spacing (MD030, MD032)

- **Surround Lists with Blank Lines (MD032)**: Ensure a blank line precedes the start of a list and follows the end of a list.
- **Consistent List Markers**: Use hyphens (`-`) for unordered lists and sequential numbers (`1.`, `2.`, `3.`) for ordered lists.
- **Single Space After Marker (MD030)**: Use a single space between the bullet/number and list content (e.g., `- Item`, `1. Step`).

### 4. Whitespace and Blank Lines (MD009, MD012)

- **No Trailing Spaces (MD009)**: Never leave trailing spaces at the end of lines.
- **No Multiple Consecutive Blank Lines (MD012)**: Keep paragraph separation to exactly one blank line. Never leave two or more blank lines in a row.
- **File End (MD047)**: Ensure all Markdown files end with exactly one trailing newline.

### 5. Tables and Visual Blocks

- Separate tables from preceding and following paragraphs with blank lines.
- Keep table columns aligned with pipes `|` and balanced hyphens for clean source readability.

### 6. YAML Frontmatter Standards

- Files with metadata must begin on Line 1 with `---` and close frontmatter with `---` on its own line.
- Leave exactly one blank line between the closing `---` and the first Markdown heading.

## Auditing & Fixing Checklist

When reviewing or writing Markdown files:

1. Scan for code blocks attached directly to preceding text or list bullets without blank lines.
2. Check that all fenced blocks specify a syntax language.
3. Verify that headings have blank lines above and below them.
4. Ensure no consecutive empty lines exist.
