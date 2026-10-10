# UI Kit

## Tiếng Việt

Màn hình mẫu cho mọi thành phần của NickelDrawer (`ui.json`, kit 1), dành cho nhà phát triển và người thử. Ứng dụng này **ẩn mặc định** (`"hidden": true` trong `app.json`) nên không hiện trong Ngăn.

**Cách hiện nó:** cắm USB, sao lưu `/.adds/apps/uikit/app.json` vào `backups/<ngày>/`, rồi đổi `"hidden": true` thành `"hidden": false`. Eject, mở Ngăn, bấm **Làm mới danh sách** (phần Hệ thống). Muốn ẩn lại, đổi về `true` (cài lại bằng `tools/install-to-kobo.sh` cũng trả về `true`). Ứng dụng ẩn vẫn chạy được theo mã: `appsctl drawer screen uikit`.

Ba màn hình:

- **Bộ giao diện** (chính): văn bản, khoá-giá trị, mã lớn, ghi chú (và ghi chú tự ẩn khi trống), các bước, dòng chạm được, dòng sang màn khác, danh sách có menu nhấn giữ, nút chính. Biểu tượng làm mới quét lại ảnh.
- **Thêm thành phần**: công tắc, lựa chọn (radio) có dòng phụ, ô nhập chữ và ô nhập số, các nút (phụ, tắt/bật nhờ biến, có xác nhận), mã QR, khối ghép nối, và liên kết trên tiêu đề.
- **Thư viện ảnh**: lưới ảnh lấy từ các tệp PNG nhỏ (tối đa 1 MB, tối đa 6 ảnh) trong `Pictures`. Không có ảnh thì hiện màn trống. Ứng dụng chỉ **chép** ảnh vào `data/samples/`, không đổi hay xoá ảnh gốc.

Mọi thao tác chỉ phản hồi bằng toast đã khai báo để bạn kiểm tra cú chạm. Giá trị của công tắc, lựa chọn và ô nhập được lưu trong `data/` để màn hình hiện lại đúng giá trị. Các ca thử KIT-* trong `docs/testing/final-checklist.md` (mục 5b, UIK-*) dùng ứng dụng này.

## English

Sample screens for every NickelDrawer component (`ui.json`, kit 1), for developers and testers. The app is **hidden by default** (`"hidden": true` in `app.json`), so it does not show in the drawer.

**To show it:** plug in over USB, copy `/.adds/apps/uikit/app.json` to `backups/<date>/` first, then change `"hidden": true` to `"hidden": false`. Eject, open the drawer and tap **Refresh list** (System section). To hide it again set it back to `true` (reinstalling with `tools/install-to-kobo.sh` also resets it). A hidden app still runs by id: `appsctl drawer screen uikit`.

Three screens:

- **UI Kit** (main): text styles, key-value rows, big code, notices (and a notice that hides itself when empty), steps, a tappable row, a row that pushes a screen, a list with a long-press menu, and the primary button. The refresh icon rescans pictures.
- **More components**: a switch, radio options with sub-lines, a text field and a number field, buttons (secondary, enabled or disabled through a `$var`, with a confirm), a QR code, the pairing block and a header link.
- **Gallery**: a picture grid built from small PNG files (up to 1 MB each, at most 6) in `Pictures`. With none, the empty state shows. The app only **copies** them into `data/samples/`; originals are never changed or deleted.

Every action only echoes a declared toast so a tap can be verified. The switch, radio and field values are stored in `data/` so the screen shows them again. The KIT-* cases in `docs/testing/final-checklist.md` (section 5b, UIK-*) use this app.

## For developers

Copy a component's JSON from `ui.json` when you need an example. Because an app has at most 12 actions, plain taps (header link, buttons, rows, tiles) share the `tap` action; controls store their value through `bin/lib.sh` (`remember_value`). Preview without a device:

```sh
go run ./cmd/ndpreview render apps/uikit --locale vi --state apps/uikit/preview-state.json --out build/preview/uikit-main.png
go run ./cmd/ndpreview render apps/uikit --locale vi --state apps/uikit/preview-state.json --screen more --out build/preview/uikit-more.png
go run ./cmd/ndpreview render apps/uikit --locale vi --state apps/uikit/preview-state.json --screen pictures --out build/preview/uikit-pictures.png
```

The preview state's pictures come from `go run ./tools/wallpaper-samples --out build/preview/wallpaper-samples`.
