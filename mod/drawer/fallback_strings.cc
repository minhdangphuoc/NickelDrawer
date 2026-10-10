#include "fallback_strings.h"

#include <QSettings>

#include "drawer_logic.h"

static const char *const NICKEL_CONF_PATH = "/mnt/onboard/.kobo/Kobo/Kobo eReader.conf";
static const char *const LOCALE_KEY = "ApplicationPreferences/CurrentLocale";

struct Translation {
    const char *english;
    const char *vietnamese;
};

// Indexed by FallbackStrings::Key; {name}, {n}, {status} are placeholders.
static const Translation TABLE[] = {
    {"Apps", "Ứng dụng"},
    {"Loading…", "Đang tải…"},
    {"Error", "Lỗi"},
    {"appsctl is not installed. Reinstall koboapps.", "Chưa cài appsctl. Hãy cài lại koboapps."},
    {"appsctl did not answer in time.", "appsctl không trả lời kịp."},
    {"appsctl failed (code {n}).", "appsctl bị lỗi (mã {n})."},
    {"appsctl sent an unreadable reply.", "appsctl trả về dữ liệu không đọc được."},
    {"Update NickelDrawer to use this version of koboapps.", "Hãy cập nhật NickelDrawer để dùng bản koboapps này."},
    {"Try again", "Thử lại"},
    {"PROBLEMS", "SỰ CỐ"},
    {"No apps yet", "Chưa có ứng dụng"},
    {"Copy an app folder to .adds/apps and refresh.", "Chép thư mục ứng dụng vào .adds/apps rồi làm mới."},
    {"Long-press an app for info or remove", "Nhấn giữ một ứng dụng để xem thông tin hoặc gỡ"},
    {"Info", "Thông tin"},
    {"Remove…", "Gỡ…"},
    {"Remove", "Gỡ"},
    {"Cancel", "Hủy"},
    {"Close", "Đóng"},
    {"Remove {name} from this Kobo?", "Gỡ {name} khỏi Kobo này?"},
    {"Wi-Fi is off. Turn it on to continue.", "Wi-Fi đang tắt. Hãy bật để tiếp tục."},
    {"Turn on Wi-Fi", "Bật Wi-Fi"},
    {"Could not connect to Wi-Fi.", "Không kết nối được Wi-Fi."},
    {"Version", "Phiên bản"},
    {"Reads", "Đọc"},
    {"Writes", "Ghi"},
    {"Network", "Mạng"},
    {"Folder", "Thư mục"},
    {"yes", "có"},
    {"no", "không"},
    {"1 update", "1 bản cập nhật"},
    {"{n} updates", "{n} bản cập nhật"},
    {"{name} {status}", "{name} {status}"},
    {" · ", " · "},
    {"NickelDrawer is not available (see log).", "NickelDrawer không dùng được (xem nhật ký)."},
    {"On", "Bật"},
    {"Off", "Tắt"},
    {"Back", "Quay lại"},
    {"Can't open the app", "Không mở được ứng dụng"},
    {"Change this from your phone or computer.", "Hãy đổi mục này từ điện thoại hoặc máy tính."},
    {"OK", "OK"},
    {"Couldn't do that. See the app log.", "Không thực hiện được. Xem nhật ký ứng dụng."},
};

static_assert(sizeof(TABLE) / sizeof(TABLE[0]) == FallbackStrings::KEY_COUNT, "one translation per FallbackStrings::Key");

static DrawerLogic::Language currentLanguage = DrawerLogic::LANGUAGE_ENGLISH;

namespace FallbackStrings {

void reloadLocale()
{
    QSettings nickelConf(QString::fromUtf8(NICKEL_CONF_PATH), QSettings::IniFormat);
    QByteArray locale = nickelConf.value(QString::fromLatin1(LOCALE_KEY)).toString().toLatin1();
    currentLanguage = DrawerLogic::languageForLocale(locale.constData());
}

QString text(Key key)
{
    if (key < 0 || key >= KEY_COUNT) {
        return QString();
    }
    const Translation &entry = TABLE[key];
    bool isVietnamese = currentLanguage == DrawerLogic::LANGUAGE_VIETNAMESE;
    return QString::fromUtf8(isVietnamese ? entry.vietnamese : entry.english);
}

} // namespace FallbackStrings
