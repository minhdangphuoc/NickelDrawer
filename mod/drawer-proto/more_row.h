#ifndef NDDRAWER_MORE_ROW_H
#define NDDRAWER_MORE_ROW_H

#include <QObject>

class QWidget;

// Prototype A: one extra IconLeftButton row ("Apps" / "Ứng dụng") in Nickel's
// More tab, added right after MoreView's constructor.
namespace MoreRow {

void addTo(QWidget *moreView);

} // namespace MoreRow

// Receives the row's IconButtonBase::tapped() signal. Child of the row, so it
// is deleted together with the More view.
class MoreRowTapReceiver : public QObject {
    Q_OBJECT
public:
    explicit MoreRowTapReceiver(QObject *row);

public slots:
    void onTapped();
};

#endif
