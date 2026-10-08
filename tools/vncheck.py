import sys
from fontTools.ttLib import TTFont
vn="ĂăÂâĐđÊêÔôƠơƯưÀàẢảÃãÁáẠạẰằẲẳẴẵẮắẶặẦầẨẩẪẫẤấẬậÈèẺẻẼẽÉéẸẹỀềỂểỄễẾếỆệÌìỈỉĨĩÍíỊịÒòỎỏÕõÓóỌọỒồỔổỖỗỐốỘộỜờỞởỠỡỚớỢợÙùỦủŨũÚúỤụỪừỬửỮữỨứỰựỲỳỶỷỸỹÝýỴỵ"
for p in sys.argv[1:]:
    f=TTFont(p,lazy=True); cmap=f.getBestCmap()
    miss=[c for c in vn if ord(c) not in cmap]
    print(f"{len(miss):3d} missing  {f['name'].getDebugName(4)!r:40} {p.split('/')[-1]}  {''.join(miss[:20])}")
