import sys, re, markdown
src, out = sys.argv[1], sys.argv[2]
md = open(src, encoding='utf-8').read()
md = re.sub(r'^## Table of contents\n.*?(?=^## )', '', md, flags=re.S|re.M)  # own nav instead
conv = markdown.Markdown(extensions=['tables', 'fenced_code', 'toc', 'sane_lists'],
                         extension_configs={'toc': {'toc_depth': '2-2'}})
body = conv.convert(md)
body = re.sub(r'<h1[^>]*>.*?</h1>', '', body, count=1, flags=re.S)
body = re.sub(r'<table>', '<div class="tbl"><table>', body); body = body.replace('</table>', '</table></div>')
body = re.sub(r'\*\*verify\*\*|\bverify\b(?![^<]*</code>)', '<mark>verify</mark>', body)
body = body.replace('<strong><mark>verify</mark></strong>', '<mark>verify</mark>')
def walk(ts):
    for x in ts:
        if x['level'] == 2: yield x
        yield from walk(x.get('children', []))
nav = ''.join(f'<a href="#{x["id"]}">{x["name"]}</a>' for x in walk(conv.toc_tokens))
tpl = open(sys.argv[3], encoding='utf-8').read()
open(out, 'w', encoding='utf-8').write(tpl.replace('{{NAV}}', nav).replace('{{BODY}}', body))
