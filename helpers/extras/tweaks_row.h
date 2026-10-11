#ifndef EXTRAS_TWEAKS_ROW_H
#define EXTRAS_TWEAKS_ROW_H

#include <QObject>
#include <QPointer>
#include <QWidget>

// The "Tweaks" row in Nickel's More tab (a Nickel IconLeftButton, MoreRow). Tap opens the Tweaks
// panel (TweaksSession). Deliberately not a child of the row (re/notes/protoA-crash.md); deleted
// with it.
class TweaksRow : public QObject {
    Q_OBJECT
public:
    // Handler of the Ui_MoreView::setupUi hook. Fails soft: logs and adds nothing when anything is
    // missing; the rest of the More tab is unaffected.
    static void addTo(QWidget *moreView);

private slots:
    void onTapped();
    // From the event loop, after Nickel's tap gesture handler returned.
    void handleTap();

private:
    explicit TweaksRow(QWidget *row);

    QPointer<QWidget> row;
    bool isTapPending;
};

#endif
