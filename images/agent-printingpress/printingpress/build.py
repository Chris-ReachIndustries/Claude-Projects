"""
PrintingPress — Core Build Engine

Assembles CSS (brand + layout + components) and HTML content into a
complete document, then writes it to output/ for WeasyPrint conversion.

Usage from a document file:
    from build import build_document
    build_document(title="My Doc", ..., content_html=CONTENT)
"""
import os, sys, importlib, datetime

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
TODAY = datetime.date.today().strftime("%B %Y")

# ─── Classification presets ───────────────────────────────────────────────────

CLASSIFICATIONS = {
    'internal': {
        'badge': 'Internal',
        'footer': 'Internal Use Only',
        'footer_html': 'Internal Use Only',
    },
    'confidential': {
        'badge': 'Internal &middot; Confidential',
        'footer': 'Confidential \\B7  Internal Use Only',
        'footer_html': 'Confidential &middot; Internal Use Only',
    },
    'strictly-confidential': {
        'badge': 'STRICTLY CONFIDENTIAL &middot; HR Only',
        'footer': 'STRICTLY CONFIDENTIAL \\B7  HR Distribution Only',
        'footer_html': 'STRICTLY CONFIDENTIAL &middot; HR Distribution Only',
    },
    'pre-nda': {
        'badge': 'Pre-NDA &middot; External',
        'footer': 'Pre-NDA \\B7  External Distribution',
        'footer_html': 'Pre-NDA &middot; External Distribution',
    },
    'non-confidential': {
        'badge': 'Non-Confidential',
        'footer': 'Non-Confidential',
        'footer_html': 'Non-Confidential',
    },
}

# ─── Brand config ─────────────────────────────────────────────────────────────

BRAND_CONFIG = {
    'reach': {
        'company_name': 'Reach Industries',
        'watermark': 'REACH',
        'website': 'reach.industries',
        'logo_module': 'brands.reach.logo',
        'cover_css': 'brands/reach/cover.css',
        'theme_css': None,  # Reach uses component CSS directly
    },
    'lumi': {
        'company_name': 'Lumi',
        'watermark': 'LUMI',
        'website': 'lumi.systems',
        'logo_module': 'brands.lumi.logo',
        'cover_css': 'brands/lumi/cover.css',
        'theme_css': 'brands/lumi/theme.css',  # Lumi has its own component styles
    },
    'personal': {
        'company_name': '',
        'watermark': '',
        'website': '',
        'logo_module': 'brands.personal.logo',
        'cover_css': 'brands/personal/cover.css',
        'theme_css': None,
    },
    'plain': {
        'company_name': '',
        'watermark': '',
        'website': '',
        'logo_module': 'brands.plain.logo',
        'cover_css': 'brands/plain/cover.css',
        'theme_css': None,
    },
}

# ─── CSS loading ──────────────────────────────────────────────────────────────

def _read_css(relative_path):
    """Read a CSS file relative to the PrintingPress root."""
    full = os.path.join(BASE_DIR, relative_path)
    with open(full, 'r') as f:
        return f.read()

def _load_all_components():
    """Load all CSS files from the components/ directory."""
    comp_dir = os.path.join(BASE_DIR, 'components')
    parts = []
    for fname in sorted(os.listdir(comp_dir)):
        if fname.endswith('.css'):
            parts.append(_read_css(f'components/{fname}'))
    return '\n'.join(parts)

def _load_components(names):
    """Load specific component CSS files by name (without .css extension)."""
    parts = []
    for name in names:
        parts.append(_read_css(f'components/{name}.css'))
    return '\n'.join(parts)

# ─── Logo loading ─────────────────────────────────────────────────────────────

_asset_cache = {}

def _get_logo(brand):
    """Fetch and cache the brand logo."""
    key = f'{brand}_logo'
    if key not in _asset_cache:
        config = BRAND_CONFIG[brand]
        if BASE_DIR not in sys.path:
            sys.path.insert(0, BASE_DIR)
        mod = importlib.import_module(config['logo_module'])
        _asset_cache[key] = mod.get_logo()
    return _asset_cache[key]

def _get_logo_white(brand):
    """Fetch and cache the white variant of a brand logo (for dark backgrounds)."""
    key = f'{brand}_logo_white'
    if key not in _asset_cache:
        config = BRAND_CONFIG[brand]
        if BASE_DIR not in sys.path:
            sys.path.insert(0, BASE_DIR)
        mod = importlib.import_module(config['logo_module'])
        _asset_cache[key] = mod.get_logo_white() if hasattr(mod, 'get_logo_white') else None
    return _asset_cache[key]

def _get_lumi_badges():
    """Fetch and cache ISO and SOC2 badges for Lumi brand."""
    if 'iso_badge' not in _asset_cache:
        if BASE_DIR not in sys.path:
            sys.path.insert(0, BASE_DIR)
        mod = importlib.import_module('brands.lumi.logo')
        _asset_cache['iso_badge'] = mod.get_iso_badge()
        _asset_cache['soc2_badge'] = mod.get_soc2_badge()
    return _asset_cache.get('iso_badge'), _asset_cache.get('soc2_badge')

# ─── Page rules generation ────────────────────────────────────────────────────

def _page_rules_css_reach(title, classification, date_str):
    """Generate @page content rules for Reach brand."""
    cls = CLASSIFICATIONS.get(classification, CLASSIFICATIONS['confidential'])
    footer_text = f"{cls['footer']}  \\B7  {date_str}"
    return f"""
@page {{
  @top-right {{
    content: "{title}";
    font-family: Inter, Helvetica, sans-serif;
    font-size: 6.5pt;
    color: #9ca3af;
    vertical-align: middle;
  }}
  @bottom-left {{
    content: "{footer_text}";
    font-family: Inter, Helvetica, sans-serif;
    font-size: 6.5pt;
    color: #d1d5db;
    text-transform: uppercase;
    letter-spacing: 0.5pt;
    vertical-align: middle;
  }}
}}
"""

def _page_rules_css_lumi(classification):
    """Generate @page content rules for Lumi brand."""
    cls = CLASSIFICATIONS.get(classification, CLASSIFICATIONS['confidential'])
    return f"""
@page {{
  size: A4;
  margin: 18mm 20mm 22mm 20mm;
  @top-left {{
    content: element(running-logo);
    vertical-align: middle;
  }}
  @top-right {{
    content: "Lumi AI Platform  \\2022  {cls['footer']}";
    font-family: Inter, sans-serif;
    font-size: 7.5pt;
    color: #9ca3af;
    vertical-align: middle;
  }}
  @bottom-center {{
    content: "Page " counter(page) " of " counter(pages);
    font-family: Inter, sans-serif;
    font-size: 7.5pt;
    color: #9ca3af;
  }}
  @bottom-right {{ content: none; }}
  @bottom-left {{ content: none; }}
}}
@page :first {{
  margin: 0;
  @top-left {{ content: none; }}
  @top-right {{ content: none; }}
  @bottom-center {{ content: none; }}
}}
@page back-cover {{
  margin: 0;
  @top-left {{ content: none; }}
  @top-right {{ content: none; }}
  @bottom-center {{ content: none; }}
}}
#running-logo {{ position: running(running-logo); }}
#running-logo img {{ height: 16pt; }}
"""

# ─── Cover page HTML — Personal ──────────────────────────────────────────────

def _cover_html_personal(title, subtitle, classification, cover_eyebrow,
                         cover_pills, cover_bottom_bar):
    """Generate a clean, unbranded personal/family document cover page."""
    cls = CLASSIFICATIONS.get(classification, CLASSIFICATIONS['non-confidential'])

    pills_html = ''
    if cover_pills:
        pill_items = [f'<span class="cover-pill">{p}</span>' for p in cover_pills]
        pills_html = f'<div class="cover-pills">{"".join(pill_items)}</div>'

    bar_html = ''
    if cover_bottom_bar:
        bar_a = cover_bottom_bar[0] if len(cover_bottom_bar) > 0 else ''
        bar_b = cover_bottom_bar[1] if len(cover_bottom_bar) > 1 else ''
        bar_html = f"""<div class="cover-bottom-bar">
    <div class="cover-bar-a">{bar_a}</div>
    <div class="cover-bar-b">{bar_b}</div>
  </div>"""

    eyebrow_html = f'<div class="cover-eyebrow">{cover_eyebrow}</div>' if cover_eyebrow else ''

    return f"""
<div class="cover">
  <div class="cover-top">
    <div class="cover-badge">{cls['badge']}</div>
  </div>
  <div class="cover-main">
    {eyebrow_html}
    <div class="cover-title">{title}</div>
    <div class="cover-subtitle">{subtitle}</div>
    {pills_html}
  </div>
  <div class="cover-divider"></div>
  {bar_html}
  <div class="cover-footer">
    <div class="cover-footer-left">{TODAY}</div>
    <div class="cover-footer-right">{title}</div>
  </div>
</div>
""", ''  # No back cover for personal brand


def _page_rules_css_personal(title, date_str):
    """Generate @page content rules for personal brand."""
    return f"""
@page {{
  @top-right {{
    content: "{title}";
    font-family: Inter, Helvetica, sans-serif;
    font-size: 6.5pt;
    color: #9c8060;
    vertical-align: middle;
  }}
  @bottom-left {{
    content: "{title}  \\B7  {date_str}";
    font-family: Inter, Helvetica, sans-serif;
    font-size: 6.5pt;
    color: #c9a96e;
    text-transform: uppercase;
    letter-spacing: 0.5pt;
    vertical-align: middle;
  }}
  @bottom-right {{
    content: "Page " counter(page);
    font-family: Inter, Helvetica, sans-serif;
    font-size: 6.5pt;
    color: #9c8060;
    vertical-align: middle;
  }}
}}
"""


# ─── Cover page HTML — Plain ─────────────────────────────────────────────────

def _cover_html_plain(title, subtitle, classification, cover_eyebrow,
                      cover_pills, cover_bottom_bar):
    """Generate a clean, neutral, unbranded cover page."""
    cls = CLASSIFICATIONS.get(classification, CLASSIFICATIONS['non-confidential'])

    pills_html = ''
    if cover_pills:
        pill_items = [f'<span class="cover-pill">{p}</span>' for p in cover_pills]
        pills_html = f'<div class="cover-pills">{"".join(pill_items)}</div>'

    bar_html = ''
    if cover_bottom_bar:
        bar_a = cover_bottom_bar[0] if len(cover_bottom_bar) > 0 else ''
        bar_b = cover_bottom_bar[1] if len(cover_bottom_bar) > 1 else ''
        bar_html = f"""<div class="cover-bottom-bar">
    <div class="cover-bar-a">{bar_a}</div>
    <div class="cover-bar-b">{bar_b}</div>
  </div>"""

    eyebrow_html = f'<div class="cover-eyebrow">{cover_eyebrow}</div>' if cover_eyebrow else ''

    return f"""
<div class="cover">
  <div class="cover-accent"></div>
  <div class="cover-top">
    <div class="cover-badge">{cls['badge']}</div>
  </div>
  <div class="cover-main">
    {eyebrow_html}
    <div class="cover-title">{title}</div>
    <div class="cover-subtitle">{subtitle}</div>
    {pills_html}
  </div>
  <div class="cover-divider"></div>
  {bar_html}
  <div class="cover-footer">
    <div class="cover-footer-left">{TODAY}</div>
    <div class="cover-footer-right">{title}</div>
  </div>
</div>
""", ''  # No back cover for plain brand


def _page_rules_css_plain(title, date_str):
    """Generate @page content rules for plain brand."""
    return f"""
@page {{
  @top-right {{
    content: "{title}";
    font-family: Inter, Helvetica, sans-serif;
    font-size: 6.5pt;
    color: #9ca3af;
    vertical-align: middle;
  }}
  @bottom-left {{
    content: "{title}  \\B7  {date_str}";
    font-family: Inter, Helvetica, sans-serif;
    font-size: 6.5pt;
    color: #d1d5db;
    text-transform: uppercase;
    letter-spacing: 0.5pt;
    vertical-align: middle;
  }}
  @bottom-right {{
    content: "Page " counter(page);
    font-family: Inter, Helvetica, sans-serif;
    font-size: 6.5pt;
    color: #9ca3af;
    vertical-align: middle;
  }}
}}
"""


# ─── Cover page HTML — Reach ─────────────────────────────────────────────────

def _cover_html_reach(title, subtitle, classification, cover_eyebrow,
                      cover_pills, cover_bottom_bar):
    """Generate the Reach Industries cover page HTML."""
    config = BRAND_CONFIG['reach']
    cls = CLASSIFICATIONS.get(classification, CLASSIFICATIONS['confidential'])
    logo_src = _get_logo('reach')

    cover_logo = f'<img src="{logo_src}" alt="Reach Industries">' if logo_src else ''
    header_logo = f'<img src="{logo_src}" alt="Reach Industries">' if logo_src else '<strong>Reach Industries</strong>'

    pills_html = ''
    if cover_pills:
        pill_items = []
        for i, pill in enumerate(cover_pills):
            pill_items.append(f'<span class="cover-pill pill-{i+1}">{pill}</span>')
        pills_html = f'<div class="cover-pills">{"".join(pill_items)}</div>'

    bar_html = ''
    if cover_bottom_bar:
        bar_a = cover_bottom_bar[0] if len(cover_bottom_bar) > 0 else ''
        bar_b = cover_bottom_bar[1] if len(cover_bottom_bar) > 1 else ''
        bar_html = f"""<div class="cover-bottom-bar">
    <div class="cover-bar-a">{bar_a}</div>
    <div class="cover-bar-b">{bar_b}</div>
  </div>"""

    return f"""
<div id="header-logo">{header_logo}</div>
<div class="cover">
  <div class="cover-bg">{config['watermark']}</div>
  <div class="cover-top">
    <div class="cover-brand">
      <div class="cover-logo-box">{cover_logo}</div>
      <div class="cover-company">{config['company_name']}</div>
    </div>
    <div class="cover-badge">{cls['badge']}</div>
  </div>
  <div class="cover-stripe"></div>
  <div class="cover-main">
    <div class="cover-eyebrow">{cover_eyebrow}</div>
    <div class="cover-title">{title}</div>
    <div class="cover-subtitle">{subtitle}</div>
    {pills_html}
  </div>
  {bar_html}
  <div class="cover-footer">
    <div class="cover-footer-left">{config['company_name']} &nbsp;&middot;&nbsp; {cls['footer_html']} &nbsp;&middot;&nbsp; 2026</div>
    <div class="cover-footer-right">{config['website']}</div>
  </div>
</div>
""", ''  # No back cover for Reach

# ─── Cover page HTML — Lumi ──────────────────────────────────────────────────

def _cover_html_lumi(title, subtitle, classification, doc_type, toc_entries):
    """Generate the Lumi cover page, optional TOC, and back cover HTML."""
    cls = CLASSIFICATIONS.get(classification, CLASSIFICATIONS['confidential'])
    logo_src = _get_logo('lumi')
    logo_white_src = _get_logo_white('lumi')
    reach_logo_src = _get_logo('reach')
    iso_badge, soc2_badge = _get_lumi_badges()

    logo_img = f'<img src="{logo_src}" alt="Lumi">' if logo_src else '<strong>Lumi</strong>'
    reach_img = f'<img src="{reach_logo_src}" alt="Reach Industries">' if reach_logo_src else ''
    iso_img = f'<img src="{iso_badge}" alt="ISO 27001:2022">' if iso_badge else ''
    soc2_img = f'<img src="{soc2_badge}" alt="SOC 2">' if soc2_badge else ''
    # Use white logo on gradient header, fall back to regular
    cover_logo_src = logo_white_src or logo_src

    # Cover
    cover = f"""
<div id="running-logo">{logo_img}</div>
<div class="cover">
  <div class="cover-top">
    <img src="{cover_logo_src}" class="cover-logo" alt="Lumi">
    <h1>{title}</h1>
    <p class="cover-subtitle">{subtitle}</p>
  </div>
  <div class="cover-meta">
    <p><strong>Document type:</strong> {doc_type}</p>
    <p><strong>Date:</strong> {TODAY}</p>
    <p><strong>Classification:</strong> {cls['footer']}</p>
  </div>
  <div class="cover-footer">
    <div style="display:flex;gap:8mm;align-items:center;">
      {logo_img}
      {reach_img}
    </div>
    <div class="cover-badges">
      {iso_img}
      {soc2_img}
    </div>
  </div>
</div>
"""

    # TOC (optional)
    toc = ''
    if toc_entries:
        toc_items = '\n'.join([
            f'<div class="toc-entry"><span class="toc-num">{n}.</span>'
            f'<a class="toc-title" href="#{anchor}">{label}</a>'
            f'<span class="toc-dots"></span>'
            f'<a class="toc-page-num" href="#{anchor}"></a></div>'
            for n, anchor, label in toc_entries
        ])
        toc = f"""
<div class="toc-page">
  <h2>Contents</h2>
  <div class="toc-group">Sections</div>
  {toc_items}
</div>
"""

    # Back cover
    back = f"""
<div class="back-cover">
  <img src="{logo_src}" alt="Lumi">
  <p style="font-size:11pt;font-weight:600;color:#111827;margin:2mm 0;">Reach Industries</p>
  <p>lumi.systems</p>
  <div class="badges">{iso_img}{soc2_img}</div>
  <p>{cls['footer']} &bull; {TODAY}</p>
</div>
"""

    return cover + toc, back

# ─── Main build function ─────────────────────────────────────────────────────

def build_document(
    title,
    subtitle='',
    brand='reach',
    layout='portrait',
    classification='confidential',
    cover_eyebrow='',
    cover_pills=None,
    cover_bottom_bar=None,
    content_html='',
    components=None,
    extra_css='',
    output_name='document',
    output_dir=None,
    date_str='March 2026',
    # Lumi-specific options
    doc_type='',
    toc_entries=None,
):
    """
    Assemble a complete HTML document from brand, layout, components, and content.

    Args:
        title:            Document title (cover + header)
        subtitle:         Cover subtitle text
        brand:            'reach' or 'lumi'
        layout:           'portrait' or 'landscape'
        classification:   'internal', 'confidential', 'strictly-confidential', 'pre-nda', 'non-confidential'
        cover_eyebrow:    Small text above the title on cover (Reach only)
        cover_pills:      List of 1-3 pill labels for the cover (Reach only)
        cover_bottom_bar: List of [left_text, right_text] for bottom bar (Reach only)
        content_html:     The HTML body content (pages)
        components:       List of component names to include, or None for all
        extra_css:        Additional CSS appended after everything else
        output_name:      Output filename (without extension)
        output_dir:       Output directory (default: PrintingPress/output/)
        date_str:         Date string for footer (default: 'March 2026')
        doc_type:         Document type label for cover metadata (Lumi only)
        toc_entries:      List of (num, anchor, label) for TOC (Lumi only)

    Returns:
        Path to the generated HTML file.
    """
    if brand not in BRAND_CONFIG:
        raise ValueError(f"Unknown brand '{brand}'. Available: {list(BRAND_CONFIG.keys())}")

    config = BRAND_CONFIG[brand]

    # 1. Assemble CSS — @import must be first
    css_parts = []
    css_parts.append("@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap');")
    css_parts.append(_read_css(f'layouts/{layout}.css'))
    css_parts.append(_read_css(config['cover_css']))

    # Load brand theme CSS if it exists (Lumi has its own component styles)
    if config.get('theme_css'):
        css_parts.append(_read_css(config['theme_css']))

    if components:
        css_parts.append(_load_components(components))
    else:
        css_parts.append(_load_all_components())

    # Brand-specific page rules
    if brand == 'lumi':
        css_parts.append(_page_rules_css_lumi(classification))
    elif brand == 'personal':
        css_parts.append(_page_rules_css_personal(title, date_str))
    elif brand == 'plain':
        css_parts.append(_page_rules_css_plain(title, date_str))
    else:
        css_parts.append(_page_rules_css_reach(title, classification, date_str))

    if extra_css:
        css_parts.append(extra_css)
    combined_css = '\n'.join(css_parts)

    # 2. Generate cover (and optional back cover)
    back_cover = ''
    if brand == 'lumi':
        cover, back_cover = _cover_html_lumi(title, subtitle, classification,
                                              doc_type or 'Document', toc_entries)
    elif brand == 'personal':
        cover, back_cover = _cover_html_personal(title, subtitle, classification,
                                                  cover_eyebrow, cover_pills, cover_bottom_bar)
    elif brand == 'plain':
        cover, back_cover = _cover_html_plain(title, subtitle, classification,
                                               cover_eyebrow, cover_pills, cover_bottom_bar)
    else:
        cover, back_cover = _cover_html_reach(title, subtitle, classification,
                                               cover_eyebrow, cover_pills, cover_bottom_bar)

    # 3. Assemble full HTML
    html = f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{config['company_name']} &mdash; {title}</title>
<style>
{combined_css}
</style>
</head>
<body>
{cover}
<!-- CONTENT -->
{content_html}
{back_cover}
</body>
</html>"""

    # 4. Write output
    if output_dir is None:
        output_dir = os.path.join(BASE_DIR, 'output')
    os.makedirs(output_dir, exist_ok=True)
    output_path = os.path.join(output_dir, f'{output_name}.html')
    with open(output_path, 'w', encoding='utf-8') as f:
        f.write(html)

    size_kb = len(html) // 1024
    print(f"  {output_name}.html: {size_kb}KB")

    # Write a marker file so build.sh knows which HTML to convert
    marker = os.path.join(output_dir, '.last_build')
    with open(marker, 'w') as f:
        f.write(output_path)

    return output_path
