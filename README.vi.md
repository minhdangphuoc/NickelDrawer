# NickelDrawer (Tiếng Việt)

Ngăn ứng dụng cho máy đọc sách Kobo chạy phần mềm gốc (Nickel). NickelDrawer thêm mục **Ứng dụng** và bảng **Tinh chỉnh** vào tab Thêm, cùng một số công cụ nhỏ: nhận tệp từ điện thoại (LocalSend), đồng bộ sách từ GitHub, dùng ảnh của bạn làm màn hình chờ, và dọn thư viện. Không cần tài khoản đám mây và không gửi dữ liệu đi đâu nếu bạn không bật tính năng cần mạng.

**Trạng thái: 1.0.0-beta.1.** Mới thử trên một máy và một bản firmware. Bản tiếng Anh đầy đủ ở [README.md](README.md).

## Firmware hỗ trợ

Kobo Libra 2, năm bản 4.38 mới nhất: 4.38.23828, 4.38.23697, 4.38.23684, 4.38.23648, 4.38.23552. Firmware 5.x **chưa** được hỗ trợ. Ngăn ứng dụng tự tắt và ghi log nếu gặp bản firmware lạ.

## Cài đặt

1. **Sao lưu trước.** Chép toàn bộ thư mục `.kobo` và `.adds` của Kobo vào máy tính.
2. Giải nén `koboapps-<phiên bản>.zip`, chép thư mục `.adds` trong `onboard/` vào thư mục gốc của Kobo (gộp với `.adds` có sẵn).
3. Chép `KoboRoot-nickeldrawer.tgz` vào `.kobo/KoboRoot.tgz`.
4. Ngắt kết nối (eject) Kobo. Máy tự cài và khởi động lại; tab Thêm sẽ có mục **Ứng dụng**.

Trên macOS hoặc Linux có thể dùng `tools/install-to-kobo.sh` (tự sao lưu và không đụng vào thư mục `data/` của ứng dụng).

## An toàn và gỡ cài đặt

- Ngăn ứng dụng có cơ chế **failsafe**: nếu Nickel treo trong 15 giây đầu sau khi khởi động, mod tự gỡ ở lần khởi động sau và máy chạy như bản gốc.
- **Gỡ cài đặt:** tạo tệp rỗng `.adds/nickeldrawer/uninstall`, ngắt kết nối và đợi máy khởi động lại. Sách của bạn không bị động đến.
- Tạo tệp `.adds/nickeldrawer/disable` để tạm tắt mà không gỡ.
- Ứng dụng chạy với quyền root. Chỉ cài ứng dụng bạn tin tưởng. Xem [SECURITY.md](SECURITY.md).

## Ngôn ngữ

Tiếng Anh và tiếng Việt do dự án duy trì. Thêm ngôn ngữ mới chỉ cần một tệp JSON, xem [docs/TRANSLATING.md](docs/TRANSLATING.md).

## Giấy phép

[PolyForm Noncommercial 1.0.0](LICENSE): miễn phí cho cá nhân, học tập, sở thích và tổ chức phi lợi nhuận; **không dùng cho mục đích thương mại**. NickelDrawer là dự án độc lập, không liên kết với Rakuten Kobo.
