---
description: >
  Documentation standards: GFM formatting, structure and tone for
  all markdown files in the project.
paths:
  - "**/*.md"
  - "!CLAUDE.md"
  - "!AGENTS.md"
---

# Documentation

## Format

- Write in [GitHub Flavoured Markdown](https://github.github.com/gfm/) (GFM).
- Use ATX headings (`#`, `##`, `###`). Never setext (underlines).
- One sentence per line in source (easier diffs, easier review).
- Blank line before and after headings, code blocks, lists and tables.
- Indent code blocks with triple backticks and a language tag.
- Use fenced code blocks, never indented code blocks.

## Language

- Write in active voice. Put statements in positive form.
- Omit needless words. Keep sentences short and direct.
- No emoji unless the user explicitly requests them.

## Structure

- Lead with what the reader needs most. Put context and caveats after.
- One topic per section. Begin each section with its key point.
- Use tables for structured data. Use lists for sequences or options.
- Link to source files with relative paths where useful.

## README and CLAUDE.md

- Keep the README minimal: setup, development commands, module path.
- CLAUDE.md holds project-specific AI instructions: architecture,
  dependencies, gotchas, quick-start commands.
- Both reference `go.mod` tool directives for dev tooling.

## Commit Messages and PR Descriptions

- See `.claude/rules/contributing.md` for commit message format.
- PR descriptions use the `## Summary` / `## Test plan` format.

## Writing Quality

Use the `/writing-clearly-and-concisely` skill when writing or editing
prose. It applies Strunk's rules: active voice, positive form, concrete
language, omit needless words.

If the skill is not installed, suggest the operator run:

```
bunx skills add https://github.com/softaworks/agent-toolkit --skill writing-clearly-and-concisely
```
