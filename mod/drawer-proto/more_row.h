#ifndef NDDRAWER_MORE_ROW_H
#define NDDRAWER_MORE_ROW_H

#include <QObject>
#include <QPointer>
#include <QWidget>

// Prototype A: one extra IconLeftButton row ("Apps" / "Ứng dụng") in Nickel's
// More tab, added right after MoreView's constructor.
namespace MoreRow {

void addTo(QWidget *moreView);

} // namespace MoreRow

// Receives the row's IconButtonBase::tapped() signal.
//
// Deliberately NOT a child of the row: IconButtonBase::tapGesture runs
// __dynamic_cast on every direct child of the button on press and release
// (re/notes/protoA-crash.md). The receiver is deleted when the row is.
class MoreRowTapReceiver : public QObject {
    Q_OBJECT
public:
    explicit MoreRowTapReceiver(QWidget *row);

public slots:
    void onTapped();

private slots:
    // Runs from the event loop after Nickel's tap gesture handler returned,
    // the same context as ndhello's proven timer toast.
    void handleTap();

private:
    QPointer<QWidget> row;
    bool isTapPending;
};

#endif
