# Extras: buổi thử trên máy (G1, chỉ khi có người dùng bên cạnh)
> Thuộc về helper **Extras** ([../README.md](../README.md)), không thuộc NickelDrawer. Chưa chạy trên máy thật. Máy: Kobo Libra 2, firmware 4.38. Chỉ người dùng chạm vào máy; người bảo trì làm phần Mac qua USB. Không bao giờ `cd` vào `/Volumes/KOBOeReader`.
## 0. Trước khi cài
- ☐ Sao lưu máy (`backups/<ngày>/`, kèm `KoboReader.sqlite` và `Kobo eReader.conf`). Pin từ 50 % trở lên.
- ☐ `make build`, `make -C helpers/extras docker-build`, `helpers/extras/check-symbols.sh`, `helpers/extras/test-install.sh` đều xanh.
- ☐ **Diễn tập gỡ và failsafe trước** (cổng G1): tạo tệp rỗng `.adds/helpers/extras/uninstall`, khởi động lại, xác nhận `libextras.so` biến mất; rồi cài lại và tắt nguồn trong 15 giây đầu sau khi mod nạp, xác nhận mod bị bỏ ở lần khởi động sau.
- ☐ Máy đang có helper cũ (`.adds/helpers/screenshot`, `libstatusbarshot.so`)? `install.sh` gỡ nó cùng lần khởi động; ghi lại: ____
- ☐ Ghi trạng thái ban đầu của các công tắc Tinh chỉnh để trả lại cuối buổi: Màn hình chờ ____ · Nút chụp màn hình ____ · Tự kết nối USB ____ · Chữ rõ nét ____
## 1. Cài chỉ Extras (máy không có NickelDrawer: không `.adds/koboapps`, không `libnickeldrawer.so`)
| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| EXT-01 | `helpers/extras/install.sh`, `eject`, khởi động lại | Nickel lên bình thường; không toast lỗi | (syslog) `(extras) active` | ☐ | ☐ |
| EXT-02 | Nhìn thanh trạng thái trên Trang chủ | Có nút chụp màn hình ngay trái biểu tượng độ sáng; **cao bằng và nét dày bằng** các biểu tượng của Nickel (không nằm cao hơn); khoảng cách đều | `button added …`, `lightIcon …` và `button …` (hình học sau khi bố trí) | ☐ | ☐ |
| EXT-03 | Bấm giữ rồi thả nút chụp | Hiện đĩa đen, toast "Đã chụp màn hình" kèm tên tệp; `screen_001.png` ở thư mục gốc, ảnh không có nút, đĩa hay toast | `capture saved: …` | ☐ | ☐ |
| EXT-04 | Mở tab Thêm | Có hàng "Tinh chỉnh" (biểu tượng bánh răng của Nickel) ở cuối nhóm; chạm mở cửa sổ kiểu Cài đặt | `more row: … added` | ☐ | ☐ |
| EXT-05 | Bấm Thêm, bấm hàng "Tinh chỉnh". Ghi lại các công tắc đang Bật hay Tắt | Cửa sổ kiểu Cài đặt của Kobo: tiêu đề nhỏ ở giữa, nút ✕, tiêu đề lớn có chân, các hàng chữ nghiêng có đường kẻ, danh sách 2 cột. Có: Màn hình chờ, Nút chụp màn hình, Tự kết nối USB, Chữ rõ nét, Địa chỉ IP, Đảo màu và khởi động lại, Ngủ, Khởi động lại, Quét lại thư viện. Ba hành động Ngủ, Khởi động lại, Đảo màu và khởi động lại nằm cuối. Công tắc hiện Bật/Tắt | `tweaks: Nickel N3Dialog ... pushed`, `tweaks panel: 9 tweak(s), N page(s)` | ☐ | ☐ |
| EXT-06 | Bấm "Nút chụp màn hình" (dòng nhỏ "Trên thanh trạng thái") | Đổi Bật/Tắt ngay, toast "Đã tắt Nút chụp màn hình" (hoặc "Đã bật …"). Đóng cửa sổ: nút trên thanh trạng thái biến mất (hoặc hiện lại) khi thanh hiện lại; tệp `screenshot-off` có hoặc không trong `.adds/helpers/extras`. Để **Bật** ở cuối | `tweaks panel: tap on "screenshot-button"`, `tweaks: "screenshot-button" applied` | ☐ | ☐ |
| EXT-07 | Bấm "Tự kết nối USB" để đổi, ghi trạng thái | Đổi Bật/Tắt, có toast. Để nguyên cho đến khi người bảo trì kiểm trên Mac ở mục 4, rồi trả lại | `tweaks panel: tap on "auto-usb"`, `tweaks: "auto-usb" applied` | ☐ | ☐ |
| EXT-08 | Bấm "Chữ rõ nét" | Đổi Bật/Tắt; toast nói "Mở lại sách để áp dụng". Trả lại như cũ ở cuối | `tweaks panel: tap on "legibility"`, `tweaks: "legibility" applied` | ☐ | ☐ |
| EXT-09 | Bấm "Màn hình chờ" (tắt), rồi bấm lại (bật) | Mỗi lần đổi ngay kèm toast. Cuối cùng để **Bật**. Mac kiểm: thư mục `.kobo/screensaver` đổi tên thành `.kobo/screensaver.disabled` khi Tắt và về lại khi Bật; ảnh không mất | `tweaks panel: tap on "screensaver"`, `tweaks: "screensaver" applied` ×2 | ☐ | ☐ |
| EXT-10 | Xem hàng "Địa chỉ IP". Bấm vào nó | Hiện địa chỉ IP của Wi-Fi (hoặc "Không có Wi‑Fi"). Bấm không làm gì | `tweaks panel: tap on "ip"` (không có dòng `applied`) | ☐ | ☐ |
| EXT-11 | Bấm "Quét lại thư viện" | Toast "Đang quét thư viện". Máy quét nhanh, không treo, không hộp thoại | `tweaks: "library-rescan" applied` | ☐ | ☐ |
| EXT-12 | Bấm "Khởi động lại" rồi chọn **Hủy**. Làm tương tự với "Đảo màu và khởi động lại" | Có hộp xác nhận ("Khởi động lại ngay?", "Đảo màu màn hình và khởi động lại ngay?"). Hủy thì không có gì xảy ra | `tweaks: "reboot" cancelled`, `tweaks: "invert-reboot" cancelled` | ☐ | ☐ |
| EXT-13 | Nếu cửa sổ có hơn 1 trang: lật bằng phím chuyển trang. Bấm ✕. Mở lại, bấm ra ngoài khung | Phím lật trang đổi trang; ✕ và bấm ngoài đều đóng; Thêm hiện lại bình thường | `tweaks panel: page 1 -> 2`, `tweaks: closing, pop controller`, `tweaks: modal destroyed` | ☐ | ☐ |
| EXT-14 | Mở Tinh chỉnh, bấm "Ngủ". Chờ 1 phút, bấm nút nguồn để đánh thức | Toast "Đang vào chế độ ngủ"; máy ngủ.Đánh thức: về đúng màn hình trước đó, không vệt | `tweaks: "sleep" applied`, `bridge: power action sleep` | ☐ | ☐ |
| EXT-15 | Mở Tinh chỉnh, bấm "Khởi động lại", **OK** | Toast "Đang khởi động lại", rồi máy khởi động. Sau khi lên: các công tắc còn đúng cài đặt cuối, Tinh chỉnh mở lại bình thường | `tweaks: "reboot" confirmed`, `tweaks: "reboot" applied`, `bridge: power action reboot` | ☐ | ☐ |
| EXT-16 | Mở Tinh chỉnh, bấm "Đảo màu và khởi động lại", OK. Sau khi lên, mở lại và chạy **một lần nữa** để trả lại | Máy khởi động với màu bị đảo ngược; lần chạy thứ hai trả lại bình thường. Không kẹt ở màu đảo | `tweaks: "invert-reboot" confirmed`, `tweaks: "invert-reboot" applied` ×2 | ☐ | ☐ |

Nếu kẹt ở màu đảo (EXT-16 lần đầu): cắm USB, đặt `InvertScreen=false` trong `Kobo eReader.conf` (sau khi sao lưu), khởi động lại.

## 2. Nút ẩn, tắt cưỡng bức và gỡ

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| EXT-17 | Tạo tệp rỗng `.adds/helpers/extras/screenshot-off` qua USB, rút, mở Trang chủ | Không có nút chụp; xóa tệp thì nút hiện lại ở lần hiện thanh sau | `button removed` | ☐ | ☐ |
| EXT-18 | Đổi tên `bin/extras-tweaks` thành `extras-tweaks.off`, mở Tinh chỉnh | Nickel không sập; cửa sổ báo "Thiếu một phần của Extras. Hãy cài lại." thay cho lưới; ✕ đóng được. Trả tên lại | `tweaks: list unavailable: …` | ☐ | ☐ |
| EXT-19 | Tạo `disable` qua USB, khởi động lại | Không nút chụp, không hàng Tinh chỉnh, không toast; Nickel bình thường. Xóa `disable` sau đó | (syslog) `kill switch … present, staying inert` | ☐ | ☐ |
| EXT-20 | Tạo `debug`, khởi động lại, mở Trang chủ rồi Tinh chỉnh | `.adds/helpers/extras/debug.log` có dòng hình học của `lightIcon` và nút (để xác nhận căn lề), rồi xóa `debug`. Chép log vào `backups/<ngày>/` | `lightIcon …`, `button …` | ☐ | ☐ |
| EXT-21 | `install.sh --uninstall` (hoặc tạo `uninstall`), khởi động lại | `libextras.so` biến mất; Nickel bình thường; thư mục `.adds/helpers/extras` còn (xóa tay nếu muốn) | (syslog) NickelHook uninstall | ☐ | ☐ |

## 3. Cùng cài NickelDrawer (chỉ khi muốn kiểm cùng tồn tại)

Mỗi mod cần một lần khởi động riêng (chỉ một `KoboRoot.tgz` chờ mỗi lần).

| ID | Bước | Kết quả mong đợi | Log | Đạt | Lỗi |
|---|---|---|---|---|---|
| EXT-22 | Cài Extras, khởi động lại, cài NickelDrawer, khởi động lại. Mở Thêm | Hai hàng riêng: "Ứng dụng" và "Tinh chỉnh", mỗi hàng mở đúng chức năng của nó; nút chụp vẫn có | `more row: … added` (cả hai) | ☐ | ☐ |
| EXT-23 | Gỡ NickelDrawer (`tools/install-to-kobo.sh --uninstall`), khởi động lại | Extras vẫn đủ: nút chụp, hàng Tinh chỉnh | | ☐ | ☐ |
| EXT-24 | Cài lại NickelDrawer; rồi gỡ Extras, khởi động lại | NickelDrawer vẫn đủ: hàng Ứng dụng; không còn nút chụp và hàng Tinh chỉnh | | ☐ | ☐ |
