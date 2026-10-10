# Wallpaper

## Tiếng Việt

Dùng ảnh của bạn làm màn hình chờ (ảnh hiện khi máy ngủ, bấm nút nguồn để xem).

1. Cắm USB, chép ảnh PNG hoặc JPEG (tối đa 20 MB mỗi ảnh) thẳng vào thư mục `Pictures`. Thư mục này được tạo khi bạn bấm **Hình nền · Nhập ảnh** lần đầu nếu chưa có.
2. Lần đầu mở ứng dụng với thư viện trống, ảnh đã có trong `Pictures` được nhập tự động (chỉ một lần); sau đó bấm **Hình nền · Nhập ảnh** để nhập ảnh mới. Ảnh được đổi sang kích thước màn hình, 16 mức xám, rồi đặt vào `.kobo/screensaver`. Việc này chạy nền; màn hình hiển thị tiến độ.
3. Màn hình chính là thư viện ảnh: chạm một ảnh để xem, nhấn giữ để đặt làm hình nền, xem thông tin hoặc xóa khỏi Hình nền (ảnh gốc trong `Pictures` vẫn được giữ; **Xóa cả ảnh gốc…** xóa cả ảnh gốc). Biểu tượng bánh răng mở **Tùy chọn**: **Lấp đầy / Vừa khung** (phủ kín màn hình hoặc giữ cả ảnh trên nền trắng) và **Cố định / Ngẫu nhiên**. Ở chế độ Ngẫu nhiên, mỗi lần máy thức dậy ứng dụng chọn một ảnh khác (không lặp ảnh vừa hiện và ảnh trước đó) cho lần ngủ sau; việc này cần NickelDrawer.

Điều kiện để ảnh hiện: bật trong Cài đặt → Tiết kiệm pin và riêng tư cả «Hiển thị sách đang đọc» và «Hiển thị bìa sách toàn màn hình», và máy có một cuốn sách đã mở gần đây. Ứng dụng sẽ nhắc nếu hai mục này đang tắt.

Ứng dụng chỉ đọc ảnh trực tiếp trong `Pictures`, không bao giờ sửa ảnh gốc, chỉ xóa ảnh gốc khi bạn chọn **Xóa cả ảnh gốc…** (phải giống hệt ảnh đã nhập, và ngăn sẽ hỏi trước), và chỉ ghi các file `nd-wallpaper-*.png` trong `.kobo/screensaver`; file khác ở đó giữ nguyên. Nếu bạn dùng mục NickelMenu "Screensaver" (đổi tên `.kobo/screensaver`), hãy để nó ở trạng thái BẬT thì ảnh mới hiện. Chi tiết: `docs/spec/wallpaper.md`.

### Cài đặt, cập nhật, gỡ

- **Cài:** tải `app-wallpaper-<phiên bản>.zip` từ [bản phát hành](../../../../releases), giải nén và gộp thư mục `.adds` trong đó vào thư mục gốc của ổ Kobo (Finder: chọn **Gộp (Merge)**, đừng chọn Thay thế). Hoặc từ bản checkout: `make dist` rồi `tools/install-to-kobo.sh --app wallpaper`. Ngăn ứng dụng phải được cài trước. Các bước chi tiết: [apps/README.vi.md](../README.vi.md).
- **Cập nhật:** cài zip mới theo cách trên. Tệp của ứng dụng được thay, `data/` được giữ.
- **Gỡ:** nhấn giữ ứng dụng trong ngăn rồi chọn Gỡ (Remove), hoặc xóa thư mục `.adds/apps/wallpaper` trên ổ.

## English

Use your own pictures as the Kobo's sleep screen (shown when the device sleeps; press power to see it).

1. Over USB, copy PNG or JPEG files (up to 20 MB each) into `Pictures`. The folder is created the first time you tap **Wallpaper · Import pictures** if it is missing.
2. The first time the app opens with an empty gallery, pictures already in `Pictures` are imported automatically (once only); afterwards tap **Wallpaper · Import pictures** for new ones. Pictures are scaled to the screen, reduced to 16 gray levels with dithering and placed in `.kobo/screensaver`. It runs in the background; the screen shows the progress.
3. The main screen is a picture gallery: tap a picture to view it, long-press to set it as the wallpaper, see its info or remove it from Wallpaper (removing from Wallpaper keeps the original in `Pictures`; **Remove with original…** deletes it too). The gear opens **Options**: **Fill / Fit** (cover the screen or keep the whole picture on white) and **Fixed / Random**. With Random, each time the device wakes up the app picks another picture (never the one just shown or the one before it) for the next sleep; this needs NickelDrawer.

Requirement: in Settings → Energy saving and privacy turn on both “Show current read” and “Show book cover full screen”, and have a recently opened book. The app shows a hint while either setting is off.

The app only reads files directly inside `Pictures`, never changes your originals, deletes one only when you choose **Remove with original…** (it must be identical to the imported picture, and the drawer asks first), and only writes `nd-wallpaper-*.png` in `.kobo/screensaver`; other files there are left alone. If you use the NickelMenu "Screensaver" toggle (it renames `.kobo/screensaver`), keep it ON or the pictures will not show. Details: `docs/spec/wallpaper.md`.

`bin/wallpaper` is copied from `build/` by `make dist` and is not committed.

### Install, update, remove

- **Install:** download `app-wallpaper-<version>.zip` from the [release](../../../../releases), unzip it and merge its `.adds` folder onto the root of the Kobo drive (Finder: choose **Merge**, never Replace). Or, from a checkout, `make dist` then `tools/install-to-kobo.sh --app wallpaper`. The drawer must already be installed. Steps and details: [apps/README.md](../README.md).
- **Update:** install the new zip the same way. The app's files are replaced and its `data/` is kept.
- **Remove:** long-press the app in the drawer and choose Remove, or delete `.adds/apps/wallpaper` on the drive.
