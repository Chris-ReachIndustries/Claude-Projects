"""
PrintingPress — Mermaid Diagram Helper

Renders Mermaid diagram code to PNG images using mermaid-cli (mmdc),
which uses the real mermaid.js library via puppeteer/chromium for
pixel-perfect output matching the mermaid live editor.

Supported diagram types: flowchart, sequence, class, state, ER, pie, gantt,
journey, timeline, mindmap, git graph, XY chart, quadrant, sankey, kanban,
C4, block, architecture, requirement, ZenUML, packet, radar, treemap.

Usage from a document file:
    from mermaid_helper import render_mermaid

    diagram = render_mermaid('''
        graph TD
            A[Start] --> B{Decision}
            B -- Yes --> C[Done]
            B -- No --> D[Retry]
            D --> B
    ''')

    CONTENT = f'<img src="{diagram}" style="width:100%;">'
"""
import base64
import json
import os
import subprocess
import tempfile


def render_mermaid(code, width=2400, height=None):
    """
    Render a Mermaid diagram to a base64 PNG data URI.

    Args:
        code:   Mermaid diagram source code (e.g. "graph TD\\n  A --> B")
        width:  Output image width in pixels (default 2400 for print quality).
                Use 1200 for half-width diagrams.
        height: Optional output height in pixels. If not set, auto-sized.

    Returns:
        A data:image/png;base64,... URI string ready for <img src="...">.

    Raises:
        RuntimeError: If mmdc is not installed or the diagram syntax is invalid.
    """
    tmp_dir = tempfile.mkdtemp(prefix="mermaid_")
    mmd_path = os.path.join(tmp_dir, "diagram.mmd")
    png_path = os.path.join(tmp_dir, "diagram.png")
    config_path = os.path.join(tmp_dir, "config.json")
    puppet_path = os.path.join(tmp_dir, "puppeteer.json")

    try:
        # Write mermaid code to temp file
        with open(mmd_path, "w", encoding="utf-8") as f:
            f.write(code.strip() + "\n")

        # mmdc config — default light theme for PDF white backgrounds
        config = {"theme": "default"}
        with open(config_path, "w") as f:
            json.dump(config, f)

        # Puppeteer config to use system chromium
        puppet_config = {
            "executablePath": "/usr/bin/chromium",
            "args": ["--no-sandbox", "--disable-setuid-sandbox"],
        }
        with open(puppet_path, "w") as f:
            json.dump(puppet_config, f)

        # Render with mmdc
        cmd = [
            "mmdc",
            "-i", mmd_path,
            "-o", png_path,
            "-c", config_path,
            "-p", puppet_path,
            "-w", str(width),
            "-s", "4",  # scale factor for print quality
            "-b", "transparent",
        ]
        if height is not None:
            cmd.extend(["-H", str(height)])

        result = subprocess.run(cmd, capture_output=True, text=True)

        if result.returncode != 0:
            error_msg = result.stderr.strip() or result.stdout.strip()
            raise RuntimeError(
                f"mmdc failed to render diagram (exit code {result.returncode}).\n"
                f"Check your mermaid syntax.\n"
                f"Error: {error_msg}"
            )

        if not os.path.exists(png_path):
            raise RuntimeError(
                "mmdc did not produce output. Check your mermaid syntax."
            )

        with open(png_path, "rb") as f:
            png_data = f.read()

        b64 = base64.b64encode(png_data).decode("ascii")
        return f"data:image/png;base64,{b64}"

    except FileNotFoundError:
        raise RuntimeError(
            "mmdc binary not found. Run builds via ./build.sh to use the "
            "Docker container where mermaid-cli is installed."
        )
    finally:
        # Clean up temp files
        for p in [mmd_path, png_path, config_path, puppet_path]:
            if os.path.exists(p):
                os.remove(p)
        if os.path.exists(tmp_dir):
            os.rmdir(tmp_dir)
