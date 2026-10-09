#ifndef NDDRAWER_DRAWER_VIEW_H
#define NDDRAWER_DRAWER_VIEW_H

#include <QList>
#include <QWidget>

class QKeyEvent;
class PressArea;
class QLabel;
class QShowEvent;
class QVBoxLayout;

// Prototype C: an empty drawer page. Header (back chevron + title), three
// placeholder rows and a page indicator. The top margin leaves room for
// Nickel's status bar, which overlays the page (status_bar_inset.h). The page-turn keys arrive as arrow
// keys (Nickel's TactileButtonFilter already applied the swap setting).
class DrawerView : public QWidget {
    Q_OBJECT
public:
    explicit DrawerView(const QString &title);

signals:
    void backRequested();

protected:
    void keyPressEvent(QKeyEvent *event);
    void showEvent(QShowEvent *event);

private slots:
    void logLayoutGeometry();
    void onBackTapped();

private:
    QWidget *createHeader(const QString &title);
    PressArea *createBackButton(QWidget *parent);
    QWidget *createRows();
    QLabel *createPageIndicator();
    void showPage(int page);
    void requestFullRefresh();
    void applyStatusBarInset();

    QVBoxLayout *rootLayout;
    QWidget *header;
    QList<QLabel *> rows;
    QLabel *pageIndicator;
    int currentPage;
    bool hasLoggedLayout;
};

#endif
