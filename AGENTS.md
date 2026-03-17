# Repository Guidelines for AI Agents & Developers

**CRITICAL CONTEXT:**
This project relies heavily on **Code Generation**.

1. **NEVER** modify files inside `gen/`, `generated/`, or `ent/` directories directly.
2. **ALWAYS** edit the **Source** (Schema, GraphQL definitions, PKL) and run the appropriate generation task.

***

## 📚 Documentation Index

For detailed instructions, rules, and navigation, please refer to the specialized guides below.

## ⚙️ Key Configuration Files

Analyzing these files provides immediate context on dependencies and workflows:

* **`go.work`**: Defines the active modules in the workspace. **Source of Truth for module resolution.**
* **`Taskfile.yml`**: The registry of all runnable commands. Check this before proposing shell scripts.

***

## 📝 Git Commit Rules

* **NEVER** include "Generated with Claude Code" or similar AI attribution in commit messages
* **NEVER** include "Co-Authored-By: Claude" or any AI co-author lines
* Keep commit messages clean and focused on the changes only

## Summary

The structure separates concerns based on the 'reader' (Humans vs. AI vs. Editor):

* `**docs/**` These are explanatory guides for **human developers** (setup, workflows, API usage).

* `**AGENT.md**` **&** `**.ai/**` This is specifically for the AI Agent. `AGENT.md` acts as the entry point (system prompt) standardizing the context initialization. The `.ai` folder contains imperative rules, architectural constraints, and code examples optimized for LLM token efficiency, not human readability.

* `**.cursor/rules**` These aren't documentation but **active rules** for the Cursor editor. They inject specific context automatically before edits. If we aren't using Cursor, this folder is safe to remove.
