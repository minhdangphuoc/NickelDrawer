import sys,unicodedata,pathlib,re
tot=0;nfd=0;files=0
for p in pathlib.Path(sys.argv[1]).rglob('*'):
    if p.suffix.lower() in ('.xhtml','.html','.htm'):
        t=p.read_text('utf-8',errors='replace'); files+=1
        comb=len(re.findall('[̀-ͯ]',t)); nfd+=comb
        if unicodedata.normalize('NFC',t)!=t: tot+=1
print(f"html files={files} files_not_NFC={tot} combining_marks={nfd}")
