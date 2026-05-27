# Production Scaling Stages

> **Human reference only.** This file is not loaded at AI runtime.

The framework scales across four stages. Start at Stage 1 and advance only when the current stage creates friction. Skipping stages adds complexity without benefit.

---

## Stage 1 — Manual Markdown Routing

**What it is:** The AI reads `framework_runtime.md` and uses its routing table to decide which modules to load. All file reads and writes are manual or AI-assisted within the session.

**Works with:**
- Claude Code
- Cursor
- VS Code with AI extensions
- ChatGPT Projects
- Any AI with file access

**When to advance:** When you're spending time manually curating which files get loaded, or when context assembly is becoming inconsistent across sessions.

---

## Stage 2 — Script-Assisted Routing

**What it is:** Add lightweight Python scripts that automate the mechanical parts of the framework.

**Add:**
- `relevance_scanner.py` — Scans files, reads metadata, pre-scores relevance before the AI session starts
- `context_builder.py` — Assembles the execution prompt from pre-scored files automatically
- `summarizer.py` — Compresses memory files that exceed size limits on a schedule

**Benefit:** The AI session starts with context already assembled. Reduces token overhead on long-running projects.

**When to advance:** When you want semantic understanding of relevance (beyond keyword matching) or when the project has enough files that keyword-based scoring misses things.

---

## Stage 3 — Lightweight Semantic Search

**What it is:** Add vector embeddings to the relevance scoring layer, enabling semantic file retrieval without changing the framework's markdown structure.

**Add:**
- File embeddings (generated on write/update, stored locally)
- A vector search step in `relevance_scanner.py` that replaces or supplements keyword matching
- Re-ranking: keyword score + semantic similarity score → combined threshold

**Works with:**
- Local embedding models (Ollama, LM Studio)
- OpenAI embeddings API
- Any embedding model accessible from Python

**Benefit:** Surfaces relevant files the keyword matcher would miss (e.g. a file about "vendor deduplication" being relevant to a task about "supplier normalization").

**When to advance:** When the project has grown to the point that a single AI orchestrating everything is too slow or context-limited for the work at hand.

---

## Stage 4 — Agentic Automation

**What it is:** Add multi-agent orchestration so specialized sub-agents handle distinct domains in parallel, with the framework's modules acting as shared context.

**Add:**
- Orchestrator agent — routes tasks to specialist agents using `modules/routing.md` logic
- Specialist agents — each loaded with only their domain modules
- Autonomous task generation — the orchestrator creates and queues tasks without human prompting
- Execution chaining — completed tasks automatically trigger dependent tasks

**Works with:**
- Claude Code (multi-agent)
- OpenAI Assistants with tool use
- LangGraph, CrewAI, or equivalent orchestration frameworks
- Custom agent pipelines

**Benefit:** Parallelism and specialization at scale. The markdown architecture remains the shared source of truth across all agents.

**Note:** The framework's modular structure is what makes Stage 4 possible without chaos. If you try to run agentic automation on a monolithic prompt, agents will conflict. The module boundaries are the agent boundaries.
