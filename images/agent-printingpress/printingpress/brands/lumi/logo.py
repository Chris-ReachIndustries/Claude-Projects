"""Lumi logos and compliance badges — loaded from local design team assets."""
import base64, os

_ASSETS_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'assets')

def _load(filename, mime):
    path = os.path.join(_ASSETS_DIR, filename)
    with open(path, 'rb') as f:
        data = base64.b64encode(f.read()).decode()
    return f'data:{mime};base64,{data}'

def get_logo():
    """Black logo for light backgrounds (running header, cover footer, back cover)."""
    return _load('lumi-logo-black.png', 'image/png')

def get_logo_white():
    """White logo for the gradient cover header."""
    return _load('lumi-logo-white.png', 'image/png')

def get_logo_colour():
    """Full colour logo (SVG)."""
    return _load('lumi-logo-colour.svg', 'image/svg+xml')

def get_iso_badge():
    """ISO 27001:2022 badge."""
    return _load('iso-badge.svg', 'image/svg+xml')

def get_soc2_badge():
    """SOC 2 badge."""
    return _load('soc2-badge.webp', 'image/webp')
