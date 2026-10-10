# Wallpaper

## Tiếng Việt

Dùng ảnh của bạn làm màn hình chờ (ảnh hiện khi máy ngủ, bấm nút nguồn để xem).

1. Cắm USB, chép ảnh PNG hoặc JPEG (tối đa 20 MB mỗi ảnh) thẳng vào thư mục `Pictures`. Thư mục này được tạo khi bạn bấm **Hình nền · Nhập ảnh** lần đầu nếu chưa có.
2. Lần đầu mở ứng dụng với thư viện trống, ảnh đã có trong `Pictures` được nhập tự động (chỉ một lần); sau đó bấm **Hình nền · Nhập ảnh** để nhập ảnh mới. Ảnh được đổi sang kích thước màn hình, 16 mức xám, rồi đặt vào `.kobo/screensaver`. Việc này chạy nền; màn hình hiển thị tiến độ.
3. Màn hình chính là thư viện ảnh: chạm một ảnh để xem, nhấn giữ để đặt làm hình nền, xem thông tin hoặc xoá khỏi Hình nền (ảnh gốc trong `Pictures` vẫn được giữ). Biểu tượng bánh răng mở **Tùy chọn**: **Lấp đầy / Vừa khung** (phủ kín màn hình hoặc giữ cả ảnh trên nền trắng) và **Cố định / Ngẫu nhiên**.

Điều kiện để ảnh hiện: bật trong Cài đặt → Tiết kiệm pin và riêng tư cả «Hiển thị sách đang đọc» và «Hiển thị bìa sách toàn màn hình», và máy có một cuốn sách đã mở gần đây. Ứng dụng sẽ nhắc nếu hai mục này đang tắt.

Ứng dụng chỉ đọc ảnh trực tiếp trong `Pictures`, không bao giờ sửa hay xóa ảnh gốc, và chỉ ghi các file `nd-wallpaper-*.png` trong `.kobo/screensaver`; file khác ở đó giữ nguyên. Nếu bạn dùng mục NickelMenu "Screensaver" (đổi tên `.kobo/screensaver`), hãy để nó ở trạng thái BẬT thì ảnh mới hiện. Chi tiết: `docs/spec/wallpaper.md`.

## English

Use your own pictures as the Kobo's sleep screen (shown when the device sleeps; press power to see it).

1. Over USB, copy PNG or JPEG files (up to 20 MB each) into `Pictures`. The folder is created the first time you tap **Wallpaper · Import pictures** if it is missing.
2. The first time the app opens with an empty gallery, pictures already in `Pictures` are imported automatically (once only); afterwards tap **Wallpaper · Import pictures** for new ones. Pictures are scaled to the screen, reduced to 16 gray levels with dithering and placed in `.kobo/screensaver`. It runs in the background; the screen shows the progress.
3. The main screen is a picture gallery: tap a picture to view it, long-press to set it as the wallpaper, see its info or remove it from Wallpaper (the original in `Pictures` is kept). The gear opens **Options**: **Fill / Fit** (cover the screen or keep the whole picture on white) and **Fixed / Random**.

Requirement: in Settings → Energy saving and privacy turn on both “Show current read” and “Show book cover full screen”, and have a recently opened book. The app shows a hint while either setting is off.

The app only reads files directly inside `Pictures`, never changes or deletes your originals, and only writes `nd-wallpaper-*.png` in `.kobo/screensaver`; other files there are left alone. If you use the NickelMenu "Screensaver" toggle (it renames `.kobo/screensaver`), keep it ON or the pictures will not show. Details: `docs/spec/wallpaper.md`.

`bin/wallpaper` is copied from `build/` by `make dist` and is not committed.
