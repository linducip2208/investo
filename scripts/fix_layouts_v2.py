#!/usr/bin/env python3
"""
Fix_layouts v2 — Transform all Investo HTML templates to consistent responsive layout.
Fixes: sidebarOpen removal, detection for all sidebar patterns, double-transform safety.
"""

import re
import os
import glob

TEMPLATE_DIR = os.path.join(os.path.dirname(os.path.dirname(__file__)), 'web', 'templates')

SIDEBAR_HTML = '''    <aside class="w-56 bg-[#0b1120] border-r border-slate-700/50 flex flex-col min-h-screen shrink-0 max-md:hidden" id="sidebar">
        <div class="flex items-center gap-2 px-4 h-14 border-b border-slate-700/50">
            <span class="w-7 h-7 bg-gradient-to-br from-blue-600 to-blue-800 rounded flex items-center justify-center text-white font-bold text-xs">I</span>
            <span class="font-bold text-white">Investo</span>
        </div>
        <nav class="flex-1 p-3 space-y-0.5 overflow-y-auto">
            <a href="/dashboard" class="nav-link">📊 Dashboard</a>
            <a href="/saham" class="nav-link">📈 Saham</a>
            <a href="/forex" class="nav-link">💱 Forex</a>
            <a href="/screener" class="nav-link">🔍 Screener</a>
            <a href="/ai/signal-center" class="nav-link">📡 Signals</a>
            <div class="border-t border-slate-700/50 my-2"></div>
            <p class="px-3 text-[10px] font-semibold text-slate-600 uppercase tracking-widest mb-1">Market</p>
            <a href="/market/heatmap" class="nav-link">🗺️ Heatmap</a>
            <a href="/market/recap" class="nav-link">📋 Recap</a>
            <a href="/market/fear-greed" class="nav-link">😨 Fear & Greed</a>
            <a href="/market/economic-calendar" class="nav-link">📅 Calendar</a>
            <a href="/market/pattern-scan" class="nav-link">🔬 Patterns</a>
            <a href="/market/live-signals" class="nav-link">⚡ Signals</a>
            <a href="/market/bandarmologi" class="nav-link">🕵️ Bandarmologi</a>
            <a href="/market/foreign-flow" class="nav-link">🌏 Foreign Flow</a>
            <a href="/market/ipo" class="nav-link">🚀 IPO</a>
            <a href="/market/waran" class="nav-link">🎫 Waran</a>
            <a href="/market/macro-dashboard" class="nav-link">🏛️ Macro</a>
            <div class="border-t border-slate-700/50 my-2"></div>
            <p class="px-3 text-[10px] font-semibold text-slate-600 uppercase tracking-widest mb-1">AI</p>
            <a href="/ai/coach" class="nav-link">💬 Coach</a>
            <a href="/ai/agents" class="nav-link">🧠 Agents</a>
            <a href="/ai/briefing" class="nav-link">📰 Briefing</a>
            <a href="/ai/compare" class="nav-link">⚖️ Compare</a>
            <a href="/ai/thesis" class="nav-link">📝 Thesis</a>
            <div class="border-t border-slate-700/50 my-2"></div>
            <a href="/dashboard/portfolios" class="nav-link">💼 Portofolio</a>
            <a href="/dashboard/watchlists" class="nav-link">⭐ Watchlist</a>
            <a href="/dashboard/alerts" class="nav-link">🔔 Alert</a>
            <a href="/ideas" class="nav-link">💡 Ideas</a>
            <a href="/leaderboard" class="nav-link">🏆 Leaderboard</a>
            <a href="/pengaturan" class="nav-link">⚙️ Pengaturan</a>
            <a href="/logout" class="nav-link text-red-400">🚪 Keluar</a>
        </nav>
    </aside>

    <div id="sidebar-overlay" class="fixed inset-0 z-50 bg-black/60 hidden max-md:block" onclick="document.getElementById('sidebar').classList.toggle('max-md:hidden');this.classList.toggle('hidden')"></div>'''

MOBILE_HEADER = '''        <header class="flex items-center h-14 px-4 bg-[#0b1120] border-b border-slate-700/50 md:hidden">
            <button onclick="document.getElementById('sidebar').classList.toggle('max-md:hidden');document.getElementById('sidebar-overlay').classList.toggle('hidden')" class="p-2 text-slate-400">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/></svg>
            </button>
            <span class="font-bold text-white ml-2">Investo</span>
        </header>'''

NAV_LINK_CSS = '.nav-link{display:flex;align-items:center;gap:8px;padding:7px 12px;border-radius:8px;color:#94a3b8;font-size:13px;font-weight:500;text-decoration:none;transition:all .15s}.nav-link:hover{background:#1e293b;color:#e2e8f0}'

SKIP_DIRS = {'components', 'auth', 'layouts', 'admin'}

# ---- Helper: already transformed? ----
def already_transformed(content):
    return 'max-md:hidden' in content and 'id="sidebar"' in content and 'sidebar-overlay' in content

# ---- Helper: has old sidebar? ----
def has_old_sidebar(content):
    """Detect any old sidebar pattern that needs transformation."""
    # Pattern 1: class="sidebar" on aside
    if re.search(r'<aside[^>]*class="[^"]*\bsidebar\b', content, re.IGNORECASE):
        return True
    # Pattern 2: x-data on aside with sidebar refs
    if re.search(r'<aside\s+x-data\b', content, re.IGNORECASE):
        return True
    # Pattern 3: x-show="sidebarOpen" on aside
    if re.search(r'<aside[^>]*x-show\s*=\s*"sidebarOpen"', content, re.IGNORECASE):
        return True
    # Pattern 4: :class with open/sidebarOpen
    if re.search(r'<aside[^>]*:class\s*=\s*"\s*\{\s*[\"\\]open[\"\\]', content, re.IGNORECASE):
        if 'sidebarOpen' in content:
            return True
    return False

# ---- Extract x-data/x-init from body ----
def get_body_alpine(content):
    m = re.search(r'<body\b([^>]*)>', content, re.IGNORECASE | re.DOTALL)
    if not m:
        return '', ''
    attrs = m.group(1)
    xd = re.search(r'x-data\s*=\s*"([^"]*)"', attrs)
    xi = re.search(r'x-init\s*=\s*"([^"]*)"', attrs)
    return (xd.group(1) if xd else ''), (xi.group(1) if xi else '')

# ---- Clean sidebarOpen from Alpine data ----
def clean_alpine_data(data_str):
    """Remove sidebarOpen property entirely from an Alpine data string."""
    if not data_str:
        return ''
    # Remove sidebarOpen: value, (with optional leading whitespace)
    data_str = re.sub(r'\s*sidebarOpen\s*:\s*(?:true|false|!sidebarOpen)\s*,?\s*', '', data_str)
    # Double commas
    data_str = re.sub(r',\s*,', ',', data_str)
    # Leading comma after {
    data_str = re.sub(r'\{\s*,', '{', data_str)
    # Trailing comma before }
    data_str = re.sub(r',\s*\}', '}', data_str)
    data_str = data_str.strip()
    return data_str

# ---- Clean sidebarOpen from raw JavaScript ----
def clean_script_sidebaropen(script_text):
    """Remove sidebarOpen property from inline JavaScript function."""
    # Remove: sidebarOpen: false,
    script_text = re.sub(r'\s*sidebarOpen\s*:\s*(?:true|false|!sidebarOpen)\s*,', '', script_text)
    # Remove: sidebarOpen: false (last property before })
    script_text = re.sub(r'\s*sidebarOpen\s*:\s*(?:true|false|!sidebarOpen)\s*', '', script_text)
    # Clean up empty objects like return { }
    script_text = re.sub(r'return\s*\{\s*\}', 'return {}', script_text)
    # Fix double commas
    script_text = re.sub(r',\s*,', ',', script_text)
    return script_text

# ---- Find page content region ----
def find_page_content(body_content):
    """Find the page content between the old sidebar and </body>.
    Returns (before_old_sidebar, old_sidebar_block, after_sidebar, main_inner, bottom_scripts)."""

    # Find the old <aside> block
    aside_start = re.search(r'<aside\b', body_content, re.IGNORECASE)
    if not aside_start:
        return None, None, None, None, None

    before_aside = body_content[:aside_start.start()]

    # Find matching </aside> by counting
    depth = 1
    pos = aside_start.end()
    while depth > 0:
        open_m = re.search(r'<aside\b', body_content[pos:], re.IGNORECASE)
        close_m = re.search(r'</aside>', body_content[pos:], re.IGNORECASE)
        if not close_m:
            return None, None, None, None, None
        if open_m and open_m.start() < close_m.start():
            depth += 1
            pos += open_m.end()
        else:
            depth -= 1
            if depth == 0:
                pos += close_m.end()
                break
            pos += close_m.end()

    aside_block = body_content[aside_start.start():pos]
    after_aside = body_content[pos:]

    # Now find the page content in after_aside
    # Strategy: find <main> tag or <div class="main-content"> or just content divs
    # We want to extract inner content (after removing old topbar/header)

    # Find a <main> tag
    main_m = re.search(r'<main\b[^>]*>', after_aside, re.IGNORECASE)
    main_inner = None
    bottom_scripts = ''

    if main_m:
        # Find matching </main>
        main_content_start = main_m.end()
        # Find all </main>
        all_close = list(re.finditer(r'</main>', after_aside, re.IGNORECASE))
        if all_close:
            main_close = all_close[-1].end()
            main_inner = after_aside[main_content_start:main_close - len('</main>')]
            bottom_scripts = after_aside[main_close:]
    else:
        # Fallback: find <div class="main-content">
        div_m = re.search(r'<div\s+class="main-content"[^>]*>', after_aside, re.IGNORECASE)
        if div_m:
            content_start = div_m.end()
            # Find matching </div>
            depth = 1
            pos = content_start
            end_pos = None
            while depth > 0:
                open_d = re.search(r'<div\b', after_aside[pos:], re.IGNORECASE)
                close_d = re.search(r'</div>', after_aside[pos:], re.IGNORECASE)
                if not close_d:
                    break
                if open_d and open_d.start() < close_d.start():
                    depth += 1
                    pos += open_d.end()
                else:
                    depth -= 1
                    if depth == 0:
                        end_pos = pos + close_d.end()
                        break
                    pos += close_d.end()
            if end_pos:
                main_inner = after_aside[content_start:end_pos - len('</div>')]
                bottom_scripts = after_aside[end_pos:]
        else:
            # Last resort: take everything after <aside> up to </body>
            body_end = after_aside.rfind('</body>')
            if body_end > 0:
                main_inner = after_aside[:body_end]
                bottom_scripts = after_aside[body_end:]

    return before_aside, aside_block, after_aside, main_inner, bottom_scripts

# ---- Rebuild file ----
def transform_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    # Already transformed?
    if already_transformed(content):
        return 'already'

    # Get head section
    head_end = content.find('</head>')
    if head_end == -1:
        return 'error'

    head_html = content[:head_end + len('</head>')]
    # Clean h-full from html tag
    head_html = re.sub(r'class="dark\s+h-full"', 'class="dark"', head_html)
    head_html = re.sub(r'class="dark h-full"', 'class="dark"', head_html)
    head_html = re.sub(r'class="h-full"', '', head_html)
    head_html = re.sub(r'class=["\']h-full\s+dark["\']', 'class="dark"', head_html)

    # Get the <head> inner content
    head_inner_m = re.search(r'<head>(.*?)</head>', head_html, re.DOTALL)
    head_inner = head_inner_m.group(1) if head_inner_m else ''

    body_from_head = content[head_end + len('</head>'):]

    # Get x-data / x-init
    xdata, xinit = get_body_alpine(body_from_head)
    xdata_clean = clean_alpine_data(xdata)

    # Extract inline script (first <script> in body, typically the Alpine function)
    body_start_in_content = body_from_head.find('<body')
    script_block = ''
    if body_start_in_content != -1:
        script_m = re.search(r'<script\b[^>]*>(.*?)</script>', body_from_head[body_start_in_content:], re.DOTALL)
        if script_m:
            script_block = script_m.group(0)
            # Clean sidebarOpen from script
            script_inner = clean_script_sidebaropen(script_m.group(1))
            script_block = f'<script>{script_inner}</script>'

    # Find page content
    body_content = body_from_head  # includes <body> tag
    before, old_sidebar, after_sidebar, main_inner, bottom_scripts = find_page_content(body_content)

    if main_inner is None:
        return 'no_content'

    # Clean main_inner: strip leading whitespace, remove <!-- CONTENT --> comments
    main_inner = main_inner.strip()
    # Remove <!-- CONTENT --> comment if present
    main_inner = re.sub(r'<!--\s*CONTENT\s*-->\s*', '', main_inner, count=1)
    # Remove <!-- MAIN --> comment if present
    main_inner = re.sub(r'<!--\s*MAIN\s*-->\s*', '', main_inner, count=1)

    # Build x-data attribute
    if xdata_clean:
        xdata_attr = f' x-data="{xdata_clean}"'
        if xinit:
            xdata_attr += f' x-init="{xinit}"'
    elif xinit:
        xdata_attr = f' x-init="{xinit}"'
    else:
        xdata_attr = ''

    # Add nav-link CSS to head inner (append to </style>)
    head_inner = re.sub(r'(</style>)', f'        {NAV_LINK_CSS}\n\\1', head_inner, count=1)

    # Clean up old CSS classes from head_inner
    to_remove = [
        r'\.sidebar-link\s*\{[^}]*\}\s*',
        r'\.sidebar-link\s*:hover\s*\{[^}]*\}\s*',
        r'\.sidebar-link\.active\s*\{[^}]*\}\s*',
        r'\.sidebar-link\s*\.icon\s*\{[^}]*\}\s*',
        r'@media\s*\(\s*max-width\s*:\s*102[34]px\s*\)\s*\{\s*\.sidebar\s*\{[^}]+\}\s*\.sidebar\.open\s*\{[^}]+\}\s*\}',
        r'@media\s*\(\s*max-width\s*:\s*102[34]px\s*\)\s*\{\s*\.sidebar\s*\{[^}]+\}\s*\}',
        r'\.topbar\s*\{[^}]*\}\s*',
        r'\.main-content\s*\{[^}]*\}\s*',
    ]
    for pattern in to_remove:
        head_inner = re.sub(pattern, '', head_inner, flags=re.DOTALL)

    # Cleanup bottom scripts - remove the old mobile overlay if present
    bottom_scripts = re.sub(r'<div\s+x-show\s*=\s*"sidebarOpen"[^>]*>(?:(?!</div>).)*?</div>\s*', '', bottom_scripts, flags=re.DOTALL)
    bottom_scripts = re.sub(r'<div\s+@click\s*=\s*"sidebarOpen\s*=\s*false"[^>]*>(?:(?!</div>).)*?</div>\s*', '', bottom_scripts, flags=re.DOTALL)
    bottom_scripts = re.sub(r'<button\s+@click\s*=\s*"sidebarOpen\s*=\s*!sidebarOpen"[^>]*>(?:(?!</button>).)*?</button>\s*', '', bottom_scripts, flags=re.DOTALL)

    # Also strip overlay divs from main_inner (they may have leaked there)
    main_inner = re.sub(r'\s*<div\s+x-show\s*=\s*"sidebarOpen"[^>]*>(?:(?!</div>).)*?</div>\s*', '', main_inner, flags=re.DOTALL)
    main_inner = re.sub(r'\s*<div\s+@click\s*=\s*"sidebarOpen\s*=\s*false"[^>]*>(?:(?!</div>).)*?</div>\s*', '', main_inner, flags=re.DOTALL)
    main_inner = re.sub(r'\s*<button\s+@click\s*=\s*"sidebarOpen\s*=\s*!sidebarOpen"[^>]*>(?:(?!</button>).)*?</button>\s*', '', main_inner, flags=re.DOTALL)

    # Clean bottom_scripts: remove old </body></html>, old wrapper close </div>
    bottom_scripts = re.sub(r'</body>\s*</html>\s*$', '', bottom_scripts, flags=re.DOTALL)
    bottom_scripts = re.sub(r'^\s*</div>\s*', '', bottom_scripts)
    bottom_scripts = bottom_scripts.strip()

    # Build output
    parts = [
        '<!DOCTYPE html>\n<html lang="id" class="dark">\n',
        '<head>',
        head_inner.strip(),
        '</head>\n',
        '<body class="bg-[#0b1120] text-slate-200 flex">\n',
    ]

    # Add inline script
    if script_block.strip():
        parts.append(script_block + '\n')

    # Add sidebar
    parts.append(SIDEBAR_HTML + '\n')

    # Main content area
    parts.append(f'    <div class="flex-1 min-w-0"{xdata_attr}>\n')
    parts.append(MOBILE_HEADER + '\n')
    parts.append('        <div class="max-w-6xl mx-auto px-4 md:px-6 py-6">\n')
    parts.append(main_inner + '\n')
    parts.append('        </div>\n')
    parts.append('    </div>\n')

    # Bottom scripts
    if bottom_scripts.strip():
        parts.append(bottom_scripts.strip() + '\n')

    parts.append('</body>\n</html>')

    result = ''.join(parts)
    # Clean up excessive blank lines
    result = re.sub(r'\n{4,}', '\n\n\n', result)

    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(result)

    return 'transformed'

# ---- Fix residues in already-transformed files ----
def fix_transformed_residues(filepath):
    """Clean up leftover sidebarOpen refs in files."""
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    changed = False

    # Remove sidebarOpen from script data objects
    # Pattern: sidebarOpen: false, or sidebarOpen: false
    new_content = re.sub(r'\s*sidebarOpen\s*:\s*(?:true|false|!sidebarOpen)\s*,?\s*', '', content)

    # Fix double commas: , , -> ,
    new_content = re.sub(r',\s*,', ',', new_content)
    # Fix { , -> {
    new_content = re.sub(r'\{\s*,', '{', new_content)
    # Fix , } -> }
    new_content = re.sub(r',\s*\}', '}', new_content)
    # Fix { } -> {}
    new_content = re.sub(r'\{\s*\}', '{}', new_content)

    # Remove overlay divs: <div x-show="sidebarOpen" ...></div>
    new_content = re.sub(
        r'\n\s*<div\s+x-show\s*=\s*"sidebarOpen"[^>]*>[\s\S]*?</div>\s*',
        '\n', new_content
    )
    # Remove overlay with @click
    new_content = re.sub(
        r'\n\s*<div\s+@click\s*=\s*"sidebarOpen\s*=\s*false"[^>]*>[\s\S]*?</div>\s*',
        '\n', new_content
    )
    # Remove floating sidebar toggle button
    new_content = re.sub(
        r'\n\s*<button\s+@click\s*=\s*"sidebarOpen\s*=\s*!sidebarOpen"[^>]*>[\s\S]*?</button>\s*',
        '\n', new_content
    )

    # Clean up excessive blank lines
    new_content = re.sub(r'\n{4,}', '\n\n\n', new_content)

    if new_content != content:
        changed = True
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(new_content)

    return 'fixed' if changed else 'clean'

# ---- Fix standalone pages (no sidebar) ----
def fix_standalone(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    changed = False

    # Remove h-full from html
    new_content = re.sub(r'class="dark\s+h-full"', 'class="dark"', content)
    new_content = re.sub(r'class="h-full\s+dark"', 'class="dark"', new_content)
    new_content = re.sub(r'class="h-full"', '', new_content)

    # Replace h-full on body with min-h-screen
    new_content = re.sub(r'class="h-full\s+flex\s+items-center\s+justify-center', 'class="min-h-screen flex items-center justify-center', new_content)
    new_content = re.sub(r'class="h-full\s+flex\s+items-center', 'class="min-h-screen flex items-center', new_content)
    new_content = re.sub(r'class="h-full\s+flex', 'class="min-h-screen flex', new_content)
    new_content = re.sub(r'class="([^"]*)h-full\s+([^"]*)"', r'class="\1\2"', new_content)
    new_content = re.sub(r'class="([^"]*)\s+h-full([^"]*)"', r'class="\1\2"', new_content)

    if new_content != content:
        changed = True
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(new_content)

    return 'cleaned' if changed else 'unchanged'

# ---- Main ----
def main():
    all_files = glob.glob(os.path.join(TEMPLATE_DIR, '**/*.html'), recursive=True)
    results = {'transformed': [], 'already': [], 'no_content': [], 'error': [], 'cleaned': [], 'skipped': [], 'unchanged': [], 'fixed': [], 'clean': []}

    for filepath in sorted(all_files):
        rel = os.path.relpath(filepath, TEMPLATE_DIR)
        top_dir = rel.split(os.sep)[0]

        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()

        # Skip partials/defines
        if '[[define ' in content[:200] and top_dir in ('layouts',):
            results['skipped'].append(rel)
            continue

        # Skip components
        if top_dir == 'components':
            results['skipped'].append(rel)
            continue

        # Skip admin (already handled separately)
        if top_dir == 'admin':
            results['skipped'].append(rel)
            continue

        # Already transformed? Check for residues
        if already_transformed(content):
            status = fix_transformed_residues(filepath)
            results[status].append(rel)
            if status == 'fixed':
                print(f"  FIXED_RESIDUE: {rel}")
            continue

        # Has old sidebar that needs transformation?
        if has_old_sidebar(content):
            status = transform_file(filepath)
            results[status].append(rel)
            print(f"  {status.upper()}: {rel}")
        else:
            # Standalone - just clean h-full
            status = fix_standalone(filepath)
            results[status].append(rel)
            if status == 'cleaned':
                print(f"  CLEANED: {rel}")

    print(f"\n=== Summary ===")
    for k, v in results.items():
        if v:
            print(f"  {k}: {len(v)}")

    # Second pass: clean residues from all newly transformed files
    print(f"\n=== Second pass: cleaning residues ===")
    residue_count = 0
    for filepath in sorted(all_files):
        rel = os.path.relpath(filepath, TEMPLATE_DIR)
        top_dir = rel.split(os.sep)[0]
        if top_dir in ('layouts', 'components', 'admin'):
            continue
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        if 'sidebarOpen' in content:
            status = fix_transformed_residues(filepath)
            if status == 'fixed':
                residue_count += 1
                print(f"  RESIDUE_FIXED: {rel}")
    print(f"  Total residue fixes: {residue_count}")

    # Third pass: clean duplicate </body></html> and other artifacts
    print(f"\n=== Third pass: cleaning HTML artifacts ===")
    artifact_count = 0
    for filepath in sorted(all_files):
        rel = os.path.relpath(filepath, TEMPLATE_DIR)
        top_dir = rel.split(os.sep)[0]
        if top_dir in ('layouts', 'components', 'admin'):
            continue
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        changed = False
        new_content = content

        # Remove duplicate </body></html> at end
        new_content = re.sub(r'(</body>\s*</html>\s*){2,}$', r'\1', new_content)
        # Remove stray </div> before <style> at the very end if followed by </body>
        new_content = re.sub(r'(\s*</div>\s*\n)(?=\s*<style[^>]*>[\s\S]*?</style>\s*</body>)', '\n', new_content)

        if new_content != content:
            changed = True
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(new_content)

        if changed:
            artifact_count += 1
            print(f"  ARTIFACT_FIXED: {rel}")
    print(f"  Total artifact fixes: {artifact_count}")

if __name__ == '__main__':
    main()
