#!/usr/bin/env python3
"""
Transform all Investo HTML templates to use the new consistent responsive layout.
- Removes h-full, fixed sidebar, sticky headers
- Uses flex layout with standard sidebar nav
- Preserves all Go template syntax, Alpine.js, and page scripts
"""

import re
import os
import glob

TEMPLATE_DIR = os.path.join(os.path.dirname(os.path.dirname(__file__)), 'web', 'templates')

# Standard sidebar nav (used on ALL sidebar pages)
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

    <!-- MOBILE OVERLAY -->
    <div id="sidebar-overlay" class="fixed inset-0 z-50 bg-black/60 hidden max-md:block" onclick="document.getElementById('sidebar').classList.toggle('max-md:hidden');this.classList.toggle('hidden')"></div>'''

MOBILE_HEADER_HTML = '''        <header class="flex items-center h-14 px-4 bg-[#0b1120] border-b border-slate-700/50 md:hidden">
            <button onclick="document.getElementById('sidebar').classList.toggle('max-md:hidden');document.getElementById('sidebar-overlay').classList.toggle('hidden')" class="p-2 text-slate-400">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/></svg>
            </button>
            <span class="font-bold text-white ml-2">Investo</span>
        </header>'''

NAV_LINK_CSS = '.nav-link{display:flex;align-items:center;gap:8px;padding:7px 12px;border-radius:8px;color:#94a3b8;font-size:13px;font-weight:500;text-decoration:none;transition:all .15s}.nav-link:hover{background:#1e293b;color:#e2e8f0}'

def find_matching_tag_end(content, tag, start):
    """Find matching closing tag position by counting nesting."""
    # Simplified: find the next </tag> at the same nesting level
    # Since <aside> doesn't nest inside itself, this is simpler
    close_tag = f"</{tag}>"
    depth = 1
    pos = start
    open_tag_pattern = re.compile(f"<{tag}\\b", re.IGNORECASE)
    close_tag_pattern = re.compile(f"</{tag}>", re.IGNORECASE)

    while depth > 0:
        open_match = open_tag_pattern.search(content, pos)
        close_match = close_tag_pattern.search(content, pos)
        if not close_match:
            return -1
        if open_match and open_match.start() < close_match.start():
            depth += 1
            pos = open_match.end()
        else:
            depth -= 1
            if depth == 0:
                return close_match.end()
            pos = close_match.end()
    return -1

def extract_xdata(content):
    """Extract x-data and x-init from body tag."""
    body_match = re.search(r'<body\b([^>]*)>', content, re.IGNORECASE | re.DOTALL)
    if not body_match:
        return '', ''

    body_attrs = body_match.group(1)
    xdata = ''
    xinit = ''

    xdata_match = re.search(r'x-data\s*=\s*"([^"]*)"', body_attrs)
    if xdata_match:
        xdata = xdata_match.group(1)

    xinit_match = re.search(r'x-init\s*=\s*"([^"]*)"', body_attrs)
    if xinit_match:
        xinit = xinit_match.group(1)

    return xdata, xinit

def extract_inline_script(content):
    """Extract the first inline <script>...</script> block in the body."""
    body_start = content.find('<body')
    if body_start == -1:
        return ''

    body_content = content[body_start:]
    script_match = re.search(r'<script\b[^>]*>(.*?)</script>', body_content, re.DOTALL)
    if script_match:
        return script_match.group(0)
    return ''

def remove_sidebar_open(data_str):
    """Remove sidebarOpen references from Alpine data strings."""
    if not data_str:
        return data_str
    # Remove sidebarOpen: false, or sidebarOpen: false (with possible whitespace variations)
    data_str = re.sub(r'\s*sidebarOpen\s*:\s*(true|false|!sidebarOpen),?\s*', '', data_str)
    # Also handle cases where sidebarOpen is the only/remaining property  
    data_str = re.sub(r'^\s*sidebarOpen\s*:\s*(true|false|!sidebarOpen)\s*,?\s*$', '', data_str, flags=re.MULTILINE)
    # Handle empty objects
    data_str = re.sub(r'\{[\s,]*\}', '', data_str)
    return data_str

def has_sidebar(content):
    """Check if file has a sidebar (aside with sidebar class)."""
    return bool(re.search(r'<aside\s+class="sidebar', content, re.IGNORECASE))

def transform_standalone(content):
    """Just remove h-full from html/body for standalone pages."""
    # Remove h-full from html tag
    content = re.sub(r'(<html\b[^>]*?)class="dark\s+h-full"', r'\1class="dark"', content)
    content = re.sub(r'(<html\b[^>]*?)class="h-full"', r'\1', content)
    content = content.replace('class="h-full"', '')
    # Remove h-full from body class
    content = content.replace('class="h-full flex items-center', 'class="min-h-screen flex items-center')
    content = content.replace('class="h-full flex', 'class="min-h-screen flex')
    content = content.replace('class="h-full ', 'class="')
    content = re.sub(r'class="([^"]*)h-full\s+([^"]*)"', r'class="\1\2"', content)
    content = re.sub(r'class="([^"]*)\s+h-full([^"]*)"', r'class="\1\2"', content)
    return content

def extract_head_and_footer(content):
    """Extract the <head> section and everything after the main content but before </body>."""
    head_end = content.find('</head>')
    if head_end == -1:
        return '', '', ''

    head = content[:head_end + len('</head>')]

    # Clean head
    head = re.sub(r'(<html\b[^>]*?)class="dark\s+h-full"', r'\1class="dark"', head)
    head = re.sub(r'(<html\b[^>]*?)class="h-full"', r'\1', head)

    # Find the main content block
    # There are a few patterns:
    # Pattern A: `<main class="flex-1 p-4...">` ... `</main>`  
    # Pattern B: `<main class="flex-1 ml-0 lg:ml-60...">` ... `</main>`
    # Pattern C: `<div class="main-content">` ... `</div>` (before </body>)

    body_content = content[head_end + len('</head>'):]
    body_end = body_content.rfind('</body>')

    # Find main content - look for <main ...> tag
    main_match = re.search(r'<main\b[^>]*class="[^"]*flex-1[^"]*"[^>]*>', body_content)
    main_end = None
    bottom_scripts = ''

    if main_match:
        # Find the matching </main> - it's the one closest before </body> that doesn't have nested <main>
        main_start_pos = main_match.start()
        # Use regex to find all </main> positions
        main_end_matches = list(re.finditer(r'</main>', body_content))
        if main_end_matches:
            main_end = main_end_matches[-1].end()  # Use the last one

    elif 'class="main-content"' in body_content:
        # bandarmologi pattern
        main_match = re.search(r'<div class="main-content"[^>]*>', body_content)
        if main_match:
            main_start_pos = main_match.start()
            # Find matching </div> by counting nesting
            div_start = main_match.end()
            depth = 1
            pos = div_start
            while depth > 0 and pos < len(body_content):
                open_div = re.search(r'<div\b', body_content[pos:])
                close_div = re.search(r'</div>', body_content[pos:])
                if not close_div:
                    main_end = len(body_content)
                    break
                if open_div and open_div.start() < close_div.start():
                    depth += 1
                    pos += open_div.end()
                else:
                    depth -= 1
                    if depth == 0:
                        main_end = pos + close_div.end()
                        break
                    pos += close_div.end()
    else:
        # Pattern: some files have no explicit main, content is directly in a div
        # Fall back to finding the content between sidebar and </body>
        return head, body_content[:body_end] if body_end > 0 else '', ''

    if main_end is None:
        return head, '', ''

    main_content = body_content[main_match.start():main_end]

    # Bottom scripts: everything between main_end and </body>
    if body_end > 0:
        bottom_scripts = body_content[main_end:body_end]

    return head, main_content, bottom_scripts

def transform_sidebar_file(filepath):
    """Transform a file with sidebar to the new layout."""
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    # Extract parts
    xdata, xinit = extract_xdata(content)
    head, main_content, bottom_scripts = extract_head_and_footer(content)

    if not main_content:
        print(f"  WARNING: Could not extract main content from {filepath}, skipping")
        return False

    # Clean up sidebarOpen from xdata 
    xdata_clean = remove_sidebar_open(xdata) if xdata else ''
    if not xdata_clean:
        xdata_clean = ''

    # Build x-data attribute for the inner wrapper
    xdata_attr = ''
    if xdata_clean:
        xdata_attr = f' x-data="{xdata_clean}"'
        if xinit:
            xdata_attr += f' x-init="{xinit}"'
    elif xinit:
        xdata_attr = f' x-init="{xinit}"'

    # Content wrapper opening
    if xdata_attr:
        content_wrapper_open = f'    <!-- MAIN CONTENT -->\n    <div class="flex-1 min-w-0"{xdata_attr}>\n'
    else:
        content_wrapper_open = '    <!-- MAIN CONTENT -->\n    <div class="flex-1 min-w-0">\n'

    content_wrapper_close = '    </div>'

    # Build the new body section
    new_body_parts = [
        content_wrapper_open,
        MOBILE_HEADER_HTML,
        '        <!-- PAGE CONTENT -->',
        '        <div class="max-w-6xl mx-auto px-4 md:px-6 py-6">',
        main_content.strip(),
        '        </div>',
        content_wrapper_close,
    ]

    # Combine everything
    new_content = [
        '<!DOCTYPE html>\n<html lang="id" class="dark">\n',
    ]

    # Extract everything from <head> to </head> (skip DOCTYPE and html opening)
    head_match = re.search(r'<head>(.*?)</head>', head, re.DOTALL)
    if head_match:
        new_content.append('<head>')
        new_content.append(head_match.group(1))
        new_content.append('</head>')
    else:
        new_content.append(head)

    # Body opening
    new_content.append('\n')
    new_content.append('<body class="bg-[#0b1120] text-slate-200 flex">\n')

    # Add inline script if present (but clean sidebarOpen from it)
    inline_script = extract_inline_script(content)
    if inline_script:
        cleaned_script = re.sub(r'sidebarOpen\s*:', '', inline_script)
        # Remove empty properties in the data object
        cleaned_script = re.sub(r',\s*,', ',', cleaned_script)
        cleaned_script = re.sub(r'{[\s,]*}', '', cleaned_script)
        # Fix common leftovers like `sidebarOpen, false,` → `false,`
        cleaned_script = re.sub(r'\n\s+sidebarOpen,?\s*\n', '\n', cleaned_script)
        cleaned_script = re.sub(r'\n\s+sidebarOpen\s*,?\s*\n', '\n', cleaned_script)
        if cleaned_script.strip():
            new_content.append(cleaned_script + '\n')

    # Sidebar + overlay
    new_content.append(SIDEBAR_HTML)
    new_content.append('\n')

    # Main content area (with mobile header + content + bottom scripts)
    new_content.extend(new_body_parts[:3])  # content wrapper open + mobile header + page content comment

    new_content.append('        <div class="max-w-6xl mx-auto px-4 md:px-6 py-6">')

    # Add main content - need to extract the inner content from the old <main> wrapper
    inner_start = main_content.find('>')
    if inner_start != -1:
        inner = main_content[inner_start+1:]
        # Remove the old header/topbar section if present (everything before <!-- CONTENT -->)
        content_comment = inner.find('<!-- CONTENT -->')
        if content_comment != -1:
            inner = inner[content_comment + len('<!-- CONTENT -->'):]
        inner = inner.strip()
        new_content.append(inner)

    new_content.append('        </div>\n')
    new_content.append(content_wrapper_close)

    # Bottom scripts  
    if bottom_scripts.strip():
        new_content.append(bottom_scripts)

    new_content.append('\n</body>\n</html>')

    # Add nav-link CSS to style blocks
    result = ''.join(new_content)
    # Insert nav-link CSS before the closing </style> tag
    result = re.sub(r'(</style>)', f'        {NAV_LINK_CSS}\n\\1', result, count=1)

    # Clean up: remove old sidebar-link references in style
    result = re.sub(r'\.sidebar-link\s*\{[^}]*\}\s*', '', result)
    result = re.sub(r'\.sidebar-link\s*:hover\s*\{[^}]*\}\s*', '', result)
    result = re.sub(r'\.sidebar-link\.active\s*\{[^}]*\}\s*', '', result)
    result = re.sub(r'\.sidebar-link\s*\.icon\s*\{[^}]*\}\s*', '', result)

    # Remove old media queries for sidebar
    result = re.sub(r'@media\s*\(\s*max-width\s*:\s*102[34]px\s*\)\s*\{\s*\.sidebar\s*\{[^}]+\}\s*\.sidebar\.open\s*\{[^}]+\}\s*\}', '', result)
    result = re.sub(r'@media\s*\(\s*max-width\s*:\s*102[34]px\s*\)\s*\{\s*\.sidebar\s*\{[^}]+\}\s*\}', '', result)

    # Remove old sticky topbar styles
    result = re.sub(r'@media\s*\(\s*max-width\s*:\s*640px\s*\)\s*\{[^}]*\.main-content\s*\{[^}]*\}\s*[^}]*\}', '', result)
    result = re.sub(r'\.topbar\s*\{[^}]*\}\s*', '', result)
    result = re.sub(r'\.main-content\s*\{[^}]*\}\s*', '', result)

    # Clean up multiple blank lines
    result = re.sub(r'\n{3,}', '\n\n', result)

    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(result)

    return True

def main():
    all_files = glob.glob(os.path.join(TEMPLATE_DIR, '**/*.html'), recursive=True)

    # Files to skip (components, layouts are partials; auth, pages, marketing are standalone)
    skip_dirs = {'components', 'auth', 'layouts', 'pages', 'marketing'}
    skipped = []
    transformed = []
    standalone = []

    for filepath in sorted(all_files):
        rel = os.path.relpath(filepath, TEMPLATE_DIR)
        top_dir = rel.split(os.sep)[0]

        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()

        if top_dir in skip_dirs or 'define' in content:
            # Handle standalone pages and partial templates
            if top_dir in ('auth', 'pages', 'marketing') and 'define' not in content[:50]:
                # Standalone - just remove h-full
                new_content = transform_standalone(content)
                with open(filepath, 'w', encoding='utf-8') as f:
                    f.write(new_content)
                standalone.append(rel)
                print(f"  CLEANED: {rel}")
            else:
                skipped.append(rel)
            continue

        if has_sidebar(content):
            if transform_sidebar_file(filepath):
                transformed.append(rel)
                print(f"  TRANSFORMED: {rel}")
            else:
                skipped.append(rel)
                print(f"  SKIPPED: {rel}")
        else:
            # No sidebar detected, check if it has h-full to clean
            if 'h-full' in content:
                new_content = transform_standalone(content)
                with open(filepath, 'w', encoding='utf-8') as f:
                    f.write(new_content)
            standalone.append(rel)

    print(f"\n=== Summary ===")
    print(f"Transformed (with new sidebar): {len(transformed)}")
    print(f"Cleaned (standalone, h-full removed): {len(standalone)}")
    print(f"Skipped (partials/components): {len(skipped)}")

if __name__ == '__main__':
    main()
