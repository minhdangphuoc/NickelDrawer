# Phát hành ứng dụng có cập nhật

> **Trạng thái: dự kiến cho bản 1.1 — chưa được triển khai.** Các lệnh và màn hình dưới đây mô tả thiết kế đã duyệt. Lệnh `appsctl app ...` chưa có trong bản hiện tại. Định dạng nằm trong [app-updates.md](../spec/app-updates.md) (tiếng Anh); nếu hai nơi khác nhau thì bản đặc tả là chuẩn.

Cách đưa một ứng dụng đã chạy tốt đến bản phát hành có chữ ký trên GitHub, rồi tiếp tục gửi bản cập nhật để người dùng Kobo nhận ngay trong Ngăn Ứng Dụng. Hãy làm ứng dụng trước theo [hướng dẫn cho nhà phát triển](developer-guide.md).

Bạn cần: thư mục ứng dụng đã xong, tài khoản GitHub (hoặc một máy chủ web tĩnh bất kỳ), và một máy tính có `appsctl`.

## Cách hoạt động trong một phút

- `app.json` cho biết bản cập nhật lấy từ đâu và khóa công khai nào ký chúng.
- Một bản phát hành gồm ba tệp: tệp zip của ứng dụng, `update.json` (phiên bản, mã băm, kích thước) và `update.json.sig` (chữ ký của bạn).
- Trên Kobo, **Cài đặt › Cập nhật › Kiểm tra** tải các tệp này, kiểm tra chữ ký và mã băm rồi thay thư mục ứng dụng. Thư mục `data/` của người dùng không bao giờ bị đụng đến.
- Thiết bị nhớ khóa của bạn từ lần cài đầu. Bản cập nhật không thể tự đổi khóa.

## 1. Tạo khóa

```sh
appsctl app keygen weather               # khóa ký
appsctl app keygen weather --recovery    # khóa khôi phục, cất ngoài máy
```

Mỗi lệnh ghi một khóa riêng vào `~/.config/nickeldrawer/apps/` (quyền 600), hoặc vào thư mục bạn chỉ định bằng `--key-dir DIR` (USB, thư mục xuất từ trình quản lý mật khẩu, ổ đĩa mã hóa; `sign` và `rotate` có cùng tùy chọn và tìm ở đó trước). Cất ở đâu là do bạn chọn và chịu trách nhiệm. Giới hạn duy nhất là `keygen` từ chối thư mục nằm trong cây làm việc git hoặc trong thư mục ứng dụng. Mỗi lệnh và in dòng để dán vào `app.json`. Cả hai cho phép đặt mật khẩu tùy chọn: nên đặt. Hãy sao lưu cả hai khóa riêng. Cất khóa khôi phục ngoài máy, ví dụ trên một USB để trong ngăn kéo, đừng để trên máy bạn dùng để phát hành. Mất khóa riêng thì không ai lấy lại được cho bạn.

Đừng bao giờ dùng lại khóa phát hành của chính Ngăn cho một ứng dụng.

`keygen` từ chối ghi khóa trong cây làm việc git hoặc trong thư mục ứng dụng, và gợi ý dòng cho `.gitignore` (`*.key`, `*.pem`, `id_ed25519*`). `pack` từ chối đóng gói, còn `verify` từ chối zip, nếu có tệp trông như khóa riêng (theo tên hoặc theo nội dung).

**Vì sao bản phát hành không làm lộ khóa của bạn.** Zip chỉ chứa khóa công khai, và không thể suy ra khóa riêng Ed25519 từ khóa công khai hay từ chữ ký. Rủi ro thật là sơ suất của chính bạn: để khóa trong thư mục ứng dụng, commit vào git, in ra nhật ký CI hoặc dán vào đâu đó. Nếu chuyện đó xảy ra, coi như khóa ký đã lộ: xóa nó khỏi mọi nơi, rồi đổi khóa bằng khóa khôi phục ([Đổi khóa](#đổi-khóa)) và phát hành ngay bản mới. Không có khóa khôi phục thì người dùng phải cài lại qua USB.

## 2. Khai báo nguồn trong app.json

```json
"update": {
  "github": "alice/kobo-weather",
  "key": "ed25519:kP3b...=",
  "recovery_key": "ed25519:Q9xT...="
}
```

Nếu dùng máy chủ khác, thay `github` bằng `"url": "https://example.org/weather/"` (xem [Lưu trữ ở nơi khác](#lưu-trữ-ở-nơi-khác)). Khóa có dạng `ed25519:` cộng base64 của 32 byte khóa công khai thô, đúng như `keygen` in ra.

Hãy ghi dấu vân tay khóa vào README, ví dụ `Dấu vân tay khóa ký: 3fa9-02bc-77de-1a40`. Hộp thoại Thông tin của ứng dụng hiện cùng dấu vân tay này, để người dùng đối chiếu sau khi tự cài bằng tay. `keygen` cũng in nó.

Chạy `appsctl validate .` sau khi sửa.

## 3. Bản phát hành đầu tiên

1. Đặt `version` trong `app.json` (ví dụ `1.0.0`) và chạy các bước kiểm tra trong [Kiểm thử](developer-guide.md#testing).
2. Đóng gói:

   ```sh
   appsctl app pack ./weather --out dist
   ```

   Lệnh này kiểm tra ứng dụng rồi ghi `dist/app-weather-1.0.0.zip` và `dist/update.json`. Zip chứa nội dung thư mục ứng dụng dưới `weather/`, không có `data/`. Xem lại `notes` trong `update.json`: văn bản thuần, tối đa 600 ký tự.
3. Ký trên máy của bạn:

   ```sh
   appsctl app sign dist --key ~/.config/nickeldrawer/apps/weather.key
   appsctl app verify dist
   ```

   `sign` ghi `dist/update.json.sig`. `verify` kiểm lại chữ ký, mã băm, kích thước và mọi quy tắc của zip.
4. Tạo bản phát hành GitHub với thẻ `v1.0.0` và đính kèm ba tệp. Đừng đánh dấu là bản thử (prerelease): người dùng kênh ổn định nhận bản GitHub gọi là "latest".
5. Bảo người dùng chép thư mục `weather` vào `.adds/apps/` qua USB một lần. Từ đó ứng dụng tự cập nhật.

## 4. Gửi bản cập nhật

1. Sửa mã. **Tăng `version`**: thiết bị bỏ qua phiên bản không cao hơn bản đã cài.
2. Chạy `appsctl app pack`, `sign`, `verify` như trên.
3. Tạo bản phát hành GitHub mới `v1.1.0` với ba tệp.
4. Người dùng bấm **Kiểm tra** trong Cài đặt › Cập nhật, rồi **Cập nhật tất cả** hoặc dòng của ứng dụng.

Quy tắc giữ người dùng an toàn:

- Đừng xóa tệp của một phiên bản mà bạn vẫn muốn người dùng cài.
- Giữ định dạng `data/` tương thích, hoặc chuyển đổi khi khởi động ([Phiên bản](developer-guide.md#versioning-and-min_host)).
- Đặt `min_host` khi cần Ngăn mới hơn; `pack` chép nó sang `requires.drawer`.

## Dùng GitHub Actions

Chép `.github/workflows/release.yml` từ mẫu ứng dụng (`appsctl new` tạo sẵn). Khi có thẻ `v*`, nó chạy `appsctl app pack` và tải `dist/` lên làm artifact của workflow. Nó **không** ký. Hãy tải artifact về, chạy `appsctl app sign` và `verify` trên máy bạn, rồi đính kèm ba tệp vào bản phát hành.

Ký ngay trên CI là có thể nhưng không khuyến khích. Nếu vẫn làm:

- Chỉ đặt khóa **ký** trong secret của một GitHub environment, không bao giờ đặt khóa khôi phục.
- Dùng environment được bảo vệ với người duyệt bắt buộc, để mỗi lần ký đều cần bạn đồng ý.
- Hiểu đánh đổi: ai đẩy được thẻ và duyệt được, hoặc xâm nhập được tài khoản GitHub của bạn, đều ký được bản cập nhật. Khóa nằm trên máy riêng của bạn không có rủi ro này. Mẫu có sẵn một job được chú thích để bạn bắt đầu.

## Kênh beta

Đánh dấu bản phát hành là **prerelease** trên GitHub. Người dùng chọn kênh beta trong cài đặt cập nhật sẽ nhận prerelease mới nhất; những người khác vẫn nhận bản ổn định. Kiểm tra beta dùng GitHub API, giới hạn 60 yêu cầu mỗi giờ cho mỗi địa chỉ IP khi chưa đăng nhập, nên mạng đông người có thể thấy "GitHub đang giới hạn yêu cầu". Kênh ổn định không có giới hạn này. Muốn đưa bản beta thành chính thức, phát hành lại cùng các tệp dưới dạng bản thường với phiên bản cao hơn.

## Lưu trữ ở nơi khác

Dùng `"url"` thay cho `"github"`:

```json
"update": { "url": "https://example.org/weather/", "key": "ed25519:kP3b...=" }
```

Đặt `update.json`, `update.json.sig` và tệp zip trong thư mục đó. Bắt buộc dùng HTTPS (ngoại lệ duy nhất là máy trong mạng LAN, xem [Ứng dụng riêng tư](#ứng-dụng-riêng-tư)). Máy chủ tĩnh nào cũng được: GitHub Pages, Codeberg hoặc Gitea, máy chủ riêng, NAS. Tải zip lên trước, `update.json` sau cùng. Nguồn `url` không có kênh.

## Ứng dụng riêng tư

Kho mã nguồn của bạn có thể để riêng tư. Chỉ ba tệp phát hành cần truy cập được. Dùng nguồn `url` và đưa các tệp lên một máy chủ công khai:

1. Chọn nơi lưu: một kho công khai nhỏ riêng (dùng phần releases hoặc GitHub Pages), R2, S3, Netlify hoặc máy chủ của bạn.
2. Trong kho riêng tư, thêm một job GitHub Actions chạy sau `pack` và `sign`, tải `update.json`, `update.json.sig` và zip lên nơi đó. Tải zip trước, `update.json` sau cùng. Lưu token triển khai của nơi lưu làm secret của environment, và đừng để khóa ký trên CI (xem phần trên).
3. Đặt `"url"` trong `app.json` trỏ đến thư mục công khai.

Với ứng dụng cá nhân, chạy một máy chủ web nhỏ trên NAS ở nhà và dùng `"url": "http://my-nas.local/weather/"`. `http://` thuần chỉ được chấp nhận cho nguồn `url` khi máy chủ là địa chỉ IP mạng riêng hoặc tên `.local`. Mọi trường hợp khác phải dùng HTTPS. (Mirror luôn được dùng HTTP, vì mọi thứ đều có chữ ký.)

Không hỗ trợ việc để thiết bị đọc trực tiếp kho GitHub riêng tư. Thiết bị sẽ cần một token; token đó nằm trên máy mà mọi ứng dụng chạy với quyền root, sẽ hết hạn, và cho phép xem mã nguồn của bạn.

## Máy chủ gương (mirror)

Mirror giúp ích khi GitHub chậm hoặc bị chặn. Chủ máy Kobo thêm chúng vào `.adds/koboapps/config/update.json`:

```json
{ "mirrors": ["https://ghproxy.example/", "http://my-nas.local/kobo-mirror/"] }
```

Thiết bị chỉ lấy tệp zip từ mirror (thử lần lượt, rồi mới đến địa chỉ gốc); `update.json` luôn lấy từ nguồn của bạn. URL mirror là tiền tố cộng URL gốc bỏ `https://`. Mirror không làm giả được gì vì thiết bị đối chiếu zip với mã băm đã ký, nhưng mirror biết thiết bị có những ứng dụng nào và có thể giữ lại tệp zip. Quy tắc đầy đủ và script cron cho mirror tại nhà: [app-updates.md](../spec/app-updates.md#mirrors).

Bạn không đặt mirror trong `app.json`.

## Đổi khóa

Đổi khóa cần **khóa khôi phục**. Nó sinh ra chính cho trường hợp lộ khóa ký.

1. Tạo khóa ký mới: `appsctl app keygen weather --new` (tệp cũ được giữ thành `weather.key.old`).
2. Ký lệnh đổi khóa bằng khóa khôi phục: `appsctl app rotate weather --recovery-key <tệp> --sequence 1`. Lệnh thêm `key_rotation` vào `update.json` kế tiếp. Mỗi lần dùng `sequence` cao hơn.
3. Đặt khóa công khai mới vào `update.key` trong `app.json`, rồi đóng gói và ký bản phát hành bằng khóa **mới**.
4. Giữ `key_rotation` trong các bản sau nữa. `rotate` cũng ghi `current-key.json` và `current-key.json.sig`: tải chúng lên cạnh `update.json`, và tải lại sau mỗi lần đổi khóa về sau. Thiết bị lấy chúng từ nguồn của bạn ở mỗi lần kiểm tra, nên một lệnh đổi khóa cũ sang khóa đã bị lộ sẽ bị từ chối.

Sau khi đổi, khóa ký cũ bị từ chối vĩnh viễn. Nếu bạn chưa từng tạo khóa khôi phục thì không thể đổi: hãy báo người dùng cài lại ứng dụng qua USB với khóa mới.

## An ninh

- Không để khóa riêng trong thư mục ứng dụng hay trên GitHub, hay máy dùng chung. Đặt mật khẩu cho khóa.
- Tạo khóa khôi phục ngay từ đầu và cất ngoài máy.
- Công bố dấu vân tay khóa trong README.
- Không dùng lại khóa phát hành của Ngăn.
- Tăng `version` ở mỗi bản phát hành. Thiết bị từ chối phiên bản bằng hoặc thấp hơn.
- Nhớ giới hạn: việc ghim khóa chống được máy chủ, mirror hoặc mạng xấu. Nó không chống được ứng dụng độc hại đã nằm trên máy. Ứng dụng chạy với quyền root.

## Xử lý sự cố

| Triệu chứng | Nguyên nhân | Cách sửa |
|---|---|---|
| "Không có bản cập nhật" sau khi phát hành | phiên bản không cao hơn bản đã cài, hoặc bản phát hành là nháp/prerelease trên kênh ổn định | tăng `version`; phát hành như bản thường |
| Chữ ký không hợp lệ | ký `update.json` cũ, sửa sau khi ký, hoặc sai khóa | chạy lại `appsctl app sign`, rồi `verify` |
| Khóa đã đổi | `update.key` trong `app.json` mới khác khóa đã ghim | khôi phục khóa cũ, hoặc đổi khóa đúng cách |
| Sai mã băm hoặc kích thước | zip bị đóng lại hoặc tải lên lại sau `pack` | chạy lại `pack`, `sign` rồi tải cả ba tệp lên |
| Zip không an toàn | có `data/`, liên kết tượng trưng, `..` hoặc đường dẫn ngoài `<id>/` | xóa chúng; `appsctl app verify` nêu tên mục lỗi |
| Cần Ngăn mới hơn | `requires.drawer` cao hơn bản người dùng có | hạ `min_host`, hoặc nhờ người dùng cập nhật Ngăn |
| GitHub đang giới hạn yêu cầu | kênh beta, nhiều lần kiểm tra từ một IP | đợi một giờ, hoặc dùng kênh ổn định |
| Không kết nối được | máy chủ lỗi, hoặc Wi-Fi không ra Internet | thử lại; thêm mirror |
| Đã hoàn tác | bản mới không qua kiểm tra trên thiết bị | chạy `appsctl validate` trên nội dung zip và sửa |

## Danh sách kiểm tra trước khi phát hành

- [ ] `version` cao hơn bản phát hành trước.
- [ ] `appsctl validate` không có lỗi hay cảnh báo.
- [ ] Không có `data/` và tệp cá nhân trong thư mục ứng dụng.
- [ ] `update.key` và `update.recovery_key` khớp với tệp khóa của bạn.
- [ ] `notes` ngắn, văn bản thuần.
- [ ] `pack`, `sign`, `verify` đều qua trên các tệp cuối cùng.
- [ ] Bản phát hành có đúng ba tệp: `app-<id>-<version>.zip`, `update.json`, `update.json.sig`.
- [ ] Bản phát hành không phải prerelease (ổn định) hoặc là prerelease (beta).
- [ ] README ghi dấu vân tay khóa và những gì ứng dụng truy cập.
- [ ] Khóa khôi phục nằm ngoài máy và đã sao lưu.
