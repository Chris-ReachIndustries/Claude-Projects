"""Reach Industries logos — loaded from local design team assets."""
import base64, os

_ASSETS_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'assets')

def _load(filename, mime):
    path = os.path.join(_ASSETS_DIR, filename)
    with open(path, 'rb') as f:
        data = base64.b64encode(f.read()).decode()
    return f'data:{mime};base64,{data}'

def get_logo():
    """Navy logo for light backgrounds (white logo box on cover)."""
    return _load('reach-logo-navy.png', 'image/png')

def get_logo_white():
    """White logo for dark backgrounds."""
    return _load('reach-logo-white.png', 'image/png')
