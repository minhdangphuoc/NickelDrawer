# T-final — buổi thử trên máy gộp (G3 + G4 + G5)

Một buổi duy nhất, khoảng 90 phút cho phần chính (mục 1 đến 6, kể cả GitHub ở mục 4b và Việc cần làm cùng Bộ giao diện ở mục 5b) và khoảng 15 phút cho phần cố ý làm hỏng (mục 7). Buổi này thay các phiên T4, T5, T6, T7 của kế hoạch (`docs/plan-detailed.md` §5.1) và qua ba cổng G3, G4, G5 cùng lúc. Không có bài thử ngâm 72 giờ (`docs/DECISIONS.md`, 2026-10-10): thay vào đó nhật ký `debug.log` luôn bật để lỗi chạy lâu vẫn lộ ra.

Máy: Kobo Libra 2, firmware 4.38.23828. Chỉ người dùng chạm vào máy. Phần "Mac" do người bảo trì làm qua USB bằng `tools/device-session.sh`; không bao giờ `cd` vào `/Volumes/KOBOeReader`.

Cách đọc bảng: mỗi dòng là một ca. Tick một ô: `☐ Đạt` hoặc `☐ Lỗi`. Cột "Log" là dòng trong `.adds/nickeldrawer/debug.log` chứng minh ca đó (người bảo trì đọc sau mỗi lần cắm USB bằng `tools/device-session.sh log`). Dòng ghi "(syslog)" không nằm trong `debug.log` (chỉ có ở nhật ký hệ thống), nên ca đó chỉ xét bằng mắt.

Quy ước tên: Thêm = tab "Thêm" (More); Tinh chỉnh và Ứng dụng là hai hàng trong tab Thêm; Hình nền, LocalSend, Thư viện là tên ứng dụng trong Ngăn Ứng Dụng.

## 0. Việc đã đạt, không làm lại

| Mục | Đã đạt | Ghi chú |
|---|---|---|
| G1: cài và gỡ mod (hello), toast sau khi khởi động, khởi động bình thường | đã đạt 2026-10-09 | `the G1 session log (kept locally)`. Kiểm tra lại bằng SAFE-03 ở cuối. |
| Failsafe của NickelHook (SAFE-01, SAFE-07) | hoãn | G1 ghi DEFERRED. Không chạy trong buổi này (mod thật sẽ bị gỡ). Chạy riêng nếu muốn. |
| LocalSend phía nền: nhận 1 EPUB từ điện thoại, sách vào Thư viện (LS-01, LS-02, LS-06 cũ qua NickelMenu) | đã đạt 2026-10-09 | `T1-2026-10-09.md`. Giao diện trong Ngăn Ứng Dụng đã đổi nên LS-* bên dưới vẫn phải thử. |
| Cài bằng `tools/install-to-kobo.sh`, sao lưu, menu tiếng Việt | đã đạt 2026-10-09 | Menu NickelMenu của ứng dụng đã bị bỏ; bước cài mới (INS-01) thử lại. |
| Hình nền nhập từ Pictures và màn hình ngủ hiện ảnh toàn màn hình | đã đạt 2026-10-10 | Cần "Hiển thị sách đang đọc" và "Hiển thị bìa sách toàn màn hình" bật, và có một sách đang đọc. Thư viện ảnh và trình xem mới nên WP-* vẫn thử. |
| Prototype A/C, phím lật trang, nút quay lại cơ bản, tiếng Việt trong tiêu đề | đã đạt 2026-10-09 | `G2-next.md`. Bản đánh bóng (10-10) chưa lên máy nên DRW-* thử lại. |
| GH-01..08 | **làm trong buổi này** (mục 4b) | GitHub đã được đưa trở lại (DECISIONS 2026-10-10, thay cho quyết định hoãn 2026-10-09). Cần chuẩn bị ở mục 1. |
| KIT-04 (ô nhập) | **làm trong buổi này** (mục 5b) | Việc cần làm (TODO-01) và Bộ giao diện (UIK-05) có ô nhập thật. |
| KIT-05 (trường bí mật) | hoãn | Không ứng dụng nào có ô bí mật: token chỉ đi qua trang ghép nối trên điện thoại. Việc "token không lọt vào log, màn hình, trạng thái" được kiểm bằng GH-02 và bước Mac sau mục 4b. |
| LS-04 (công tắc "Tự cập nhật thư viện"), SAFE-08, FI-12, DRW-11 cũ, KIT-14 "Mở menu dự phòng" | bỏ | Thiết kế mới: không còn tab NickelMenu, không còn công tắc đó, NickelMenu không còn là dự phòng. Thay bằng DRW-11 mới, FI-02, FI-03. |
| DRW-06 (thư mục ứng dụng trống), DRW-10 (ngủ khi Ngăn mở) | gộp / bỏ | DRW-06 bỏ (cần gỡ hết ứng dụng); DRW-10 thử trong TWK-10. |
| KIT-06, KIT-07, KIT-08 | gộp | Gộp vào LS-01 (BigCode), LS-03 (danh sách có giới hạn), LIB-01 (tiến độ). |
| KIT-13 (ngôn ngữ en trên máy) | hoãn sang Mac | Đổi ngôn ngữ máy cần khởi động lại hai lần. Đã có `ndpreview --locale en` và golden test (`make golden-test`). |
| FI-07 (pin dưới 15 %), FI-08, FI-09 (đẩy dữ liệu quá lớn), đầy bộ nhớ | hoãn sang test đơn vị | Không thử được an toàn trên máy (đầy bộ nhớ có thể làm hỏng cơ sở dữ liệu Kobo). Đã có test `err_no_space`, `err_tempfile`, giới hạn dòng trong `mod/drawer/tests` và `internal`. |

## 1. Chuẩn bị trên Mac (khoảng 5 phút, chưa cắm máy)

- ☐ `make check build dist` xanh, và `tools/test-install.sh` xanh.
- ☐ `build/mod/KoboRoot-nickeldrawer.tgz` mới hơn mọi tệp trong `mod/`; nếu không, build lại mod rồi chạy lại bước trên.
- ☐ Pin Kobo từ 50 % trở lên: ____ %. Wi-Fi nhà bật. Điện thoại có LocalSend, cùng Wi-Fi. Ghi giờ bắt đầu: ____
- ☐ Ghi trạng thái ban đầu của bốn công tắc Tinh chỉnh để trả lại cuối buổi: Màn hình chờ ____ · Chụp màn hình ____ · Tự kết nối USB ____ · Chữ rõ nét ____ (xem trong `Kobo eReader.conf` qua USB, hoặc ghi ở TWK-01).
- ☐ Chuẩn bị 13 ảnh PNG/JPEG (đủ 2 trang lưới: 12 + 1; có thể dùng `go run ./tools/wallpaper-samples` và nhân bản). Chưa chép vào máy.
- ☐ Chuẩn bị 2 sách thử: bản sao một EPUB tiếng Việt của bạn gắn ngôn ngữ `en` (hoặc không gắn), và một bản sao y hệt tệp đó (cùng nội dung, tên khác). Tên thư mục thử: `Books/ndtest/`. Không dùng sách riêng tư trong repo.
- ☐ Chuẩn bị ba thư mục ứng dụng giả `ndtest-bad-ui`, `ndtest-big`, `ndtest-many` (xem DRW-07 và FI-02). Chỉ để ngoài repo, trong `local/`.
- ☐ **GitHub (người dùng chuẩn bị, trên điện thoại hoặc máy tính, trước buổi):**
  1. Một repo **riêng tư** (ví dụ `<tên bạn>/kobo-thu`) có thư mục `epub/` chứa **ít nhất một sách thật** (khuyên 2), tên có dấu tiếng Việt càng tốt (ví dụ `Việt Nam sử lược.epub`). Thêm hai tệp thử bị bỏ qua: `epub/gia.epub` (tệp văn bản đổi đuôi thành .epub) và `epub/ghi-chu.txt`. Có thể thêm `epub/mau.pdf` để thấy PDF vào `Documents/GitHub`. Giữ sẵn một sách thứ ba để thêm vào repo ở GH-06.
  2. (Tùy chọn, cho GH-07) một repo riêng tư thứ hai **hoàn toàn trống** (không commit nào, không README).
  3. Một **fine-grained personal access token**: Repository access = *Only select repositories* và chỉ chọn repo ở bước 1 (thêm repo trống nếu có); Permissions = **Contents: Read-only** (Metadata tự có); hết hạn trong 7 ngày. Giữ token trong trình quản lý mật khẩu của điện thoại để dán vào trang ghép nối. **Chỉ nhập token ở trang ghép nối trên điện thoại.** Không gõ nó vào chat, vào Mac, vào Kobo, không chụp màn hình. Xóa (revoke) token trên GitHub sau buổi.
- ☐ **Bộ giao diện (UI Kit) ẩn mặc định.** Ứng dụng này là mục tiêu thử KIT-*, nên chỉ hiện khi bạn bật cho buổi này, và bật bằng chính máy (không sửa `app.json`, không cần USB): trong Ngăn bấm bánh răng ⚙ ở tiêu đề, bật **Bộ giao diện** trong mục **Ứng dụng ẩn**, bấm ←: hàng/ô **Bộ giao diện** hiện ra (ca SET-02). Cuối buổi tắt lại công tắc đó. Dữ liệu của nó nằm trong `data/` của ứng dụng, không đụng sách.
- ☐ Chuẩn bị cho TODO-06: một tệp `todo-hong.json` ngoài repo, trong `local/`, nội dung `{ hỏng` (cố ý không phải JSON).
- ☐ Hỏi lại: Kobo không đang hiện lời mời nâng cấp Qt6 (`tools/device-session.sh status` báo "none" hoặc "declined"). Nếu có, bấm Bỏ qua trên máy, không đồng ý.

## 2. Cắm máy lần 1: cài (U1, khoảng 6 phút)

Người điều phối:

1. Cắm USB, bấm Kết nối trên Kobo. `tools/device-session.sh status`: firmware 4.38.23828, không có `KoboRoot.tgz` đang chờ, không có cờ `uninstall*` hay `disable`. Nếu có tệp `._*`: `tools/device-session.sh clean`.
2. Sao lưu: `cp` `.kobo/KoboReader.sqlite`, `.kobo/Kobo/Kobo eReader.conf` và `.kobo/version` vào `backups/<ngày>/`; sao lưu `.adds/apps/wallpaper/data` và danh sách `Pictures/` (thư mục `backups/` đã bị gitignore).
3. `tools/install-to-kobo.sh --dry-run`, đọc từng dòng `[dry-run]`; rồi `tools/install-to-kobo.sh` (tự gỡ `.adds/nm/koboapps` sau khi sao lưu và tắt tab NickelMenu `menu_main_15505_enabled:0`).
4. Xóa dữ liệu Hình nền cũ để thử lần nhập đầu: xóa `.adds/apps/wallpaper/data`, các tệp `.kobo/screensaver/nd-wallpaper-*` và ảnh thử cũ trong `Pictures/` (đã sao lưu ở bước 2). Để `Pictures/` trống.
5. `tools/device-session.sh stage build/mod/KoboRoot-nickeldrawer.tgz`, rồi `tools/device-session.sh flags debug screens` (cờ `screens` bật màn hình ứng dụng), rồi `tools/device-session.sh eject`.

Người dùng: rút USB, chờ Kobo cài và khởi động lại, rồi đợi 20 giây ở Trang chủ.

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| INS-01 | Điều phối: bước 3 ở trên | Dry-run nêu thư mục sao lưu, danh sách tệp, không lỗi, "nothing was changed"; cài thật không lỗi; `.adds/nm/koboapps` không còn; `Kobo eReader.conf` có `menu_main_15505_enabled:0` | (Mac) kết quả lệnh | ☐ | ☐ |
| INS-02 | Chờ máy khởi động lại (tối đa 3 phút). Nhìn thanh tab dưới cùng | Máy vào Trang chủ, không lặp khởi động, không toast lạ. Thanh tab chỉ còn các tab gốc của Nickel (Trang chủ, Thư viện, …, Thêm), không có tab NickelMenu "Ứng dụng" | không có dòng riêng; mở `debug.log` thấy `bridge: MoreView constructed` sau khi bấm Thêm (DRW-01) | ☐ | ☐ |

Nếu máy không lên: cứ để Nickel tự khởi động lại; tắt nguồn một lần trong 15 giây đầu sau khi mod nạp để NickelHook bỏ mod ở lần khởi động sau. Hoặc cắm USB, người bảo trì `flags uninstall`, khởi động lại. Dừng buổi và gửi `debug.log`.

## 3. Ngăn Ứng Dụng và Tinh chỉnh (không cần USB, khoảng 18 phút)

Nhớ ghi giờ khi thử DRW-02 (S1).

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| DRW-01 | Bấm tab Thêm. Nhìn các hàng | Có hàng "Ứng dụng" (dòng trạng thái nhỏ, biểu tượng gốc của Nickel) và hàng "Tinh chỉnh" (biểu tượng bánh răng của Nickel), nằm cùng kiểu với các hàng Thêm khác | `bridge: MoreView constructed`, `more row: Nickel icon ... (WxH)` | ☐ | ☐ |
| DRW-02 | Bấm hàng "Ứng dụng" một lần. Bấm giờ | Ngăn mở, tiêu đề "Ứng dụng", mục "ĐÃ CÀI" và "HỆ THỐNG", thanh tab còn, tab "Thêm" sáng. Vẽ xong trong 1 giây (S1) | `entry row: opening the drawer` → `navigator: route A: push controller` → `drawer controller: viewWillAppear`; lấy giờ giữa dòng đầu và dòng cuối | ☐ | ☐ |
| DRW-13 | Nhìn các nút bên phải tiêu đề. Bấm nút chuyển danh sách/lưới. Rồi bấm lại. Thoát và mở lại Ngăn | Thứ tự các nút từ trái sang phải: ⟳ làm mới, danh sách, lưới, ⚙ cài đặt (cả ở danh sách lẫn lưới, không còn nút tải xuống riêng cho cập nhật). Đổi giữa danh sách (hàng) và lưới (ô); lựa chọn được nhớ sau khi mở lại | `drawer view: header created (... refresh, list/grid toggles, settings)`, `drawer view: restore ...` | ☐ | ☐ |
| DRW-04 | Trong danh sách, kiểm các ứng dụng. Bấm một ô/hàng LocalSend | Thấy GitHub, LocalSend, Hình nền, Thư viện (không còn hàng "Làm mới danh sách" hay "Cập nhật": chúng là nút ⟳ và ⚙ trên tiêu đề). Bấm LocalSend mở màn hình ứng dụng có thanh tiêu đề với mũi tên quay lại (xem LS-01), không hiện hộp thoại văn bản | `screen: "main" page 1/1, ...`; (syslog) `screen: open localsend` | ☐ | ☐ |
| DRW-05 | Trong màn LocalSend bấm mũi tên ← ở tiêu đề | Về Ngăn Ứng Dụng (đúng chế độ danh sách/lưới), không trắng màn hình | `screen: back tapped on "main"`, `navigator:` hoặc `navigation: pop ...` | ☐ | ☐ |
| DRW-03 | Từ Ngăn, bấm tab Thư viện, rồi Thêm, rồi "Ứng dụng" một lần nữa | Về Ngăn Ứng Dụng, trạng thái làm mới, chỉ có một Ngăn (không chồng lớp). Thoát bằng ←: về Thêm bình thường | `navigator: back`, `drawer controller destroyed` đúng một lần cho mỗi lần đóng; không có `navigator: already open` ngoài khi bấm đúp | ☐ | ☐ |
| DRW-14 | Trong Ngăn, nhấn giữ một ứng dụng (ví dụ Thư viện) | Menu kiểu Nickel (các hàng, không có nút hộp): Thông tin, các thao tác của ứng dụng, Gỡ…. Bấm ra ngoài thì đóng. Mở lại, chọn Gỡ…, bấm **Hủy**: không gỡ gì | `row menu: Nickel menu ... rows`, `popup: ... opened`, `popup: ... closed: ...`, `row menu: row N (...) tapped` | ☐ | ☐ |
| SET-01 | Trong Ngăn bấm bánh răng ⚙ ở cuối hàng nút trên tiêu đề (cả ở danh sách lẫn lưới). Nhìn màn hình. Bấm "Giới thiệu dự án". Bấm ← hai lần | Màn "Cài đặt NickelDrawer" mở với mũi tên ←: các mục Ứng dụng ẩn, Cập nhật (Phiên bản, Nguồn "minhdangphuoc/NickelDrawer"), Gỡ lỗi, Thông tin (Firmware kèm ✓ cho 4.38.23828). Nút Kiểm tra và (mờ khi chưa có bản mới) Cập nhật. "Giới thiệu dự án" mở màn Giới thiệu; ← về Cài đặt, ← nữa về Ngăn đúng chế độ danh sách/lưới. Bánh răng là biểu tượng gốc của Nickel | `screen: tap ...`, `screen: open _settings`, `navigation: push ... about`, `navigation: pop` | ☐ | ☐ |
| SET-02 | Trong ⚙ bật công tắc "Bộ giao diện" (dòng nhỏ đổi từ "Đang ẩn · dành cho nhà phát triển" sang "Đang hiện trong Ngăn"). Bấm ← về Ngăn. Quay lại ⚙, tắt công tắc, về Ngăn | Khi bật: về Ngăn thấy hàng/ô "Bộ giao diện". Khi tắt: biến mất. Không cần cắm USB hay làm mới. `app.json` của ứng dụng vẫn `"hidden": true` (kiểm bằng USB ở lần cắm sau); `data/settings.json` của máy chủ ghi `shown_hidden_apps` | `screen: tap on toggle N, action "show_hidden_uikit"`, `queue: show_hidden_uikit ...` | ☐ | ☐ |
| SET-03 | Trong ⚙ bật "Ghi nhật ký gỡ lỗi", cắm USB: xem `.adds/nickeldrawer/debug`; tắt công tắc, cắm USB xem lại | Bật: có tệp rỗng `debug`; dòng ghi chú "Có hiệu lực sau khi khởi động lại". Tắt: tệp biến mất. Công tắc đọc đúng trạng thái khi mở lại màn | `queue: debug ...` | ☐ | ☐ |
| SET-04 | Trong ⚙ bấm "Giới thiệu dự án". Nhìn mã QR; dùng camera điện thoại quét. Đọc phần Dự án và Ghi nhận | Có tiêu đề NickelDrawer, "Phiên bản …", mã QR cùng dòng địa chỉ https://github.com/minhdangphuoc/NickelDrawer và chú thích "Quét để mở GitHub"; điện thoại mở đúng trang kho mã. Người bảo trì "Minh Dang", Giấy phép PolyForm Noncommercial 1.0.0, Firmware hỗ trợ 4.38.23552 – 4.38.23828; mục Ghi nhận đủ sáu hàng; dòng cuối "Dự án độc lập, không liên kết với Rakuten Kobo…". Không chữ bị cắt đuôi | (không có dòng riêng cho QR; so sánh mắt với bản xem trước `settings-about-vi.png`) | ☐ | ☐ |
| SET-05 | Có Wi-Fi. Trong ⚙ bấm "Kiểm tra" (Cập nhật). Chờ vài giây rồi nhìn lại màn (mở lại nếu cần) | Toast "Đang kiểm tra cập nhật." rồi mục Cập nhật ghi "Đang kiểm tra cập nhật…" (nút Kiểm tra mờ, màn tự làm mới, không cần mở lại). Xong luôn có một dòng kết quả kèm giờ, giữ đến lần kiểm tra sau: đã mới nhất thì "Đã kiểm tra lúc HH:MM · Bạn đang dùng bản mới nhất (<số>)"; có bản mới thì "… · Có bản mới <số>"; lỗi thì "Không kiểm tra được lúc HH:MM: <lý do>" (chưa có Wi-Fi, không kết nối được GitHub, GitHub đang giới hạn, chữ ký sai, kênh này chưa có bản phát hành). Chưa có bản phát hành: không có dòng "Bản mới", nút lớn vẫn "Cập nhật" và mờ; có bản mới hơn: bánh răng có chấm đen, hàng "Bản mới" kèm ●, nút "Cập nhật lên <số>" sáng, bấm hiện hộp xác nhận "Cài bản <số>? …" (chọn Hủy để không cài trên buổi này). Tắt Wi-Fi rồi Kiểm tra: máy hỏi bật Wi-Fi; nếu từ chối hoặc không kết nối được thì dòng kết quả ghi lý do, không để trống | `screen: tap on button N, action "check"`, (syslog) `update check` | ☐ | ☐ |
| TWK-01 | Bấm Thêm, bấm hàng "Tinh chỉnh". Ghi lại các công tắc đang Bật hay Tắt | Cửa sổ kiểu Cài đặt của Kobo: tiêu đề nhỏ ở giữa, nút ✕, tiêu đề lớn có chân, các hàng chữ nghiêng có đường kẻ, danh sách 2 cột. Có: Màn hình chờ, Chụp màn hình, Tự kết nối USB, Chữ rõ nét, Địa chỉ IP, Đảo màu và khởi động lại, Ngủ, Khởi động lại, Quét lại thư viện. Ba hành động Ngủ, Khởi động lại, Đảo màu và khởi động lại nằm cuối. Công tắc hiện Bật/Tắt | `tweaks: Nickel N3Dialog ... pushed`, `tweaks panel: 9 tweak(s), N page(s)` | ☐ | ☐ |
| TWK-02 | Bấm "Chụp màn hình" | Đổi Bật/Tắt ngay, hiện toast "Đã bật Chụp màn hình" (hoặc "Đã tắt …"); không hộp thoại văn bản. Bấm lại để trả về như cũ | `tweaks panel: tap on "screenshots"`, `tweaks: "screenshots" applied` | ☐ | ☐ |
| TWK-03 | Bấm "Tự kết nối USB" để đổi, ghi trạng thái | Đổi Bật/Tắt, có toast. Để nguyên cho đến khi người bảo trì kiểm trên Mac ở mục 4, rồi trả lại | `tweaks panel: tap on "auto-usb"`, `tweaks: "auto-usb" applied` | ☐ | ☐ |
| TWK-04 | Bấm "Chữ rõ nét" | Đổi Bật/Tắt; toast nói "Mở lại sách để áp dụng". Trả lại như cũ ở cuối | `tweaks panel: tap on "legibility"`, `tweaks: "legibility" applied` | ☐ | ☐ |
| TWK-05 | Bấm "Màn hình chờ" (tắt), rồi bấm lại (bật) | Mỗi lần đổi ngay kèm toast. Cuối cùng để **Bật** (Hình nền cần bật). Mac kiểm: thư mục `.kobo/screensaver` đổi tên thành `.kobo/screensaver.disabled` khi Tắt và về lại khi Bật; ảnh không mất | `tweaks panel: tap on "screensaver"`, `tweaks: "screensaver" applied` ×2 | ☐ | ☐ |
| TWK-06 | Xem hàng "Địa chỉ IP". Bấm vào nó | Hiện địa chỉ IP của Wi-Fi (hoặc "Không có Wi‑Fi"). Bấm không làm gì | `tweaks panel: tap on "ip"` (không có dòng `applied`) | ☐ | ☐ |
| TWK-07 | Bấm "Quét lại thư viện" | Toast "Đang quét thư viện". Máy quét nhanh, không treo, không hộp thoại | `tweaks: "library-rescan" applied` | ☐ | ☐ |
| TWK-08 | Bấm "Khởi động lại" rồi chọn **Hủy**. Làm tương tự với "Đảo màu và khởi động lại" | Có hộp xác nhận ("Khởi động lại ngay?", "Đảo màu màn hình và khởi động lại ngay?"). Hủy thì không có gì xảy ra | `tweaks: "reboot" cancelled`, `tweaks: "invert-reboot" cancelled` | ☐ | ☐ |
| TWK-09 | Nếu cửa sổ có hơn 1 trang: lật bằng phím chuyển trang. Bấm ✕. Mở lại, bấm ra ngoài khung | Phím lật trang đổi trang; ✕ và bấm ngoài đều đóng; Thêm hiện lại bình thường | `tweaks panel: page 1 -> 2`, `tweaks: closing, pop controller`, `tweaks: modal destroyed` | ☐ | ☐ |
| ICO-01 | Trong các bước trên và ở mục 5, nhìn mọi biểu tượng: ←, ✕, bánh răng, dấu ✓, nút nhập ảnh, hình ảnh trống, chuyển danh sách/lưới | Mọi biểu tượng lấy từ tài nguyên của Nickel hoặc Tabler (MIT), cùng nét với phần còn lại của Nickel; không hình tự vẽ. Không ô trống chữ cái, không khung thiếu ảnh | `icon <tên>: ... WxH` (không có `missing from Nickel's resources`, `missing or unreadable`) | ☐ | ☐ |
| WP-00 | Thư mục `Pictures/` đang trống. Mở Ngăn, bấm **Hình nền** | Màn trạng thái trống: biểu tượng ảnh, "Chưa có ảnh", chữ hướng dẫn chép ảnh vào Pictures, nút "Nhập ảnh". Không tự nhập gì | `screen: "main" page 1/1, ...` | ☐ | ☐ |

## 4. LocalSend và cắm USB lần 2 (khoảng 12 phút)

Đầu tiên **tắt Wi-Fi** của Kobo (Cài đặt hoặc biểu tượng trạng thái) để thử DRW-12.

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| DRW-12 | Mở LocalSend, bấm "Bắt đầu nhận" khi Wi-Fi đang tắt | Hộp kết nối Wi-Fi của Nickel hiện lên; nối xong thì bắt đầu nhận. Trong 20 giây | `queue:`/(syslog) `action:` cho `start`; không có `wifi: no connection after` | ☐ | ☐ |
| LS-01 | Sau khi đã nhận: nhìn màn hình LocalSend | Mã PIN lớn (BigCode), dòng "Nhập mã PIN này trên điện thoại", Thiết bị, IP, nhãn "Đang nhận". Toast "Đang nhận. Nhập PIN trên điện thoại.". Nút chính dưới cùng là "Dừng" (nút "Bắt đầu nhận" không còn trên màn hình). **Không có hộp thoại văn bản** chứa PIN/IP (API-01). Mã PIN không nháy cả màn hình | `screen: tap on button N, action "start"`, `queue:` hoặc (syslog) `action: event start` | ☐ | ☐ |
| LS-02 | Từ điện thoại gửi 1 EPUB sang Kobo | Hàng mới trong "Đã nhận" có tên tệp; toast. Sách vào Thư viện trong khoảng 30 giây (chưa cần dừng) | `screen: "main" page ...` làm mới; (syslog) cập nhật từ `state` | ☐ | ☐ |
| LS-03 | Gửi 3 tệp cùng lúc | 3 hàng; danh sách không dài quá 10 hàng (giới hạn `max`); chỉ một lần quét lại sau khi dừng | cùng loại với LS-02 | ☐ | ☐ |
| LS-05 | Bấm ← về Ngăn mà **không** dừng. Nhìn ô/hàng LocalSend. Gửi thêm 1 tệp | Dòng trạng thái đổi thành "Đang nhận"; máy vẫn nhận tệp mới | `screen: back tapped on "main"` | ☐ | ☐ |
| LS-06 | Mở LocalSend lại, bấm "Dừng" | Toast "Đã dừng. Đang cập nhật thư viện.". Nhãn thành "Chưa nhận" và nút chính đổi lại thành "Bắt đầu nhận". Sách mới có trong Thư viện (có thể mất 30 giây); mở được | `screen: tap on button N, action "stop"` | ☐ | ☐ |
| LS-07 | Bắt đầu nhận lần nữa. Khi đang nhận, **cắm USB** vào Mac | Ngăn/màn hình ứng dụng đóng trước khi Kobo hỏi Kết nối; bấm Kết nối thì thư mục hiện ra; không treo | `navigator: USB plugged, closing the drawer before Nickel's connect dialog`, `navigator: USB plug signal ...` | ☐ | ☐ |

Người điều phối (U2, khoảng 6 phút), máy đã nối USB:

1. `tools/device-session.sh log` và lưu bản log. Đọc các ca ở mục 3 và 4.
2. Kiểm `Kobo eReader.conf` và thư mục theo các ca TWK-02..05 (ghi vào cột Ghi chú cuối file).
3. SAFE-06: chạy lại `tools/install-to-kobo.sh` (cùng bản) để thử cài đè; kiểm Hình nền/LocalSend vẫn còn dữ liệu.
4. Chép 13 ảnh vào `Pictures/`; chép 2 sách thử vào `Books/ndtest/`; chép **một sách bất kỳ** (SAFE-04) vào `Books/`; chép 3 thư mục ứng dụng giả vào `.adds/apps/`: `ndtest-bad-ui` (ui.json bị cắt cụt, ngoài ra một màn có thành phần lạ và một biến `$undefined`), `ndtest-big` (biểu tượng 5 MB và `app.json` 1 MB), `ndtest-many` (20 thư mục app rỗng dạng hợp lệ). Không sửa gì khác.
5. `tools/device-session.sh clean`, `eject`. Người dùng bấm Rút USB. **Không khởi động lại.**

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| SAFE-04 | Mở sách vừa chép qua USB trong Thư viện | Máy không báo "đang sửa thư viện" và sách mở bình thường | `debug.log` không có lỗi mới | ☐ | ☐ |
| SAFE-06 | Sau cài đè: mở Ngăn, kiểm các ứng dụng | Ứng dụng và cài đặt còn nguyên, không phải khởi động lại | `drawer view: constructed ...` | ☐ | ☐ |
| DRW-07 | Mở Ngăn Ứng Dụng sau khi rút USB (không khởi động lại) | Có thêm các ứng dụng giả mới (cuộn nếu cần). Mở trong 1 giây dù có 20+ thư mục (FI-11) | `drawer view: constructed ...` | ☐ | ☐ |
| FI-11 | Nhìn `ndtest-big`, `ndtest-many` | Biểu tượng 5 MB được thay bằng chữ cái/placeholder; `app.json` 1 MB bị từ chối: thẻ lỗi. Ngăn không treo | `picture:`/(syslog) dòng bỏ qua tệp quá lớn; Ngăn mở < 1 s | ☐ | ☐ |
| FI-02 | Mở `ndtest-bad-ui` | Màn hình lỗi kiểu Kobo: có đường dẫn tệp hỏng, có mũi tên ← quay lại, không văn bản NickelMenu. Thành phần lạ: hàng lỗi. `$undefined`: trống. Nickel không sập | (syslog) `screen: ndtest-bad-ui failed to load` hoặc `screen: ... reply kept the previous screen` | ☐ | ☐ |

## 4b. GitHub (khoảng 15 phút)

Wi-Fi bật lại (sau mục 4), điện thoại cùng Wi-Fi, token và repo đã chuẩn bị ở mục 1 sẵn sàng. Mở Ngăn > **GitHub**. Nhật ký của ứng dụng nằm ở `.adds/apps/ghsync/data/log.txt` (người bảo trì đọc sau khi cắm USB); không bao giờ dán token vào chat hay ghi nó vào cột ghi chú.

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| GH-01 | Mở GitHub lần đầu (chưa ghép nối) | Đầu trang "Chưa ghép nối". Ô xám "Chưa ghép nối" nói ghép nối bằng điện thoại để nhập repo và token, **không gõ gì trên Kobo**. Các dòng Repo/Nhánh/Thư mục/Đồng bộ lúc/Sách mới thêm là "—". Hàng "Ghép nối bằng điện thoại ›". Nút "Đồng bộ ngay" **mờ**. Không hộp thoại văn bản (API-01) | `screen: "main" page ...`; không có `tap has no action` | ☐ | ☐ |
| GH-02 | Bấm hàng "Ghép nối bằng điện thoại" (màn "Ghép nối điện thoại"), bấm "Bắt đầu ghép nối". Trên điện thoại: quét mã QR (hoặc gõ địa chỉ). Nhập **sai mã** một lần. Rồi nhập đúng mã, repo, nhánh để trống, thư mục `epub`, dán token, bấm "Kiểm tra và lưu" | Toast "Mở địa chỉ trên điện thoại." Trong 10 giây: mã QR, địa chỉ `http://<ip>:8765`, mã 6 số lớn, dòng "Còn mở n phút, đến hh:mm" (n giảm sau vài phút), đầu trang "Đang ghép nối"; "Dừng ghép nối" sáng, "Bắt đầu" mờ. Trang trên điện thoại bằng tiếng Việt; sai mã báo "Sai mã. Còn 4 lần thử."; đúng thì "Đã ghép nối". Trên Kobo trong 10 giây: ghi chú "Đã ghép nối với <repo>…", QR biến mất, đầu trang "Sẵn sàng". Bấm ←: Repo, Nhánh "Mặc định", Thư mục "epub"; "Đồng bộ ngay" sáng. **Token không hiện ở bất cứ đâu trên Kobo** | `pairing started on`, `pairing: wrong code (1 of 5)`, `paired with <repo>` (không có token) | ☐ | ☐ |
| GH-03 | Bấm "Đồng bộ ngay" | Toast "Đang đồng bộ trong nền." Đầu trang "Đang đồng bộ…"; nút mờ; dòng "Đang tải 1 / N" và tên tệp (nếu kịp thấy). Xong trong 2 phút: "N mới, 0 đã có, 0 lỗi", "Đồng bộ lúc" là giờ vừa rồi, "Sách mới thêm N". Thư viện **tự quét lại**, không bấm gì: sách xuất hiện, mở được, tên tiếng Việt đủ dấu. **Không** có `gia.epub` và `ghi-chu.txt` trong Thư viện. Không hộp thoại văn bản | `repo ...: N matching files`, `downloaded: epub/...`, `skipped, content does not match a epub file: epub/gia.epub`, `done: N new, 0 existing, 1 skipped, 0 errors` (log ứng dụng); `queue:`/(syslog) yêu cầu rescan | ☐ | ☐ |
| GH-04 | Bấm "Đồng bộ ngay" lần nữa | "0 mới, N đã có, 0 lỗi"; không tải gì; Thư viện **không** quét lại; "Sách mới thêm 0" | `unchanged since the last run` hoặc `done: 0 new, N existing` | ☐ | ☐ |
| GH-05 | Trong Thư viện **xóa một sách** đã đồng bộ (nhấn giữ > Xóa). Quay lại GitHub, bấm "Đồng bộ ngay" | Sách bị xóa **không tải lại** (add-only, nhớ trong `state.json`); "0 mới"; không lỗi. Các sách khác còn nguyên | `deleted locally, skipped: epub/...` | ☐ | ☐ |
| GH-06 | Trên điện thoại (GitHub): **xóa** một sách đã đồng bộ khỏi repo và **thêm** sách thứ ba vào `epub/`. Trên Kobo bấm "Đồng bộ ngay" | Sách mới về và vào Thư viện sau khi tự quét lại ("1 mới"). Sách đã xóa khỏi repo **vẫn còn** trên máy (không bao giờ xóa) | `downloaded: epub/<sách mới>`; không có dòng xóa | ☐ | ☐ |
| GH-07 | Bắt đầu một phiên ghép nối mới. Trên điện thoại, trong cùng phiên, thử lần lượt (mỗi lần nhập lại mã và dán lại token): repo sai tên (`<tên bạn>/khong-co-repo`); nhánh `khong-co`; thư mục `khong-co`; rồi một token sai (đổi ký tự cuối). Cuối cùng nhập đúng (repo, nhánh trống, thư mục `epub`). **Tùy chọn:** phiên mới với repo **trống**, đồng bộ, rồi ghép nối lại với repo thật | Mỗi lỗi một câu **khác nhau** trên trang điện thoại, không phải "404" chung: "Không tìm thấy repo … hoặc token không thấy repo này", "Nhánh khong-co không có trong …", "Thư mục khong-co không có trên nhánh …", "token sai hoặc hết hạn (401)". Mỗi lỗi: ô repo/nhánh/thư mục giữ nguyên, ô token trống, **không lưu gì** (cấu hình cũ còn), không mất lượt mã. Nhập đúng thì "Đã ghép nối". Repo trống: ghép nối **được**, đồng bộ cho "0 mới, 0 đã có", không lỗi | `pairing failed: ...` (không có token); với repo trống `repo ...: 0 matching files` | ☐ | ☐ |
| GH-08 | **Tắt Wi-Fi**, bấm "Đồng bộ ngay"; làm tương tự với "Bắt đầu ghép nối". Rồi bật Wi-Fi, dùng "Dừng ghép nối" khi đang mở phiên ghép nối | Wi-Fi tắt: hộp kết nối Wi-Fi của Nickel hiện (như DRW-12) hoặc toast "Chưa có Wi-Fi. Kết nối rồi thử lại."; không treo. "Dừng ghép nối": toast "Đã dừng ghép nối.", QR biến mất ≤ 10 giây, ghi chú "Đã dừng ghép nối.", trang trên điện thoại không còn mở được. Đồng bộ khi mất mạng giữa chừng (nếu thử): "Lần đồng bộ cuối lỗi" với lý do, không sập | `wifi:`/(syslog) kết nối; `pairing stopped by the user`, `pairing ended: stopped` | ☐ | ☐ |

Kiểm trên Mac, ở lần cắm USB kế tiếp (bước đầu của FI-01, người bảo trì; không sửa tệp, không đưa nội dung `data/token` vào chat hay log):

1. `Books/GitHub/` có sách đã đồng bộ với tên **chuẩn NFC** (lệnh `python3 -c "import os,sys,unicodedata as u;print([n for n in os.listdir(sys.argv[1]) if u.normalize('NFC',n)!=n])" <thư mục Books/GitHub>` in `[]`); có `Documents/GitHub/mau.pdf` nếu đã thêm PDF; **không** có `gia.epub`, `ghi-chu.txt`, và không có tệp `.ghsync-*.part` nào.
2. `.adds/apps/ghsync/data/` chỉ có `token`, `config.json`, `state.json`, `log.txt`. `grep -c "github_pat_\|ghp_"` trên `log.txt`, `state.json`, `config.json` và `.adds/nickeldrawer/debug.log` phải ra **0**. Chỉ kiểm sự có mặt của `data/token`, không in nội dung.
3. `state.json` có mục `files` cho từng sách và `last_error` rỗng sau lần đồng bộ thành công; `config.json` không có khóa `dest` (xếp theo loại).
4. Sau buổi: **xóa token trên GitHub** (Settings > Developer settings), và xóa `data/token` trên máy nếu không dùng nữa.

## 5. Hình nền, Thư viện, giao diện (khoảng 20 phút)

Mở Ngăn, bấm **Hình nền** lần đầu sau khi chép ảnh.

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| WP-01 | Mở Hình nền (Pictures có 13 ảnh, chưa nhập). Chờ | Tự nhập, không toast, không bấm gì. Tiêu đề hiện "Đang nhập n/13"; dòng "Đang chuyển đổi n/13 · tên"; các ô nét đứt chờ; ảnh dần hiện (ô chưa có ảnh nhỏ là biểu tượng mờ, rồi thành ảnh thu nhỏ). Màn không đứng | `screen: "main" page ...` lặp lại khoảng 5 giây/lần; ảnh hiện từng phần, không nháy cả màn | ☐ | ☐ |
| WP-02 | Xong nhập: xem lưới | Lưới 4 cột × 3 hàng = 12 ô, ảnh 3:4 có viền mảnh, không chú thích. Dòng tóm tắt "13 ảnh · Đang dùng: … · Lấp đầy". Ảnh đang dùng có viền dày và dấu ✓ tròn đen. Dưới có "‹ 1 / 2 ›". Nút phím lật trang qua trang 2 (1 ảnh) và về | `screen: page 1 -> 2`, `screen: page 2 -> 1` | ☐ | ☐ |
| WP-03 | Bấm một ô. Trong trình xem: bấm 1/3 bên phải, 1/3 bên trái, phím lật trang; đến ảnh đầu bấm trái; đến ảnh cuối bấm phải | Tiêu đề "← n / 13". Dưới có "Xóa…" (trái) và "Đặt làm hình nền". Mỗi lần đổi ảnh là một lần làm mới đầy màn (không để lại bóng). Ở đầu/cuối: không có gì xảy ra, không vòng lại | `viewer: picture 3 -> 4 of 13`, `viewer: no previous picture ...`/`viewer: no next picture ...` | ☐ | ☐ |
| WP-04 | Bấm giữa ảnh (ẩn thanh). Xem góc trên trái. Bấm nút tròn trắng ở góc | Ảnh phủ đầy dưới thanh trạng thái; góc trên trái có nút tròn trắng viền đen với mũi tên ←, nằm trên mọi ảnh. Bấm nút: thanh hiện lại (ảnh **không** lật). Bấm giữa lần nữa cũng đổi qua lại. Ở trạng thái ẩn, bấm 1/3 trái ngoài nút: lật ảnh trước; ở ảnh đầu: không có gì | `viewer: chrome hidden, ...`, `viewer: chrome shown, ...`, `viewer: back button tapped with the chrome already shown, ignored` (nếu bấm khi đã hiện) | ☐ | ☐ |
| NAV-01 | Từ trình xem (thanh hiện) bấm ← ở tiêu đề. Làm lại: mở ảnh số 13 (trang 2), ẩn thanh, bấm nút tròn rồi ← | Về lưới đúng **trang chứa ảnh vừa xem** (trang 2). Một lần ← chỉ lùi **một tầng**: toàn màn → trình xem → lưới → (←) Ngăn (danh sách/lưới như trước) | `screen: back tapped on "viewer"`, `navigation: pop ...`, `navigation: top value now ...` | ☐ | ☐ |
| WP-05 | Trong trình xem một ảnh chưa dùng, bấm "Đặt làm hình nền" | Toast "Đã đặt làm hình nền"; tiêu đề hiện "Đang dùng ✓"; nút chính thành "Đang dùng" và mờ. Quay lại lưới: dấu ✓ chuyển sang ảnh này. Không hộp thoại | `queue: set-current for ...: queued`, `queue: sending set-current` | ☐ | ☐ |
| WP-06 | Trong trình xem bấm "Xóa…". Bấm ra ngoài hộp. Bấm **Hủy**. Lại "Xóa…", bấm **Xóa** | Hộp "Xóa ảnh này?" kèm câu giữ ảnh gốc, hai nút Hủy/Xóa; bấm ra ngoài **không** đóng; Hủy không xóa; Xóa thì hiện ảnh kế (hoặc trước đó nếu là cuối), toast, tổng còn 12. Ảnh gốc vẫn ở `Pictures/` (Mac kiểm) | `plain dialog ...: outside tap ignored (destructive question)`, `queue: remove for ...` | ☐ | ☐ |
| WP-07 | Ở lưới, nhấn giữ một ô | Menu Nickel có tiêu đề là tên ảnh: Đặt làm hình nền, Xem, Thông tin, Xóa khỏi Hình nền…. Thông tin mở hộp tên/kích thước/ngày, bấm ra ngoài thì đóng. "Xem" mở trình xem. "Xóa khỏi Hình nền…" hỏi xác nhận như WP-06; Hủy thì giữ. "Xóa cả ảnh gốc…" hỏi "Xóa cả ảnh gốc?" (nút Xóa cả hai / Hủy): Hủy thì giữ cả hai; Xóa cả hai thì ảnh biến khỏi lưới **và** tệp gốc trong `Pictures/` bị xóa (Mac kiểm; chỉ khi tệp giống hệt, nếu đã sửa thì gốc còn) | `tile menu: "Thông tin" chosen`, `tile menu: "Xóa khỏi Hình nền…" chosen, asking first`, `tile menu: "Xóa cả ảnh gốc…" chosen, asking first`, `tile menu: dismissed` | ☐ | ☐ |
| WP-08 | Ở lưới bấm biểu tượng bánh răng (Tùy chọn). Xem, rồi chọn "Vừa khung", "Ngẫu nhiên", rồi quay "Lấp đầy", "Cố định" | Tiêu đề "← Tùy chọn". Mục CÁCH HIỂN THỊ: Lấp đầy màn hình / Vừa khung, mỗi cái có dòng phụ. Mục ĐỔI ẢNH: Cố định / Ngẫu nhiên, có dòng phụ. Mục MÀN HÌNH NGỦ: "Sẵn sàng ✓" hoặc gợi ý tên cài đặt Nickel. Chọn xong **không đứng máy** (không chờ 7 giây): lưới có chữ "Đang chuyển đổi n/12" rồi xong. Không hộp thoại văn bản. Mỗi radio đổi ngay | `screen: tap on radio N, action "mode"`, `queue: mode for ...: queued`; không có `tap has no action` | ☐ | ☐ |
| NAV-03 | Trong Tùy chọn bấm ←. Trong lưới bấm ←. Trong Ngăn bấm ← | Tùy chọn → lưới → Ngăn → Thêm, mỗi ← lùi một tầng. Mũi tên có phản hồi (tròn đen khi chạm) | `screen: back tapped on "options"`, `screen: back tapped on "main"`, `navigator: back` | ☐ | ☐ |
| NAV-02 | Mở Hình nền → trang 2 → một ảnh → bấm tab **Trang chủ**. Rồi Thêm → Ứng dụng → Hình nền | Bấm tab xóa hết ngăn xếp: Trang chủ sạch; Thêm hiện danh sách gốc, không còn Ngăn bỏ lại; Hình nền mở lại từ trang 1 của lưới, không phải trình xem | `navigation: cleared N level(s), M of them screens or modals, ...`; không có `drawer controller` bị rò | ☐ | ☐ |
| KIT-03 | Trong Tùy chọn bấm hai radio khác nhau thật nhanh, rồi cùng một radio hai lần | Cả hai bấm đều được áp dụng (không mất lần nhấn thứ hai); chỉ vùng radio vẽ lại | `queue: ... replaced the queued event of the same control` hoặc hai dòng `queued`; không có `tap has no action` | ☐ | ☐ |
| KIT-10 | Trong lưới bấm biểu tượng Nhập ảnh 5 lần trong 1 giây | Tối đa một toast mỗi 2 giây hiện, còn lại bị bỏ; không treo, không xếp hàng toast dài | `bridge: toast queued` ít hơn số lần bấm | ☐ | ☐ |
| KIT-09 | Trong Thư viện (xem LIB-02) bấm nút có hộp xác nhận, rồi Hủy | Hộp Có/Không kiểu Nickel; Hủy không gửi gì; OK mới gửi | `tile menu: ... asking first` hoặc `queue:` chỉ xuất hiện sau OK | ☐ | ☐ |
| LIB-01 | Mở **Thư viện**, bấm "Quét" | Thanh tiến độ "Đang quét n/N"; sau đó số Sách/Sai ngôn ngữ/Chưa chuẩn NFC/Nhóm trùng/Lần quét cuối. Nhóm trùng có ít nhất 1 (hai sách thử). Không hộp thoại văn bản | `screen: tap on button N, action "scan"`, `queue: scan ...` | ☐ | ☐ |
| LIB-02 | Chạm vào một sách trong "Sách trùng" rồi "Bỏ vào thùng rác", Hủy rồi lại làm và bấm Chuyển | Dòng "Đã chọn: …"; hộp xác nhận; Hủy không làm gì; Chuyển thì toast "Đã chuyển vào thùng rác 30 ngày.", tệp vào `data/trash/` (Mac kiểm), quét lại thì chỉ còn một bản. Không bao giờ xóa bản cuối | `tile menu:`/`queue: trash ...` | ☐ | ☐ |
| LIB-03 | Bấm "Sửa ngôn ngữ", OK. Rồi mở sách thử trong Thư viện (có thể cần quét lại) | Tiến độ "Đang sửa n/N"; sách đổi ngôn ngữ thành `vi` (Mac kiểm OPF và XHTML `lang`); sách vẫn mở và không hỏng; bản gốc ở `data/backups/<ngày>`. "Hoàn tác lần sửa" trả lại | `queue: fix ...` | ☐ | ☐ |
| LIB-04 | Xem số "Chưa chuẩn NFC" sau quét | Hiển thị đúng đếm (bản thử có tên không chuẩn nếu có; nếu không, 0). Không tự đổi tên tệp | (số trên màn) | ☐ | ☐ |
| LIB-05 | Bấm Quét, rồi ngay lập tức bấm ← ra khỏi ứng dụng, đợi 30 giây, mở lại | Không treo, không lỗi; mở lại thấy kết quả quét hoàn chỉnh hoặc tiến độ; không còn tệp dở dang trong thư viện | `screen: closing with N queued event(s) unsent` (nếu có), `screen: back tapped on "main"` | ☐ | ☐ |
| API-01 | Suốt mục 3 đến 5 và mục 6: ghi lại mọi hộp thoại hiện ra | Hộp thoại chỉ là hộp gốc của Nickel (xác nhận, thông tin, menu). **Không** có hộp chứa văn bản do tập lệnh của ứng dụng (kiểu "LocalSend đang nhận sách … PIN: …"). Thiếu dữ liệu hoặc lỗi chỉ cho toast ngắn. Không NickelMenu ở bất cứ chỗ nào của Ngăn | `popup:`/`bridge: settings dialog`/`bridge: toast on window`; không dòng nào hiện văn bản của ứng dụng | ☐ | ☐ |
| KIT-01 | Xem một lượt các màn: LocalSend, Hình nền (lưới, trình xem, Tùy chọn, trạng thái trống ở WP-00), Thư viện | Tiêu đề, văn bản, KV, BigCode, danh sách, nút, radio đều hiện đủ, không cắt chữ, tiếng Việt đủ dấu, không ô trống | `screen: "..." page ..., N tap target(s)` | ☐ | ☐ |
| KIT-02 | So cạnh Cài đặt › Đọc sách (hoặc Cài đặt gốc) với Tinh chỉnh và Hình nền › Tùy chọn. Cho điểm (S11) | Chiều cao hàng, đường kẻ, độ đậm chữ giống nhau. Điểm của bạn: ___ / 5 (cần từ 4) | — | ☐ | ☐ |

Hiển thị (khoảng 5 phút). Làm xong thì trả lại cài đặt.

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| KIT-11 (và DRW-08) | Bật chế độ tối (Dark Mode ở menu đọc sách hoặc Cài đặt). Mở Ngăn, LocalSend, Hình nền (lưới và trình xem), Thư viện, Tinh chỉnh | Mọi thứ đảo màu đúng; biểu tượng đổi theo; không hộp trắng; ảnh trong lưới vẫn đúng (ảnh không đảo) | `icon ...` không báo thiếu | ☐ | ☐ |
| KIT-12 (và DRW-09) | Đổi cỡ chữ nhỏ nhất, rồi lớn nhất. Mở Ngăn, Tinh chỉnh, Thư viện, Hình nền > Tùy chọn | Bố cục co giãn, không bị cắt chữ; vùng chạm đủ lớn (khoảng 120 px). | `screen: ... tap target(s)` | ☐ | ☐ |

## 5b. Việc cần làm và Bộ giao diện (khoảng 15 phút)

Việc cần làm là ứng dụng thật (tên tiếng Việt "Việc cần làm", tiếng Anh "To-do"). Bộ giao diện (UIK-*) là mục tiêu thử các ca KIT-*; cần bật theo ghi chú ở mục 1.

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| TODO-01 | Mở **Việc cần làm**. Bấm ô "Việc mới", gõ bằng bàn phím Nickel (bố cục tiếng Việt, hoặc giữ phím chữ để chọn dấu): `Đọc Truyện Kiều ở Huế`, xác nhận | Bàn phím gốc của Nickel hiện lên, chữ có đủ dấu khi gõ. Sau xác nhận: dòng mới hiện ở cuối danh sách, ô nhập trống lại, toast "Đã thêm việc", tiêu đề phụ "Còn 1 · xong 0". Mac (sau, qua USB): `data/todo.json` là JSON hợp lệ và chữ ở dạng NFC (`python3 -c "import json,unicodedata as u;t=json.load(open('todo.json'))['tasks'][0]['text'];print(t==u.normalize('NFC',t))"` in True) | `screen: tap on field`, `queue: newitem` | ☐ | ☐ |
| TODO-02 | Thêm thêm 2 việc có dấu (`Mua sữa`, `Gọi cho cô Mai`), đánh dấu một việc xong. Ghi lại thứ tự và dấu ✓. Tắt máy hẳn (giữ nút nguồn, Tắt), bật lại, mở lại Việc cần làm | Danh sách giống hệt: cùng 3 việc, cùng thứ tự, đúng dấu ✓ và số đếm. Mac: không có tệp `.tmp-todo-*` trong `data/` | `screen: "main" page ...` | ☐ | ☐ |
| TODO-03 | Chạm một việc chưa xong, rồi chạm lại. Thử một việc đã xong | Chạm lần đầu: hiện ✓ và số "xong" tăng, không toast. Chạm lại: ✓ mất. Danh sách không nhảy thứ tự | `queue: toggle ...` | ☐ | ☐ |
| TODO-04 | Nhấn giữ một việc, chọn "Xóa...". Bấm Hủy. Lại làm và bấm Xóa | Menu Nickel có "Xóa..."; hộp xác nhận kiểu Nickel; Hủy không xóa; Xóa thì dòng biến mất, toast "Đã xóa việc", số đếm giảm | `tile menu: ... asking first` hoặc `list menu:` rồi `queue: remove ...` | ☐ | ☐ |
| TODO-05 | Khi chưa có việc xong, xem nút "Xóa việc đã xong". Đánh dấu hai việc xong, bấm nút, bấm Hủy, rồi bấm lại và Xóa. Thêm lại đúng một việc đã có; bấm xác nhận khi ô trống | Chưa có việc xong: nút mờ, chạm không gửi gì. Có việc xong: hộp xác nhận; Hủy không làm gì; Xóa xóa hết việc xong, toast "Đã xóa việc đã xong". Thêm trùng: toast "Việc này đã có", danh sách không đổi. Ô trống: toast "Hãy gõ việc trước" | `queue: clear-done ...` | ☐ | ☐ |
| TODO-06 | (Mac, qua USB, máy đang mở Ngăn) sao lưu `data/todo.json` vào `backups/<ngày>/`, rồi chép `local/todo-hong.json` đè lên `data/todo.json`. Mở lại Việc cần làm. Thêm một việc. Trả lại tệp đã sao lưu | Mở lại: danh sách trống và có khung ghi chú "Danh sách đã lưu bị hỏng…"; `data/todo.json.corrupt` giữ đúng nội dung hỏng; sau khi thêm một việc khung ghi chú biến mất và danh sách chỉ có việc mới. Không treo, không lỗi hộp thoại | `screen: "main" ...` | ☐ | ☐ |
| UIK-01 (KIT-01, KIT-06, KIT-07, KIT-08) | Mở **Bộ giao diện**. Lật hết 3 trang của màn chính; bấm ← ở một trang con, rồi mở lại | Danh sách chia nhóm: Thao tác, Lựa chọn, Nhập liệu (trang 1), Phản hồi, Điều hướng (trang 2), Bố cục (trang 3), tổng 18 dòng; mỗi nhóm nằm trọn một trang. Phím lật trang và thanh trang đều lật; ← về đúng trang đã mở. Tiếng Việt đủ dấu | `screen: "main" page n/N, ... tap target(s)` | ☐ | ☐ |
| UIK-02 | Mở lần lượt từng trang thành phần | Mỗi trang mở dưới 1 giây, có mô tả, bảng **Trạng thái** (`samples`) ở trang 1, ô **Thử**, và dòng Sự kiện cuối / Giá trị sau lần chạm đầu. Mẫu "Đang nhấn" là đảo màu đúng của mẫu "Bình thường" (nền đen, chữ trắng) | `screen: "button" ...` | ☐ | ☐ |
| UIK-03 | Trang **Nút**: tắt công tắc "Bật nút chính", bấm nút chính; bật lại và bấm | Công tắc tắt thì nút chính mờ và không gửi gì; bật thì bấm được. Trong 1 giây dòng "Sự kiện cuối" hiện `action` | `queue: echo ...` | ☐ | ☐ |
| UIK-04 | Trang **Hộp thoại**: bấm "Hỏi một câu", chờ không bấm gì 120 giây; mở lại, bấm Xóa; bấm "Hiện thông tin" | Hộp xác nhận kiểu Nickel (`N3Dialog` trong log), Hủy bên trái, Xóa bên phải; phím lật trang không làm gì khi hộp đang mở; sau 120 giây đóng như Hủy; Xóa gửi sự kiện; hộp thông tin có một nút, chạm ngoài là đóng | `N3Dialog` | ☐ | ☐ |
| UIK-05 | Trang **Ô nhập**: gõ `Truyện Kiều` vào "Tìm kiếm" (bàn phím Việt), bấm Go; gõ `12` vào "Đến trang" (bàn phím số) | Bàn phím gốc của Nickel; Go lưu; trang "Sự kiện cuối" hiện `input` và giá trị; mở lại trang vẫn thấy giá trị | `queue: searchtext ...`, `queue: pagenumber ...` | ☐ | ☐ |
| UIK-06 | Trang **Công tắc**: tắt công tắc "Cho phép", rồi chạm công tắc "Tự cập nhật thư viện" | Công tắc đầu mờ và không phản ứng; bật "Cho phép" lại thì dùng được | `queue: sampleenabled ...` | ☐ | ☐ |
| UIK-07 | Trang **Tiến trình**: bấm "Chạy 3 bước" | Ba lần cập nhật cách nhau 5 giây ("Đang tải 1 / 3" … "3 / 3"), rồi "Đã đồng bộ 3 sách" kèm một lần làm mới toàn màn; giữa các lần không vẽ lại gì; nút mờ khi đang chạy | `screen: ...` | ☐ | ☐ |
| UIK-08 | Chép 1 đến 3 ảnh PNG (dưới 1 MB) vào `Pictures`. Trang **Thư viện và xem ảnh**: chạm một ảnh, mở toàn màn, bấm ← ba lần | Lưới 4 cột; chạm mở trình xem; ← về lưới, ← về trang Bố cục, ← về màn chính; độ sâu không quá 2 (log). Không có PNG: màn trống "Chưa có ảnh PNG" | `screen: "gallery" ...`, `navigator: depth` | ☐ | ☐ |
| UIK-09 | Xem mọi trang bằng tiếng Việt, kể cả nút 168 px và nhãn mục | Không bị cắt dấu ở nhãn mục hoặc nút; không chuỗi nào dài quá 150% bản tiếng Anh (validator) | (xem màn hình) | ☐ | ☐ |
| UIK-10 | (Mac) `go run ./cmd/appsctl validate --apps-dir apps` | `uikit` báo `ok`, không cảnh báo | | ☐ | ☐ |

Các ca KIT-* còn lại làm trên Bộ giao diện khi tiện: KIT-02 (so với Cài đặt), KIT-11 (chế độ tối) và KIT-12 (cỡ chữ) có thể làm với ba màn của nó (Màn chính, Thêm thành phần, Thư viện ảnh) thay cho, hoặc thêm vào, các màn ở mục 5. Cuối buổi tắt lại công tắc Bộ giao diện trong ⚙ (SET-02).

## 6. Ba hành động cuối của Tinh chỉnh (khoảng 8 phút)

Làm theo đúng thứ tự này. Trả công tắc về như cũ trước (TWK-02..05 đã xong). Cho Hình nền một ảnh đang dùng và một sách đang đọc.

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| TWK-10 | Mở Ngăn (hoặc một màn ứng dụng). Mở Tinh chỉnh, bấm "Ngủ". Chờ 1 phút, bấm nút nguồn để đánh thức | Toast "Đang vào chế độ ngủ"; máy ngủ. **Màn hình ngủ hiện ảnh Hình nền toàn màn hình** (WP-09). Đánh thức: về đúng màn hình trước đó, không vệt (DRW-10, FI-10). Nếu không thấy ảnh: kiểm "Hiển thị sách đang đọc" và "Hiển thị bìa sách toàn màn hình" ở Cài đặt › Tiết kiệm pin và riêng tư | `tweaks: "sleep" applied`, `bridge: power action sleep` | ☐ | ☐ |
| WP-09 | Ngay sau TWK-10, xem lại ảnh ngủ | Ảnh đặt ở WP-05 hiện toàn màn hình, đúng chế độ Lấp đầy/Vừa khung đã chọn | (xem màn hình) | ☐ | ☐ |
| WP-10 | Hình nền có ≥3 ảnh. Tùy chọn → Ngẫu nhiên. Cho máy ngủ rồi đánh thức 6 lần (thử cả lúc đèn bật và tắt). Sau đó chọn Cố định và ngủ/thức thêm 1 lần | Mỗi lần ngủ hiện ảnh khác lần trước, không bao giờ A→B→A. `.kobo/screensaver` luôn chỉ có một tệp `nd-wallpaper-*.png`; dấu ✓ ở lưới đi theo ảnh sẽ hiện lần sau. Với 2 ảnh: luân phiên; 1 ảnh: giữ nguyên, không lỗi. Sau khi chọn Cố định: thức dậy không đổi ảnh nữa, tệp `data/rotate-on-wake` biến mất | `wake rotation: started wallpaper next` một lần mỗi lần thức | ☐ | ☐ |
| TWK-11 | Mở Tinh chỉnh, bấm "Khởi động lại", **OK** | Toast "Đang khởi động lại", rồi máy khởi động. Sau khi lên: các công tắc còn đúng cài đặt cuối, Hình nền còn dữ liệu, Tinh chỉnh mở lại bình thường | `tweaks: "reboot" confirmed`, `tweaks: "reboot" applied`, `bridge: power action reboot` | ☐ | ☐ |
| TWK-12 | Mở Tinh chỉnh, bấm "Đảo màu và khởi động lại", OK. Sau khi lên, mở lại và chạy **một lần nữa** để trả lại | Máy khởi động với màu bị đảo ngược; lần chạy thứ hai trả lại bình thường. Không kẹt ở màu đảo | `tweaks: "invert-reboot" confirmed`, `tweaks: "invert-reboot" applied` ×2 | ☐ | ☐ |

Nếu kẹt ở màu đảo: cắm USB, người bảo trì đặt `InvertScreen=false` trong `Kobo eReader.conf` (sau khi sao lưu), khởi động lại.

## 7. Cố ý làm hỏng (cuối buổi, mỗi bước có cách khôi phục)

Từ bước này Ngăn có thể mất tạm; không sao. Làm lần lượt. Mỗi bước: người bảo trì cắm USB, làm xong rồi `eject`.

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| FI-01 | Người bảo trì (U3): đổi tên `.adds/koboapps/bin/appsctl` thành `appsctl.off`, `eject`. Người dùng: mở Ngăn, bấm Tinh chỉnh | Nickel **không sập**. Ngăn trống hoặc báo lỗi ngắn, Tinh chỉnh báo toast "không dùng được" thay vì cửa sổ rỗng; ✕/← vẫn đóng được | `tweaks: list unavailable: ...`; (syslog) `appsctl failed to start: ...` | ☐ | ☐ |

Khôi phục FI-01: cắm USB, đổi `appsctl.off` về `appsctl`. Nếu máy không nhận USB: Nickel vẫn chạy được (mod không treo), khởi động lại rồi thử lại.

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| FI-03 | Người bảo trì (U4): trả `appsctl`; `flags debug screens disable`; `eject`. Người dùng: khởi động lại (nút nguồn) và mở Thêm | Mod tải nhưng **im lặng**: Thêm chỉ có các hàng gốc, **không** hàng Ứng dụng, **không** Tinh chỉnh; không toast lúc khởi động; mọi chức năng của Nickel bình thường. Lệnh khôi phục hoạt động | (syslog) `kill switch ... present, staying inert`; trong `debug.log` không có `entry row:` | ☐ | ☐ |
| FI-04 (tùy chọn) | Người bảo trì (U5): xóa cờ `disable` (`flags debug screens`), **sao lưu** `.kobo/version`, sửa trường firmware (trường thứ 3) thành `4.99.99999` trên bản chép rồi ghi lại, `eject`. Người dùng: khởi động lại, mở Thêm | Bản firmware lạ: mod tắt ngăn, hàng Ứng dụng/Tinh chỉnh không có, Nickel bình thường | (syslog) `bridge: firmware 4.99.99999 not in the tested list, drawer disabled` | ☐ | ☐ |
| FI-13 | Người bảo trì (U5b): **khôi phục `.kobo/version`** nếu đã làm FI-04; `flags debug screens test-unknown-model`; `eject`. Người dùng: khởi động lại, mở Thêm, chờ 5 giây | Mẫu máy "lạ": Thêm chỉ có các hàng gốc, **không** hàng Ứng dụng/Tinh chỉnh; đúng **một** toast "NickelDrawer chưa hỗ trợ máy này" (en: "NickelDrawer does not support this model yet") vài giây sau khi mở Thêm; Nickel bình thường. Cờ được gỡ ở bước SAFE-03 | `bridge: test flag /mnt/onboard/.adds/nickeldrawer/test-unknown-model present, device 00000000-0000-0000-0000-000000000388 treated as unsupported, drawer disabled (no More rows)`; (syslog) `bridge: telling the user the drawer is disabled` | ☐ | ☐ |
| SAFE-03 (và SAFE-05, S10) | Người bảo trì (U6): **khôi phục `.kobo/version`** từ bản sao lưu nếu đã làm FI-04 mà chưa khôi phục; `flags debug screens uninstall`; `eject`. Người dùng: khởi động lại | Không toast; cờ `uninstall` bị tiêu thụ; mod đã biến mất (Thêm chỉ có hàng gốc); `.adds/apps` còn nguyên; tab NickelMenu vẫn tắt (không quay lại) | (syslog) NickelHook gỡ mod; `.adds/nickeldrawer/uninstall` không còn | ☐ | ☐ |

Khôi phục: nếu `.kobo/version` bị sửa nhầm hoặc máy báo lạ, chép bản sao lưu về `.kobo/version`. Nếu muốn dùng lại Ngăn sau buổi thử: stage lại `KoboRoot-nickeldrawer.tgz` và đặt `flags debug`.

Hoàn tất (người bảo trì): `tools/device-session.sh log` (lưu `backups/`), dọn thư mục thử (`Books/ndtest/`, ba ứng dụng giả, ảnh thử nếu không giữ), `clean`, `eject`. Trả lại cài đặt công tắc đã ghi ở mục 1. Pin lúc cuối: ____ %. Thời gian: ____ phút.

## 8. Đo S-criteria

| ID | Giá trị đo | Ngưỡng | Cách đo |
|---|---|---|---|
| S1 mở Ngăn | ____ ms (3 ứng dụng), ____ ms (20+ ứng dụng) | ≤ 600 ms, ≤ 1000 ms | Hiệu giữa `entry row: opening the drawer` và `drawer controller: viewWillAppear`, kèm đồng hồ bấm tay |
| S2 mở màn ứng dụng | ____ ms | ≤ 800 ms | Hiệu giữa bấm hàng và dòng `screen: "main" page ...` đầu tiên |
| S4, S5 (không ngâm 72 giờ) | không đo | — | Bỏ theo DECISIONS 2026-10-10. Ghi pin lúc đầu và cuối buổi và xem `debug.log` có lỗi lạ không |
| S6, S7 (CPU, bộ nhớ) | chỉ ghi nhận bằng mắt; không đo được nếu không có shell | — | Nếu có telnet: `top`, `/proc/<pid>/status` trước và sau khi mở 10 màn |
| S8 USB | ☐ đạt ☐ không | USB khi mở màn ứng dụng hoạt động, màn đóng sạch | LS-07 |
| S11 giao diện | ____ / 5 | ≥ 4 | KIT-02 |
| S12 chịu lỗi | ____ / ____ ca FI không làm sập Nickel | 100 % | Mục 7 và FI-02, FI-11 |

## 9. Ghi chú (người điều phối và người dùng)

Ghi bước, văn bản đúng trên máy, thời gian; ảnh chụp màn hình nếu lỗi (không đăng ảnh có thông tin riêng):

____________________________________________________________

## 10. Ký xác nhận (mẫu §9.10, cổng G3, G4, G5)

Ngày: ____ · Bản mod (hash): ____ · Phiên: T-final

| Cổng | Tiêu chí (đã cập nhật theo thiết kế hiện tại) | Ca | Kết quả | Quyết định | Ký |
|---|---|---|---|---|---|
| **G3** (Ngăn) | DRW-01..05, 07, 12, 13, 14 đạt; TWK-01..12 đạt; NAV-01..03 đạt; S1 đạt; S8 đạt (LS-07); mô phỏng firmware lạ/mẫu máy lạ/cờ `disable` an toàn (FI-03, FI-04 hoặc FW-03 tĩnh, FI-13); SAFE-03/04/06 đạt; S4 không đo (bỏ ngâm) | DRW-*, TWK-*, NAV-*, ICO-01, API-01, SAFE-*, FI-03/04/13 | ☐ Đạt ☐ Lỗi | ☐ go ☐ no-go | ____ |
| **G4** (UI kit) | KIT-01..04, 09..12 đạt (làm trên Bộ giao diện: UIK-01..08; ô nhập còn qua TODO-01); S11 ≥ 4 trên 5; golden test và test đơn vị xanh (`make golden-test`); KIT-05 hoãn (không có ô bí mật), KIT-13 kiểm trên Mac | KIT-*, UIK-*, TODO-*, FI-02, FI-11 | ☐ Đạt ☐ Lỗi | ☐ go ☐ no-go | ____ |
| **G5** (Ứng dụng) | LS-01..07, WP-00..09, LIB-01..05, TODO-01..06 đạt; FI-01..04 đạt, FI-11 đạt; S12 đạt; GH-01..08 đạt; S5..S7 ghi nhận không đo (bỏ ngâm); giao diện chỉ dùng API NickelDrawer (API-01) | LS-*, WP-*, LIB-*, TODO-*, GH-*, FI-* | ☐ Đạt ☐ Lỗi | ☐ go ☐ no-go | ____ |

Việc còn mở (liên kết báo cáo chặn `docs/blockers/B-NNN.md` nếu có): ____________________

Chữ ký người dùng: ____ · Ngày: ____
