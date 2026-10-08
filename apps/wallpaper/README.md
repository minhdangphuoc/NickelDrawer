# Wallpaper

## Tiếng Việt

Dùng ảnh của bạn làm màn hình chờ (ảnh hiện khi máy ngủ, bấm nút nguồn để xem).

1. Cắm USB, chép ảnh PNG hoặc JPEG (tối đa 20 MB mỗi ảnh) vào `Pictures/Wallpapers` (hoặc thẳng vào `Pictures`). Thư mục này được tạo khi bạn bấm **Hình nền · Nhập ảnh** lần đầu.
2. Bấm **Hình nền · Nhập ảnh**. Ảnh được đổi sang kích thước màn hình, 16 mức xám, rồi đặt vào `.kobo/screensaver`. Việc này chạy nền; một phút sau bấm **Trạng thái** để xem kết quả.
3. **Ảnh kế tiếp** đổi ảnh. **Lấp đầy / Vừa khung**: phủ kín màn hình (cắt mép) hoặc giữ nguyên cả ảnh trên nền trắng. **Cố định / Ngẫu nhiên**: một ảnh, hoặc máy chọn ngẫu nhiên trong tất cả. **Tắt hình nền** gỡ ảnh của ứng dụng khỏi `.kobo/screensaver`.

Ứng dụng chỉ đọc ảnh trực tiếp trong `Pictures/Wallpapers` và `Pictures`, không bao giờ sửa hay xóa ảnh gốc, và chỉ ghi các file `nd-wallpaper-*.png` trong `.kobo/screensaver`; file khác ở đó giữ nguyên. Nếu bạn dùng mục NickelMenu "Screensaver" (đổi tên `.kobo/screensaver`), hãy để nó ở trạng thái BẬT thì ảnh mới hiện. Chi tiết: `docs/spec/wallpaper.md`.

## English

Use your own pictures as the Kobo's sleep screen (shown when the device sleeps; press power to see it).

1. Over USB, copy PNG or JPEG files (up to 20 MB each) into `Pictures/Wallpapers` (or directly into `Pictures`). The folder is created the first time you tap **Wallpaper · Import pictures**.
2. Tap **Wallpaper · Import pictures**. Pictures are scaled to the screen, reduced to 16 gray levels with dithering and placed in `.kobo/screensaver`. It runs in the background; tap **Status** a minute later for the result.
3. **Next picture** changes the picture. **Fill / Fit**: cover the screen (edges cropped) or keep the whole picture on white. **Fixed / Random**: one picture, or Nickel picks any. **Turn off** removes the app's pictures from `.kobo/screensaver`.

The app only reads files directly inside `Pictures/Wallpapers` and `Pictures`, never changes or deletes your originals, and only writes `nd-wallpaper-*.png` in `.kobo/screensaver`; other files there are left alone. If you use the NickelMenu "Screensaver" toggle (it renames `.kobo/screensaver`), keep it ON or the pictures will not show. Details: `docs/spec/wallpaper.md`.

`bin/wallpaper` is copied from `build/` by `make dist` and is not committed.
