# PrintingPress

Generate professional branded PDFs from markdown content using Claude Code. Supports Reach Industries and Lumi branding with multiple classification levels, all rendered via WeasyPrint inside Docker.

## How It Works

```
You give Claude a markdown file (or paste content)
  → Claude converts it to styled HTML using the component library
    → WeasyPrint renders it to a branded PDF inside Docker
```

No design tools needed. No manual formatting. Just content in, PDF out.

## Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/Reach-Industries/PrintingPress.git
cd PrintingPress

# 2. Make sure Docker Desktop is running

# 3. Build the example showcases
./build.sh documents/examples/reach_showcase.py
./build.sh documents/examples/lumi_showcase.py

# 4. Open output/Reach_Component_Showcase.pdf or output/Lumi_Component_Showcase.pdf
```

## Using with Claude Code

Open the repo in Claude Code. There are two ways to create documents:

### Option 1: Slash Commands

- **`/new-doc`** — Interactive document creation. Claude asks for title, brand, classification, and source content, then builds the PDF.
- **`/build`** — Rebuild an existing document from `documents/`.

### Option 2: Natural Language Prompts

You don't need slash commands. Just tell Claude what you want. Here are some example prompts:

**Creating a new document from a markdown file:**
> Turn this markdown file into a Reach branded PDF: /path/to/my-document.md

**Specifying classification:**
> Create a strictly confidential Reach document from this markdown — it's going to HR

**Choosing the Lumi brand:**
> I need a Lumi branded PDF from this content. It's a technical deep-dive, classification is confidential.

**Pasting content directly:**
> Turn this into a Reach branded PDF:
>
> [paste your markdown here]

**Rebuilding after edits:**
> Rebuild the gateway plan document

**Landscape layout:**
> Create a Lumi landscape PDF from this architecture overview

**Asking what's available:**
> What brands and components are available in PrintingPress?

Claude reads the `CLAUDE.md` file automatically when you open this repo, so it knows all the available CSS classes, brands, layouts, and classification levels without you having to explain anything.

## Brands

| Brand | Cover Style | Use For |
|-------|------------|---------|
| **Reach** | Navy full-bleed with REACH watermark | Corporate docs, HR, OKRs, internal reviews |
| **Lumi** | White with blue→purple gradient header, ISO/SOC2 badges | Product docs, investor materials, technical plans |

## Classification Levels

| Level | Cover Badge | Footer |
|-------|------------|--------|
| `non-confidential` | Non-Confidential | Non-Confidential |
| `internal` | Internal | Internal Use Only |
| `confidential` | Internal · Confidential | Confidential · Internal Use Only |
| `strictly-confidential` | STRICTLY CONFIDENTIAL · HR Only | STRICTLY CONFIDENTIAL · HR Distribution Only |
| `pre-nda` | Pre-NDA · External | Pre-NDA · External Distribution |

## Available Components

The CSS component library includes:

- **Typography** — Headings (h1/h2/h3), body text, lists, horizontal rules
- **Person Cards** — Color-coded assessment cards with badges (red/amber/green/blue/grey)
- **Tables** — Data tables with navy headers and status-colored cells
- **Callouts** — Warning boxes (amber), critical boxes (red), expectation boxes (blue)
- **Evidence Blocks** — Grey italic quote blocks for cited data
- **Script Boxes** — Blue-bordered dialogue boxes for meeting scripts
- **Recommendations** — Numbered items with navy badges
- **Phase Badges** — Uppercase labels for organising content into stages

See [CLAUDE.md](CLAUDE.md) for the full CSS class reference.

## Mermaid Diagrams

Mermaid diagrams in source markdown are automatically rendered to high-resolution PNG images using [mermaid-cli](https://github.com/mermaid-js/mermaid-cli) (`mmdc`), which uses the real mermaid.js library via puppeteer/chromium for pixel-perfect output. All rendering is fully local inside the Docker container — no external APIs, no data leaves your machine.

```python
from mermaid_helper import render_mermaid

diagram = render_mermaid('''
    graph TD
        A[Client] --> B[Gateway]
        B --> C[Service A]
        B --> D[Service B]
''')

CONTENT = f'<img src="{diagram}" style="width:100%;">'
```

**Supported diagram types:** flowchart, sequence, class, state, ER, pie, gantt, journey, timeline, mindmap, git graph, XY chart, quadrant, sankey, kanban, C4, block, architecture, requirement, ZenUML, packet, radar, treemap.

## How Documents Stay Private

The `documents/` directory is **gitignored by default**. Only `documents/examples/` is committed. Your content stays on your machine — the repo only contains the framework (brands, CSS, build engine).

When you create a document, the Python file and PDF output stay local. Nothing gets pushed unless you explicitly add it.

## Requirements

- Docker Desktop
- That's it. Everything else runs inside the container (auto-built on first run).

## File Structure

```
PrintingPress/
├── build.py              # Core engine — build_document()
├── build.sh              # Docker wrapper (auto-builds image on first run)
├── Dockerfile            # Docker image with WeasyPrint + mermaid-cli
├── mermaid_helper.py     # Mermaid diagram renderer (render_mermaid())
├── CLAUDE.md             # Full reference (Claude reads this automatically)
├── brands/               # Brand themes, covers, and logo assets
│   ├── reach/            # Navy cover, Reach logos
│   └── lumi/             # Gradient cover, Lumi logos, ISO/SOC2 badges
├── components/           # Shared CSS component library
├── layouts/              # A4 portrait and landscape page rules
├── documents/            # Your documents (gitignored, stays local)
│   └── examples/         # Example showcases (committed)
└── output/               # Generated HTML + PDF (gitignored)
```
