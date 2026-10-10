"""Re-renders the body of a docs/guide page from its Markdown file.

Usage: python3 tools/guide2html.py docs/guide/developer-guide.md docs/guide/developer-guide.html
The existing HTML page keeps its head, style and header; only the section nav and the body
between </header> and </main> are replaced. Links to the other guide pages point at .html.
Needs the `markdown` package (pip install markdown).
"""
import re
import sys

import markdown

GUIDE_LINK = re.compile(r'href="((?:user-guide\.(?:en|vi)|developer-guide|index))\.md')


def render_body(markdown_text):
    markdown_text = re.sub(r'^## Table of contents\n.*?(?=^## )', '', markdown_text, flags=re.S | re.M)
    converter = markdown.Markdown(
        extensions=['tables', 'fenced_code', 'toc', 'sane_lists'],
        extension_configs={'toc': {'toc_depth': '2-2'}},
    )
    body = converter.convert(markdown_text)
    body = re.sub(r'<h1[^>]*>.*?</h1>', '', body, count=1, flags=re.S).lstrip('\n')
    body = body.replace('<table>', '<div class="tbl"><table>').replace('</table>', '</table></div>')
    body = GUIDE_LINK.sub(r'href="\1.html', body)
    return body, converter.toc_tokens


def section_links(tokens):
    for token in tokens:
        if token['level'] == 2:
            yield f'<a href="#{token["id"]}">{token["name"]}</a>'
        yield from section_links(token.get('children', []))


def replace_page_content(page, body, nav_links):
    header_end = page.index('</header>') + len('</header>')
    before = re.sub(
        r'(<nav class="toc"[^>]*><span class="lbl">[^<]*</span>).*?(</nav>)',
        lambda match: match.group(1) + nav_links + match.group(2),
        page[:header_end], count=1, flags=re.S)
    after = page[page.rindex('  </main>'):]
    return before + '\n    \n' + body + '\n' + after


def main(markdown_path, html_path):
    with open(markdown_path, encoding='utf-8') as source:
        body, tokens = render_body(source.read())
    with open(html_path, encoding='utf-8') as page_file:
        page = page_file.read()
    page = replace_page_content(page, body, ''.join(section_links(tokens)))
    with open(html_path, 'w', encoding='utf-8') as page_file:
        page_file.write(page)


if __name__ == '__main__':
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    main(sys.argv[1], sys.argv[2])
