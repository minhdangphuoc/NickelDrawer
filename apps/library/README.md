# Library

## Tiếng Việt

Sửa ngôn ngữ cho sách EPUB tiếng Việt và tìm sách trùng, trong `Books`, `Documents` và `Comics`.

1. Mở **Thư viện** trong ngăn ứng dụng và bấm **Quét**. Ứng dụng đọc tên sách, tác giả và một đoạn chữ của từng EPUB, rồi báo: số sách, số sách tiếng Việt bị gắn sai ngôn ngữ (thường là "en", làm máy ngắt từ kiểu tiếng Anh), số sách chưa chuẩn NFC, số nhóm sách trùng. Việc này chạy nền; mở màn hình Thư viện (ngăn ứng dụng) để xem tiến độ.
2. **Sửa ngôn ngữ** (trong màn hình Thư viện): mỗi sách được chép sang `data/backups/<ngày>/` trước, rồi chỉ đổi ngôn ngữ thành `vi` (tệp OPF và thuộc tính `lang`). Sách được kiểm tra lại trước khi thay bản cũ. **Hoàn tác lần sửa** đưa bản gốc về.
3. Sau khi sửa, thư viện tự được quét lại. Ứng dụng không bao giờ đụng đến cơ sở dữ liệu `KoboReader.sqlite`.
3a. **Sửa tên (NFC)** đưa tên tệp, tên sách và tác giả về chuẩn Unicode NFC (sao lưu trước, **Hoàn tác** trả lại cả tên cũ). **Dừng** dừng việc đang chạy sau cuốn sách hiện tại. **Khôi phục từ thùng rác** trả lại các sách trùng đã bỏ đi ở ngày gần nhất.
4. **Sách trùng**: chạm vào một sách trong danh sách để chọn, rồi bấm **Bỏ vào thùng rác**. Tệp được giữ 30 ngày trong `data/trash/` và không bao giờ bị xóa ngay; bản cuối cùng không thể bỏ đi.

Chỉ đọc `Books`, `Documents`, `Comics` (không theo liên kết tượng trưng, bỏ qua tệp ẩn, tối đa 200 MB mỗi tệp). Chi tiết: `docs/spec/library.md`.

### Cài đặt, cập nhật, gỡ

- **Cài:** tải `app-library-<phiên bản>.zip` từ [bản phát hành](../../../../releases), giải nén và gộp thư mục `.adds` trong đó vào thư mục gốc của ổ Kobo (Finder: chọn **Gộp (Merge)**, đừng chọn Thay thế). Hoặc từ bản checkout: `make dist` rồi `tools/install-to-kobo.sh --app library`. Ngăn ứng dụng phải được cài trước. Các bước chi tiết: [apps/README.vi.md](../README.vi.md).
- **Cập nhật:** cài zip mới theo cách trên. Tệp của ứng dụng được thay, `data/` được giữ.
- **Gỡ:** nhấn giữ ứng dụng trong ngăn rồi chọn Gỡ (Remove), hoặc xóa thư mục `.adds/apps/library` trên ổ.

## English

Fixes the language of Vietnamese EPUBs and finds duplicate books in `Books`, `Documents` and `Comics`.

1. Open **Library** in the app drawer and tap **Scan**. The app reads each EPUB's title, author and a text sample, then reports: books, Vietnamese books tagged with another language (usually "en", which gives English hyphenation), books not in NFC, and duplicate groups. It runs in the background; open the Library screen (app drawer) to follow it.
2. **Fix language** (on the Library screen): each book is copied to `data/backups/<date>/` first, then only its language is set to `vi` (OPF and `lang` attributes). The new file is re-read before it replaces the old one. **Undo last fix** puts the originals back.
3. After a change the library is rescanned by itself. The app never touches `KoboReader.sqlite`.
3a. **Fix names (NFC)** puts file names, titles and authors into Unicode NFC (backed up first; **Undo** brings the old names back too). **Stop** ends the running job after the current book. **Restore from trash** puts back the duplicates trashed on the latest day.
4. **Duplicates**: tap a book in the list to select it, then tap **Move to trash**. The file is kept 30 days in `data/trash/`, never deleted at once, and the last copy cannot be removed.

It only reads `Books`, `Documents` and `Comics` (no symbolic links followed, hidden files skipped, 200 MB per file at most). Details: `docs/spec/library.md`.

`bin/library` is copied from `build/` by `make dist` and is not committed.

### Install, update, remove

- **Install:** download `app-library-<version>.zip` from the [release](../../../../releases), unzip it and merge its `.adds` folder onto the root of the Kobo drive (Finder: choose **Merge**, never Replace). Or, from a checkout, `make dist` then `tools/install-to-kobo.sh --app library`. The drawer must already be installed. Steps and details: [apps/README.md](../README.md).
- **Update:** install the new zip the same way. The app's files are replaced and its `data/` is kept.
- **Remove:** long-press the app in the drawer and choose Remove, or delete `.adds/apps/library` on the drive.
