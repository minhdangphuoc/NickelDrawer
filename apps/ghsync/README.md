# GitHub sync (ghsync)

## Tiếng Việt

Tải sách mới từ một repo GitHub riêng về máy. Chỉ thêm, không bao giờ xóa hay ghi đè. Mỗi file phải đúng loại cho phép (cùng quy tắc với LocalSend: đuôi file, dung lượng tối đa, nội dung thật sự là sách) rồi được xếp theo loại: sách vào `Books/GitHub`, PDF vào `Documents/GitHub`, truyện tranh vào `Comics/GitHub`. Cấu hình cũ có `dest` riêng (ví dụ `/mnt/onboard/GitHub`) vẫn chạy như cũ: mọi file vào đó.

**Chuẩn bị (một lần, trên điện thoại hoặc máy tính):**

1. Một repo riêng có ít nhất một cuốn sách trong thư mục bạn muốn đồng bộ (ví dụ `epub/`).
2. Tạo token: GitHub > Settings > Developer settings > Personal access tokens > **Fine-grained tokens** > Generate. Repository access: **Only select repositories**, chọn đúng repo sách. Permissions: **Contents: Read-only**.

**Trên máy:** Ngăn Ứng Dụng > **GitHub** > **Ghép nối bằng điện thoại** > **Bắt đầu ghép nối**. Máy hiện mã QR, địa chỉ `http://<ip>:8765` và mã 6 số (có hiệu lực 10 phút). Trên điện thoại cùng Wi-Fi, quét mã hoặc mở địa chỉ, nhập mã, repo (`owner/name`), thư mục, và dán token ở trang đó (không bao giờ gõ token trên Kobo hay gửi cho ai). Máy kiểm tra với GitHub rồi mới lưu. Báo lỗi rõ: không thấy repo (hoặc token không có quyền), sai nhánh, hoặc không có thư mục. Repo còn trống vẫn ghép nối được; lần đồng bộ đầu sẽ báo 0 sách. Sau đó bấm **Đồng bộ ngay**: việc tải chạy nền, màn hình hiện "Đang tải 2 / 7" và tên file; xong thì thư viện tự quét lại nếu có sách mới.

Chỉ dùng trên Wi-Fi nhà bạn: trang ghép nối là HTTP, không mã hóa. Nhập sai mã 5 lần là dừng. Chi tiết và kiểm tra an toàn token: `docs/spec/ghsync-pairing.md`.

Cách thủ công (cắm USB): dán token vào `.adds/apps/ghsync/data/token`, chép `data/config.example.json` thành `data/config.json` rồi sửa `repo`, `path` (thư mục sách, `""` = cả repo), `branch` (bỏ trống = nhánh mặc định).

Chi tiết lỗi: `data/log.txt`. Sách đã tải rồi bị bạn xóa trên máy sẽ không bị tải lại; muốn tải lại, chạy `bin/ghsync run --redownload-deleted`. File tải dở (máy tắt giữa chừng) bị dọn ở lần sau và tải lại từ đầu, không bao giờ để file hỏng. Nếu GitHub báo chưa có gì mới, lần đồng bộ bỏ qua việc tải (không tốn lượt gọi API). Không bao giờ chia sẻ file `token`.

`bin/ghsync` do `make dist` chép từ `build/`, không nằm trong git.

## English

Downloads new books from a private GitHub repo. Add-only: never deletes or overwrites. Every file must pass the same allowlist as LocalSend (extension, size cap, content that really is that type) and is sorted by type: books into `Books/GitHub`, PDFs into `Documents/GitHub`, comics into `Comics/GitHub`. An older config with its own `dest`, such as `/mnt/onboard/GitHub`, keeps working: everything goes there.

**Prepare once (on a phone or computer):**

1. A private repo with at least one book in the folder you want to sync (for example `epub/`).
2. Create a **fine-grained personal access token**: Repository access = only that one repo; Permissions = Contents: Read-only.

**On the Kobo:** App drawer > **GitHub** > **Pair with your phone** > **Start pairing**. The Kobo shows a QR code, the address `http://<ip>:8765` and a one-time 6-digit code, valid for 10 minutes. On a phone on the same Wi-Fi scan the code or open the address, enter the code, the repo (`owner/name`), the folder, and paste the token on that page (never type it on the Kobo or send it to anyone). The Kobo tests everything against GitHub before saving, and says which part is wrong: repository not found (or the token has no access), branch missing, or folder missing. An empty repository still pairs; the first sync reports 0 books. Then tap **Sync now**: the download runs in the background, the screen shows "Downloading 2 / 7" and the file name, and the library rescans itself when books were added.

Use pairing only on your own Wi-Fi: the page is plain HTTP and unencrypted. Five wrong codes close it. Details and the token-safety audit: `docs/spec/ghsync-pairing.md`.

By hand (over USB): put the token in `.adds/apps/ghsync/data/token`, copy `data/config.example.json` to `data/config.json` and edit `repo`, `path` (empty = whole repo), `branch` (empty = default branch).

Errors are in `data/log.txt`. A book you deleted on the device is not downloaded again; use `bin/ghsync run --redownload-deleted` to override. A download cut short (power loss) is cleaned up on the next run and fetched again from the start, so a broken file never appears. When GitHub says nothing changed, the sync skips the downloads (and costs no API quota). Never share the `token` file.

`bin/ghsync` is copied from `build/` by `make dist` and is not committed.
