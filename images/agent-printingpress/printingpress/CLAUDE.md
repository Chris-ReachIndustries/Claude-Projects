# PrintingPress — Branded PDF Pipeline

Generate professional branded PDFs from markdown content using WeasyPrint via Docker.

## How It Works

```
Markdown → Python (HTML + CSS) → WeasyPrint (Docker) → PDF
```

Each document is a small Python file in `documents/` that imports `build_document` from `build.py`, converts markdown to HTML using the component CSS classes, and calls `build_document()` with the correct brand, layout, and classification.

## Quick Start

```bash
# Build a document
./build.sh documents/examples/sample_report.py

# Or use the /new-doc skill to create one interactively
```

## Available Brands

| Brand | Background | Watermark | Use For |
|-------|-----------|-----------|---------|
| `reach` | Navy (#041077) | REACH | Corporate docs, HR, OKRs, internal |
| `lumi` | Blue→Purple gradient | LUMI | Product docs, investor materials, technical |

## Available Layouts

| Layout | Size | Use For |
|--------|------|---------|
| `portrait` | A4 (210×297mm) | Reports, reviews, playbooks, most docs |
| `landscape` | A4 (297×210mm) | Architecture diagrams, strategy decks, wide tables |

## Classification Levels

| Key | Cover Badge | Footer Text |
|-----|------------|-------------|
| `non-confidential` | Non-Confidential | Non-Confidential |
| `internal` | Internal | Internal Use Only |
| `confidential` | Internal · Confidential | Confidential · Internal Use Only |
| `strictly-confidential` | STRICTLY CONFIDENTIAL · HR Only | STRICTLY CONFIDENTIAL · HR Distribution Only |
| `pre-nda` | Pre-NDA · External | Pre-NDA · External Distribution |

## CSS Component Classes

### Typography (`components/base.css`)
- `.h1` — 19pt, bold, navy bottom border
- `.h2` — 13pt, bold, navy blue
- `.h3` — 10pt, bold, dark grey
- `.p` — 9.5pt body text
- `.ul` — unordered list (dash bullets)
- `.ol` — ordered list (numbered)
- `.page` — forces a page break before this element
- `hr.rule` — horizontal separator

### Person Cards (`components/person-cards.css`)
- `.person-hdr` — flex container with left color border. Add color: `.red`, `.amber`, `.green`, `.blue`, `.grey`
- `.person-name` — 15pt bold name
- `.person-role` — 8pt grey role/subtitle
- `.badge` — uppercase pill badge. Variants: `.badge-red`, `.badge-amber`, `.badge-green`, `.badge-blue`, `.badge-grey`
- `.person-body` — bordered content area below header
- `.team-row` + `.team-item` + `.chip` — team member listings with role chips (`.chip-lead`, `.chip-senior`, `.chip-mid`)
- `.intro-card` — light grey bordered card

### Tables (`components/tables.css`)
- `table.table` — full-width table with navy headers
- `.td-red`, `.td-amber`, `.td-green`, `.td-grey` — status text colors
- `.td-vel-high`, `.td-vel-mid` — velocity/change indicators (green, amber)

### Callouts (`components/callouts.css`)
- `.expect` — blue left-border box for expectations/forward-looking statements
- `.expect-label` — uppercase blue label inside expect box
- `.callout` — amber left-border warning box. Add `.critical` for red variant
- `.evidence` — grey italic quote block for cited evidence

### Recommendations (`components/recommendations.css`)
- `.rec-list` + `.rec-item` — numbered recommendation items
- `.rec-num` — navy numbered badge (7×7mm)
- `.rec-body` — recommendation text
- `.gap-list` + `.gap-item` + `.gap-num` — numbered gap/issue items (red numbers)

### Scripts (`components/scripts.css`)
- `.script` — blue left-border dialogue/script box (italic)
- `.script-label` — uppercase blue label ("SAY THIS", "SUGGESTED MESSAGE")
- `.phase-badge` — navy uppercase phase badge (PHASE 1, PHASE 2, etc.)
- `.check-list` — checklist with dash markers

## Creating a Document

### 1. Create a Python file in `documents/`

```python
import sys, os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from build import build_document

CONTENT = """
<div class="page">
  <div class="h1">Section Title</div>
  <p class="p">Body text goes here.</p>
</div>
"""

build_document(
    title='My Document Title',
    subtitle='A brief description',
    brand='reach',
    layout='portrait',
    classification='confidential',
    cover_eyebrow='Document Type &middot; Q1 2026',
    cover_pills=['Tag One', 'Tag Two', 'Tag Three'],
    cover_bottom_bar=['Left Label', 'Right Label'],
    content_html=CONTENT,
    output_name='My_Document',
)
```

### 2. Build it

```bash
./build.sh documents/my_doc.py
```

### 3. Find the PDF in `output/`

## Converting Markdown to HTML

When converting markdown content to HTML for the `content_html` parameter:

- `# Heading` → `<div class="h1">Heading</div>`
- `## Heading` → `<div class="h2">Heading</div>`
- `### Heading` → `<div class="h3">Heading</div>`
- Paragraphs → `<p class="p">Text</p>`
- Bullet lists → `<ul class="ul"><li>Item</li></ul>`
- Numbered lists → `<ol class="ol"><li>Item</li></ol>`
- `---` section breaks → `<div class="page">` (new page)
- `> blockquote` → `<div class="script"><div class="script-label">Label</div>Text</div>`
- Tables → `<table class="table"><tr><th>...</th></tr><tr><td>...</td></tr></table>`
- Bold → `<strong>text</strong>`
- Italic → `<em>text</em>`
- Smart quotes: use `&ldquo;`, `&rdquo;`, `&lsquo;`, `&rsquo;`
- Em dash: use `&mdash;`
- En dash: use `&ndash;`
- Middle dot: use `&middot;`

## Embedding Images

To embed images (e.g., architecture diagrams), convert them to base64:

```python
import base64
with open('diagram.png', 'rb') as f:
    img_data = base64.b64encode(f.read()).decode()
img_src = f'data:image/png;base64,{img_data}'

# Then in content:
CONTENT = f'<img src="{img_src}" style="width:100%;">'
```

For SVG → PNG conversion, use cairosvg (available in the Docker container):
```python
import cairosvg
cairosvg.svg2png(url='diagram.svg', write_to='diagram.png', output_width=2400)
```

## Rendering Mermaid Diagrams

When source content contains mermaid diagrams (` ```mermaid ` code blocks), use the `mermaid_helper` module to pre-render them as images. WeasyPrint cannot execute JavaScript, so mermaid diagrams must be converted to images before HTML assembly. All rendering is fully local using `mermaid-cli` (mmdc) via puppeteer/chromium inside the Docker container — no external APIs.

```python
from mermaid_helper import render_mermaid

# Render a mermaid diagram to a PNG data URI
diagram_src = render_mermaid('''
    graph TD
        A[Client] --> B[Gateway]
        B --> C[Auth Service]
        B --> D[API Service]
        C --> E[(Database)]
        D --> E
''')

# Embed in content HTML
CONTENT = f'''
<div class="page">
  <div class="h1">Architecture</div>
  <img src="{diagram_src}" style="width:100%;">
</div>
'''
```

### Options
- `width` — Output width in pixels (default `2400` for print quality). Use `1200` for half-width diagrams.

### Supported Diagram Types
flowchart, sequence, class, state, ER, pie, gantt, journey, timeline, mindmap,
git graph, XY chart, quadrant, sankey, kanban, C4, block, architecture, requirement,
ZenUML, packet, radar, treemap.

## Docker Requirements

The build uses the pre-built `reach/printingpress` base image from `~/Desktop/docker-bases/`. It is always available — no first-run build needed. To rebuild: `~/Desktop/docker-bases/build-bases.sh printingpress`

System packages: libpango, libpangocairo, libcairo2, libglib2.0, libharfbuzz, libfontconfig, fonts-liberation, chromium, nodejs.
Python packages: weasyprint, cairosvg, pillow.
Mermaid rendering: mermaid-cli (mmdc) — uses real mermaid.js via puppeteer/chromium for pixel-perfect diagrams.

## File Structure

```
PrintingPress/
├── build.py          # Core engine — build_document()
├── build.sh          # Docker wrapper
├── brands/           # Brand themes (CSS + logos)
│   ├── reach/        # Navy cover, Reach logo
│   └── lumi/         # Gradient cover, Lumi logo
├── components/       # Shared CSS components
├── layouts/          # A4 portrait / landscape
├── documents/        # Your document files (gitignored except examples/)
└── output/           # Generated HTML + PDF (gitignored)
```

## Using PrintingPress from Other Projects

Documents can live in any project directory — they don't need to be inside PrintingPress.

### External document pattern

```python
from build import build_document  # PYTHONPATH set by build.sh
import os

CONTENT = """<div class="page">...</div>"""

build_document(
    title='My Report',
    brand='lumi',
    layout='portrait',
    classification='confidential',
    content_html=CONTENT,
    output_name='My_Report',
    output_dir=os.path.join(os.path.dirname(os.path.abspath(__file__)), 'output'),
)
```

### Building from another project

```bash
~/Desktop/PrintingPress/build.sh ~/Desktop/MyProject/reports/my_report.py
```

Output goes to `~/Desktop/MyProject/reports/output/` — not into PrintingPress.

### Global Claude Code skills

- `/new-doc` — Create a new document in the current project
- `/build-doc` — Build an existing document from the current project

## Important Notes

- The `documents/` directory is gitignored by default. Your content stays local.
- The `output/` directory is also gitignored.
- Only the framework (brands, components, layouts, build engine) is committed.
- If a PDF is open in a viewer, WeasyPrint can't overwrite it — close it first or use a different output name.
- Font warnings about U+0081 are cosmetic and don't affect the output.
