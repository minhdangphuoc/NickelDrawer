# Ứng dụng

Ứng dụng tách riêng khỏi ngăn NickelDrawer. Bản phát hành của ngăn chỉ có ngăn (công cụ chủ và mod Nickel); mỗi ứng dụng là một tệp tải riêng và được cài từng cái, nên bạn chỉ chép vào Kobo thứ mình dùng. Ứng dụng không thay đổi ngăn, và cập nhật ngăn không bao giờ đụng vào ứng dụng đã cài hay thư mục `data/` của nó.

English: [README.md](README.md).

## Danh sách ứng dụng

| Ứng dụng | Thư mục | Làm gì | Phiên bản | Ẩn |
|---|---|---|---|---|
| [GitHub](ghsync/README.md) | `apps/ghsync/` | Đồng bộ sách từ repo GitHub riêng (chỉ thêm, không xóa) | 0.3.0 | không |
| [Thư viện](library/README.md) | `apps/library/` | Sửa ngôn ngữ sách tiếng Việt và tìm sách trùng | 0.2.0 | không |
| [LocalSend](localsend/README.md) | `apps/localsend/` | Nhận sách và tệp từ điện thoại qua LocalSend | 1.2.0 | không |
| [Việc cần làm](todo/README.md) | `apps/todo/` | Danh sách việc cần làm nằm gọn trong máy Kobo | 1.0.1 | không |
| [UI Kit](uikit/README.md) | `apps/uikit/` | Every NickelDrawer component with sample content, for developers and testers | 2.1.0 | có (chỉ cho nhà phát triển) |
| [Hình nền](wallpaper/README.md) | `apps/wallpaper/` | Dùng ảnh của bạn làm màn hình chờ | 0.3.0 | không |

Ứng dụng ẩn không hiện trong ngăn cho đến khi bạn bật nó trong cài đặt của ngăn (biểu tượng bánh răng ở tiêu đề). UI Kit là bản mẫu mọi thành phần giao diện, dành cho nhà phát triển và người thử.

## Cài một ứng dụng

Cài ngăn trước (xem [README chính](../README.vi.md#cài-đặt)). Sau đó chọn một trong hai cách. Sao lưu thư mục `.adds` của Kobo trước.

**A. Từ bản phát hành.** Tải `app-<id>-<phiên bản>.zip` từ [bản phát hành](../../../releases) (ví dụ `app-todo-1.0.1.zip`), giải nén rồi gộp thư mục `.adds` bên trong vào thư mục gốc của ổ Kobo. Thư mục ẩn: Cmd+Shift+. trong Finder của macOS, "hiện tệp ẩn" trên Windows và Linux. **Finder trên macOS: nếu hỏi có Thay thế (Replace) thư mục đã có không, đừng chọn.** Giữ phím Option rồi chọn **Gộp (Merge)**. Mỗi tệp zip chỉ chứa ứng dụng của nó trong `.adds/apps/<id>/`, không có thư mục `data/`. Ngắt kết nối (eject) Kobo; ứng dụng hiện trong ngăn sau khi quét lại (nút làm mới của ngăn) hoặc ở lần khởi động sau.

**B. Từ bản checkout** (macOS hoặc Linux, đã cắm Kobo):

```sh
make check build dist                      # tạo dist/apps/app-<id>-<phiên bản>.zip cho mọi ứng dụng
tools/install-to-kobo.sh --dry-run --app todo   # cho xem việc sẽ làm
tools/install-to-kobo.sh --app todo        # cài một ứng dụng; thêm --app để cài nhiều cái
tools/install-to-kobo.sh --apps all        # cài mọi ứng dụng
```

Tập lệnh cài giải nén các tệp zip trong `dist/apps/`, sao lưu các ứng dụng nó thay vào `backups/<ngày-giờ>/` và giữ `data/` của từng ứng dụng. Không có `--app`/`--apps` thì nó chỉ cài ngăn và không đụng vào ứng dụng đang có trên Kobo. Bạn cũng có thể giải nén một tệp zip trong `dist/apps` rồi chép thủ công; thư mục `apps/<id>/` trong mã nguồn chưa có tệp chương trình đã dựng nên không cài thẳng được.

## Cập nhật một ứng dụng

Cài tệp zip mới theo cách A hoặc B. Tệp của ứng dụng được thay; `data/` (cài đặt, token, danh sách) được giữ. Bạn cũng có thể cập nhật ngay trong ngăn: **⚙** → **Cập nhật** → **Ứng dụng** (xem [docs/UPDATES.md](../docs/UPDATES.md)). Nó tải tệp zip của ứng dụng từ bản phát hành riêng (`<id>-v<phiên bản>`), kiểm danh sách có chữ ký, kích thước và sha256, rồi thay thư mục; `data/` được chuyển chỗ chứ không sao chép. Cũng màn hình đó cài các ứng dụng chưa có trên Kobo. Cập nhật ngăn (nút **Kiểm tra** trong cài đặt) chỉ cập nhật ngăn.

## Gỡ một ứng dụng

Nhấn giữ ứng dụng trong ngăn rồi chọn **Gỡ (Remove)**, hoặc xóa thư mục `.adds/apps/<id>` trên ổ Kobo (xóa luôn `data/` bên trong nếu cũng muốn bỏ cài đặt của nó). Sách của bạn không bao giờ bị đụng tới.

## Thêm ứng dụng vào kho mã này

Mỗi ứng dụng nằm trong `apps/<id>/` với một `README.md` (có phần cài đặt, cập nhật, gỡ) và một dòng trong bảng ở trên và trong [README.md](README.md). `make dist` đóng mỗi thư mục có `app.json` thành một tệp zip riêng; `appsctl validate --apps-dir apps` kiểm chúng. Xem [docs/APPS.md](../docs/APPS.md).
