# GitHub sync (ghsync)

## Tiếng Việt

Tải sách mới từ một repo GitHub riêng về `/mnt/onboard/GitHub`. Chỉ thêm, không bao giờ xóa hay ghi đè.

1. Tạo token: GitHub > Settings > Developer settings > Personal access tokens > **Fine-grained tokens** > Generate.
   - Repository access: **Only select repositories**, chọn đúng một repo sách.
   - Permissions: **Contents: Read-only** (Metadata read-only tự có).
2. Cắm USB. Dán token (một dòng) vào `.adds/apps/ghsync/data/token`.
3. Chép `data/config.example.json` thành `data/config.json`, sửa `repo`, `path` (thư mục sách, `""` = cả repo), `branch` (bỏ trống = nhánh mặc định).
4. Trên máy: **GitHub · Đồng bộ ngay**. Việc tải chạy nền vì menu chỉ chờ dưới 10 giây.
5. Sau vài phút: **GitHub · Trạng thái**. Bấm này cũng quét lại thư viện để sách mới hiện ra.

Chi tiết lỗi: `data/log.txt`. Sách đã tải rồi bị bạn xóa trên máy sẽ không bị tải lại (hiện "đã xoá trên máy"); muốn tải lại, chạy `bin/ghsync run --redownload-deleted`. Khi đang chạy, kết quả nằm ở `/tmp`, chép sang `data/last-run.txt` lúc xong. Không bao giờ chia sẻ file `token`.

`bin/ghsync` do `make dist` chép từ `build/`, không nằm trong git.

## English

Downloads new books from a private GitHub repo into `/mnt/onboard/GitHub`. Add-only: never deletes or overwrites. A book you deleted on the device is not downloaded again (counted as "deleted on the device"); use `bin/ghsync run --redownload-deleted` to override.

1. Create a **fine-grained personal access token**: Repository access = only the one repo; Permissions = Contents: Read-only (Metadata read-only is added automatically).
2. Over USB, paste the token into `.adds/apps/ghsync/data/token`.
3. Copy `data/config.example.json` to `data/config.json` and edit `repo`, `path`, `branch` (empty = default branch).
4. Tap **GitHub · Sync now**. The download runs in the background (NickelMenu limits commands to under 10 s). Then tap **GitHub · Status** a few minutes later; that action also rescans the library.

`bin/ghsync` is copied from `build/` by `make dist` and is not committed.
