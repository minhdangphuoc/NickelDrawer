#ifndef NDDRAWER_DRAWER_VIEW_H
#define NDDRAWER_DRAWER_VIEW_H

#include <QList>
#include <QWidget>

class QKeyEvent;
class QLabel;

// Prototype C: an empty drawer page. Header (back chevron + title), three
// placeholder rows and a page indicator. The page-turn keys arrive as arrow
// keys (Nickel's TactileButtonFilter already applied the swap setting).
class DrawerView : public QWidget {
    Q_OBJECT
public:
    explicit DrawerView(const QString &title);

signals:
    void backRequested();

protected:
    void keyPressEvent(QKeyEvent *event);

private:
    QWidget *createHeader(const QString &title);
    QWidget *createRows();
    QLabel *createPageIndicator();
    void showPage(int page);
    void requestFullRefresh();

    QList<QLabel *> rows;
    QLabel *pageIndicator;
    int currentPage;
};

#endif
