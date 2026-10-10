# UI Kit

## Tiếng Việt

Bộ mẫu cho mọi thành phần của NickelDrawer (`ui.json`, kit 1), dành cho nhà phát triển và người thử. Ứng dụng này **ẩn mặc định** (`"hidden": true` trong `app.json`) nên không hiện trong Ngăn.

**Cách hiện nó:** trong Ngăn, bấm biểu tượng bánh răng ⚙ ở đầu trang, rồi bật công tắc **Bộ giao diện** trong mục **Ứng dụng ẩn**; quay lại Ngăn là thấy. Muốn ẩn lại, tắt công tắc đó. Không cần sửa `app.json`. Ứng dụng ẩn vẫn chạy được theo mã: `appsctl drawer screen uikit`.

Cấu trúc:

- **Màn chính** là một danh sách chia nhóm, 18 dòng trong 6 nhóm: Thao tác (Nút, Dòng và nút tiêu đề, Menu nhấn giữ), Lựa chọn (Công tắc, Chọn một), Nhập liệu (Ô nhập, Ghép nối điện thoại), Phản hồi (Thông báo nhanh, Hộp thoại, Ghi chú, Tiến trình, Màn trống), Điều hướng (Tiêu đề và trang), Bố cục (Văn bản, Mục và khóa–giá trị, Mã lớn và QR, Danh sách, Thư viện và xem ảnh). Mỗi nhóm nằm trọn trong một trang.
- **Mỗi thành phần một trang**: mô tả, bảng **Trạng thái** (bình thường, đang nhấn, đã chọn, khóa) là ảnh chụp tĩnh, một bản **Thử** dùng được thật, dòng **Sự kiện cuối / Giá trị** cập nhật ngay sau mỗi lần chạm, và mục **Gợi ý** (dùng khi / không dùng khi).
- **Trình xem ảnh** là màn duy nhất ở cấp 2, mở từ trang Thư viện. Hộp thoại, menu nhấn giữ, thông báo nhanh và bàn phím là lớp phủ, không phải màn hình.

Ảnh của trang Thư viện lấy từ các tệp PNG nhỏ (tối đa 1 MB, tối đa 6 ảnh) trong `Pictures`; không có ảnh thì hiện màn trống. Ứng dụng chỉ **chép** ảnh vào `data/samples/`. Giá trị của công tắc, lựa chọn và ô nhập được lưu trong `data/` để trang hiện lại đúng giá trị. Các ca thử UIK-* trong `docs/testing/final-checklist.md` (mục 5b) dùng ứng dụng này.

## English

Sample pages for every NickelDrawer component (`ui.json`, kit 1), for developers and testers. The app is **hidden by default** (`"hidden": true` in `app.json`), so it does not show in the drawer.

**To show it:** in the drawer tap the gear ⚙ in the header, then turn on **UI Kit** under **Hidden apps**; go back to the drawer and it is listed. Turn the switch off to hide it again. No `app.json` edit is needed. A hidden app still runs by id: `appsctl drawer screen uikit`.

Structure:

- **Home** is one grouped list, 18 rows in 6 groups: Actions (Button, Row and header actions, Long-press menu), Selection (Toggle, Radio group), Text input (Field, Phone pairing), Feedback (Toast, Dialog, Notice, Steps, Empty state), Navigation (Header and pages), Containment (Text, Section and key–value, Big code and QR, List, Gallery and viewer). A group never splits across pages.
- **One page per component**: a description, a **States** matrix (normal, pressed, selected, disabled) of static pictures made with the `samples` component, one working **Try it** sample, a **Last event / Value** readout that updates right after each tap, and a **Guidance** section (use for, not for).
- **The picture viewer** is the only level-2 screen, opened from the Gallery page. Dialogs, the long-press menu, toasts and the keyboard are overlays, not screens.

The Gallery page's pictures are small PNG files (up to 1 MB each, at most 6) copied from `Pictures`; with none, the empty state shows. The app only **copies** them into `data/samples/`. The switch, radio and field values are stored in `data/` so a page shows them again. The UIK-* cases in `docs/testing/final-checklist.md` (section 5b) use this app.

## For developers

The pages are generated-looking but hand-editable JSON: `ui.json` is the home, `screens/<id>.json` one page each (the viewer included); texts are `@ui.*` keys in `locales/`. An app has at most 12 actions, so the plain events (button, row, header link or icon, list tap, menu item, gallery long-press, viewer button) share `echo`, which tells the readout what happened from the environment (`NICKELDRAWER_LIST` set: menu; `NICKELDRAWER_VALUE` set: tap; neither: action). Switches, the radio group and the fields run the action named like their variable (`sampleenabled`, `autorescan`, `shownote`, `refreshmode`, `searchtext`, `pagenumber`); the Steps page runs a fake three-step job.

Preview without a device (every page, both languages, plus the drawer and Tweaks):

```sh
sh tools/preview-all.sh
go run ./cmd/ndpreview render apps/uikit --locale vi --state apps/uikit/preview-state.json --screen button --out build/preview/uikit-button.png
```

The preview state's pictures come from `go run ./tools/wallpaper-samples --out build/preview/wallpaper-samples`.
