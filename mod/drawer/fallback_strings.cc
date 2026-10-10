#include "fallback_strings.h"

#include <QFile>
#include <QJsonDocument>
#include <QJsonObject>
#include <QSettings>

#include <NickelHook.h>

#include "drawer_logic.h"

static const char *const NICKEL_CONF_PATH = "/mnt/onboard/.kobo/Kobo/Kobo eReader.conf";
static const char *const LOCALE_KEY = "ApplicationPreferences/CurrentLocale";
static const char *const HOST_LOCALES_DIRECTORY = "/mnt/onboard/.adds/koboapps/locales";
static const int MAX_LOCALE_FILES = 3; // exact locale, language, en

struct Entry {
    const char *key; // the host locale key, "mod.*" (host/locales/<lang>.json)
    const char *english;
    const char *vietnamese;
};

// The last-resort texts, used for a key the locale files do not give (or when they are missing,
// so the drawer still works). Indexed by FallbackStrings::Key; a test keeps them equal to
// host/locales/en.json and vi.json. {name}, {n}, {status} are placeholders.
static const Entry TABLE[] = {
    {"mod.apps_title", "Apps", "Ứng dụng"},
    {"mod.loading", "Loading…", "Đang tải…"},
    {"mod.error_title", "Error", "Lỗi"},
    {"mod.helper_missing", "Part of the app drawer is missing. Reinstall koboapps.",
     "Thiếu một phần của ngăn ứng dụng. Hãy cài lại koboapps."},
    {"mod.helper_timeout", "The app drawer did not answer in time. Try again.",
     "Ngăn ứng dụng không trả lời kịp. Hãy thử lại."},
    {"mod.helper_failed", "The app drawer hit a problem. Try again.", "Ngăn ứng dụng gặp sự cố. Hãy thử lại."},
    {"mod.helper_bad_reply", "The app drawer got an answer it cannot read. Try again.",
     "Ngăn ứng dụng nhận được dữ liệu không đọc được. Hãy thử lại."},
    {"mod.update_needed", "Update NickelDrawer to use this version of koboapps.",
     "Hãy cập nhật NickelDrawer để dùng bản koboapps này."},
    {"mod.try_again", "Try again", "Thử lại"},
    {"mod.problems_section", "PROBLEMS", "SỰ CỐ"},
    {"mod.empty_title", "No apps yet", "Chưa có ứng dụng"},
    {"mod.empty_hint", "Copy an app folder to .adds/apps and refresh.",
     "Chép thư mục ứng dụng vào .adds/apps rồi làm mới."},
    {"mod.long_press_hint", "Long-press an app for info or remove", "Nhấn giữ một ứng dụng để xem thông tin hoặc gỡ"},
    {"mod.menu_info", "Info", "Thông tin"},
    {"mod.menu_remove", "Remove…", "Gỡ…"},
    {"mod.remove", "Remove", "Gỡ"},
    {"mod.cancel", "Cancel", "Hủy"},
    {"mod.close", "Close", "Đóng"},
    {"mod.remove_confirm", "Remove {name} from this Kobo?", "Gỡ {name} khỏi Kobo này?"},
    {"mod.wifi_off", "Wi-Fi is off. Turn it on to continue.", "Wi-Fi đang tắt. Hãy bật để tiếp tục."},
    {"mod.wifi_turn_on", "Turn on Wi-Fi", "Bật Wi-Fi"},
    {"mod.wifi_failed", "Could not connect to Wi-Fi.", "Không kết nối được Wi-Fi."},
    {"mod.info_version", "Version", "Phiên bản"},
    {"mod.info_reads", "Reads", "Đọc"},
    {"mod.info_writes", "Writes", "Ghi"},
    {"mod.info_network", "Network", "Mạng"},
    {"mod.info_folder", "Folder", "Thư mục"},
    {"mod.yes", "yes", "có"},
    {"mod.no", "no", "không"},
    {"mod.updates_one", "1 update", "1 bản cập nhật"},
    {"mod.updates_many", "{n} updates", "{n} bản cập nhật"},
    {"mod.status_line", "{name} {status}", "{name} {status}"},
    {"mod.summary_separator", " · ", " · "},
    {"mod.drawer_unavailable", "NickelDrawer is not available (see log).",
     "NickelDrawer không dùng được (xem nhật ký)."},
    {"mod.on", "On", "Bật"},
    {"mod.off", "Off", "Tắt"},
    {"mod.back", "Back", "Quay lại"},
    {"mod.cant_open", "Can't open the app", "Không mở được ứng dụng"},
    {"mod.edit_on_phone", "Change this from your phone or computer.", "Hãy đổi mục này từ điện thoại hoặc máy tính."},
    {"mod.keyboard_go", "OK", "OK"},
    {"mod.action_failed", "Couldn't do that. See the app log.", "Không thực hiện được. Xem nhật ký ứng dụng."},
    {"mod.tweaks_title", "Tweaks", "Tinh chỉnh"},
    {"mod.confirm_ok", "OK", "OK"},
    {"mod.action_unavailable", "Not available on this firmware.", "Không dùng được trên firmware này."},
    {"mod.unsupported_firmware", "NickelDrawer does not support this firmware yet",
     "NickelDrawer chưa hỗ trợ firmware này"},
    {"mod.unsupported_model", "NickelDrawer does not support this model yet", "NickelDrawer chưa hỗ trợ máy này"},
    {"mod.qt6_upgrade_pending", "Kobo is offering update 5.x — NickelDrawer will stop working",
     "Kobo đang mời cập nhật 5.x — NickelDrawer sẽ ngừng hoạt động"},
    {"mod.pending", "Working…", "Đang xử lý…"},
};

static_assert(sizeof(TABLE) / sizeof(TABLE[0]) == FallbackStrings::KEY_COUNT, "one entry per FallbackStrings::Key");

static DrawerLogic::Language currentLanguage = DrawerLogic::LANGUAGE_ENGLISH;
static QString localeTexts[FallbackStrings::KEY_COUNT]; // null: not in any locale file

// Reads one JSON file of flat strings; a missing or broken file is skipped (the file is closed
// before returning: nothing stays open on /mnt/onboard).
static bool readLocaleFile(const QString &path, QJsonObject *texts)
{
    QFile file(path);
    if (!file.open(QIODevice::ReadOnly)) {
        return false;
    }
    QByteArray json = file.readAll();
    file.close();
    QJsonParseError error;
    QJsonDocument document = QJsonDocument::fromJson(json, &error);
    if (error.error != QJsonParseError::NoError || !document.isObject()) {
        nh_log("locale file %s is not valid JSON (%s); using the next language", qPrintable(path),
               qPrintable(error.errorString()));
        return false;
    }
    *texts = document.object();
    return true;
}

// Fills the texts still missing from one locale file.
static void takeMissingTexts(const QJsonObject &texts)
{
    for (int key = 0; key < FallbackStrings::KEY_COUNT; key++) {
        QJsonValue value = texts.value(QString::fromLatin1(TABLE[key].key));
        if (localeTexts[key].isNull() && value.isString() && !value.toString().isEmpty()) {
            localeTexts[key] = value.toString();
        }
    }
}

static void loadLocaleFiles(const QByteArray &locale)
{
    for (int key = 0; key < FallbackStrings::KEY_COUNT; key++) {
        localeTexts[key] = QString();
    }
    char names[MAX_LOCALE_FILES][DrawerLogic::LOCALE_NAME_SIZE];
    int count = DrawerLogic::localeFileChain(locale.constData(), names, MAX_LOCALE_FILES);
    for (int i = 0; i < count; i++) {
        QString path = QString::fromLatin1(HOST_LOCALES_DIRECTORY) + QLatin1Char('/') + QString::fromLatin1(names[i]) +
                       QStringLiteral(".json");
        QJsonObject texts;
        if (readLocaleFile(path, &texts)) {
            takeMissingTexts(texts);
        }
    }
}

namespace FallbackStrings {

void reloadLocale()
{
    QSettings nickelConf(QString::fromUtf8(NICKEL_CONF_PATH), QSettings::IniFormat);
    QByteArray locale = nickelConf.value(QString::fromLatin1(LOCALE_KEY)).toString().toLatin1();
    currentLanguage = DrawerLogic::languageForLocale(locale.constData());
    loadLocaleFiles(locale);
}

QString text(Key key)
{
    if (key < 0 || key >= KEY_COUNT) {
        return QString();
    }
    if (!localeTexts[key].isNull()) {
        return localeTexts[key];
    }
    const Entry &entry = TABLE[key];
    bool isVietnamese = currentLanguage == DrawerLogic::LANGUAGE_VIETNAMESE;
    return QString::fromUtf8(isVietnamese ? entry.vietnamese : entry.english);
}

} // namespace FallbackStrings
