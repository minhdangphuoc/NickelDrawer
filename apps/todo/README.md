# To-do

## Tiếng Việt

Danh sách việc cần làm nằm gọn trong máy Kobo, không cần mạng.

1. Mở **Việc cần làm** trong ngăn ứng dụng. Bấm ô **Việc mới**, gõ bằng bàn phím của Nickel (gõ được tiếng Việt có dấu) rồi xác nhận: việc được thêm vào cuối danh sách. Chữ được đưa về chuẩn Unicode NFC, nên việc có dấu luôn khớp nhau.
2. Chạm một việc để đánh dấu xong (có dấu ✓), chạm lần nữa để mở lại. Nhấn giữ một việc rồi chọn **Xóa...** để xóa nó (có hỏi trước).
3. **Xóa việc đã xong** xóa mọi việc đã đánh dấu (có hỏi trước); nút mờ khi chưa có việc nào xong. Dòng bên phải tiêu đề cho biết số việc còn lại và số việc đã xong.
4. Danh sách có tối đa 200 việc, mỗi việc tối đa 200 ký tự, và không có hai việc trùng chữ.

Danh sách nằm trong `data/todo.json` của ứng dụng, ghi bằng tệp tạm rồi đổi tên, nên mất điện cũng không hỏng và vẫn còn sau khi khởi động lại. Nếu tệp bị hỏng (không đọc được), nó được giữ lại thành `data/todo.json.corrupt`, danh sách bắt đầu lại từ trống và màn hình có ghi chú báo việc đó. Gỡ ứng dụng không xóa `data/`.

### Cài đặt, cập nhật, gỡ

- **Cài:** tải `app-todo-<phiên bản>.zip` từ [bản phát hành](../../../../releases), giải nén và gộp thư mục `.adds` trong đó vào thư mục gốc của ổ Kobo (Finder: chọn **Gộp (Merge)**, đừng chọn Thay thế). Hoặc từ bản checkout: `make dist` rồi `tools/install-to-kobo.sh --app todo`. Ngăn ứng dụng phải được cài trước. Các bước chi tiết: [apps/README.vi.md](../README.vi.md).
- **Cập nhật:** cài zip mới theo cách trên. Tệp của ứng dụng được thay, `data/` được giữ.
- **Gỡ:** nhấn giữ ứng dụng trong ngăn rồi chọn Gỡ (Remove), hoặc xóa thư mục `.adds/apps/todo` trên ổ.

## English

A small to-do list that stays on your Kobo, no network needed.

1. Open **To-do** in the app drawer. Tap **New task**, type with Nickel's keyboard (Vietnamese with accents works) and confirm: the task is added at the end of the list. The text is normalised to Unicode NFC, so accented tasks always compare equal.
2. Tap a task to mark it done (a ✓ shows), tap again to reopen it. Long-press a task and choose **Delete...** to remove it (it asks first).
3. **Clear done tasks** deletes every task marked done (it asks first); the button is grey while nothing is done. The status line shows how many are open and how many are done.
4. The list holds up to 200 tasks of up to 200 characters each, and no two tasks have the same text.

The list lives in the app's `data/todo.json`, written through a temp file and a rename, so a power cut never leaves it half written and it survives a reboot. If the file cannot be read it is kept as `data/todo.json.corrupt`, the list starts empty and the screen shows a notice. Removing the app does not delete `data/`.

### Install, update, remove

- **Install:** download `app-todo-<version>.zip` from the [release](../../../../releases), unzip it and merge its `.adds` folder onto the root of the Kobo drive (Finder: choose **Merge**, never Replace). Or, from a checkout, `make dist` then `tools/install-to-kobo.sh --app todo`. The drawer must already be installed. Steps and details: [apps/README.md](../README.md).
- **Update:** install the new zip the same way. The app's files are replaced and its `data/` is kept.
- **Remove:** long-press the app in the drawer and choose Remove, or delete `.adds/apps/todo` on the drive.

## For developers

`bin/todo` is a small Go program (`cmd/todo`, logic in `internal/todo`, tests next to both) that `make dist` builds for the Kobo; it is not committed. The scripts in `bin/` only map the drawer's events to it and turn its outcome word (`added`, `duplicate`, `cleared 2`, ...) into a toast declared in `ui.json`. State reads one small file, well under a second.

Run on the Mac with temporary folders:

```sh
go build -o /tmp/todo ./cmd/todo
/tmp/todo add --data-dir /tmp/tododata --text "Đọc sách"
/tmp/todo state --data-dir /tmp/tododata
go run ./cmd/ndpreview render apps/todo --locale vi --state apps/todo/preview-state.json --out build/preview/todo-main.png
```
