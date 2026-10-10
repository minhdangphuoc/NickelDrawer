#ifndef STATUSBARSHOT_NICKEL_BRIDGE_H
#define STATUSBARSHOT_NICKEL_BRIDGE_H

// The only part of the helper that touches libnickel: the NickelHook hook and symbol tables, the
// model and firmware check and every call into Nickel. Everything else is plain Qt 5.2 and this
// interface.
//
// Fail soft: every symbol is optional. While the bridge is inert (kill switch, unsupported model or
// firmware, a missing symbol, or init not run) the status-bar hook only calls Nickel's original and
// every wrapper below does nothing.

#include <QObject>
#include <QPixmap>
#include <QString>

#include <NickelHook.h>

class QWidget;

namespace NickelBridge {

// NickelHook tables, referenced by the NickelHook() declaration in statusbarshot.cc.
extern struct nh_hook hookTable[];
extern struct nh_dlsym symbolTable[];

// Called once from init(): reads .kobo/version and activates the bridge only on a supported model
// with a tested firmware and every symbol present. Returns true when active.
bool activate();
bool isActive();

// Called after Nickel's StatusBarView constructor, on the GUI thread, while the bridge is active.
typedef void (*StatusBarBuiltHandler)(QWidget *statusBar);
void setStatusBarBuiltHandler(StatusBarBuiltHandler handler);

// A Nickel StatusBarIcon (a TouchLabel), or NULL (logged). Not shown or laid out yet.
QWidget *createStatusBarIcon(QWidget *parent);
void setIconPixmaps(QWidget *icon, const QPixmap &normal, const QPixmap &pressed);

// Nickel's ScreenshotMaker: its own instance, made on first use and kept.
// The name of the file the next save would write or just wrote ("/mnt/onboard/screen_NNN.png");
// empty when the maker is unavailable.
QString screenshotName();
// ScreenshotMaker::saveCurrentScreen(): blocks for the PNG encode. false when unavailable.
bool saveScreenshot();

// Nickel's toast; only logged when its symbols are missing.
void showToast(const QString &primary, const QString &secondary);

} // namespace NickelBridge

// The one toast about an unsupported device or firmware; deletes itself.
class DisabledNotice : public QObject {
    Q_OBJECT
public:
    DisabledNotice();

public slots:
    void show();
};

#endif
