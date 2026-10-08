#include "drawer_view.h"

#include <QHBoxLayout>
#include <QKeyEvent>
#include <QLabel>
#include <QPushButton>
#include <QVBoxLayout>

#include <NickelHook.h>

// Sizes in device pixels (docs/spec/ui-kit.md §2).
static const int SIDE_GUTTER = 72;
static const int HEADER_HEIGHT = 120;
static const int ROW_HEIGHT = 152;
static const int ROWS_PER_PAGE = 3;
static const int PAGE_COUNT = 2;

// Kobo's patched Qt reads custom widget attributes as e-ink refresh hints;
// 0x80 asks for a flashing full update (re/notes/G2-findings.md Q4).
static const Qt::WidgetAttribute FULL_REFRESH_ATTRIBUTE = static_cast<Qt::WidgetAttribute>(0x80);

static const char *const VIEW_STYLE =
    "DrawerView { background: white; }"
    "QLabel, QPushButton { color: black; background: white; border: none; }"
    "#drawerTitle { font-size: 44px; font-family: serif; }"
    "#drawerBack { font-size: 64px; min-width: 120px; min-height: 120px; }"
    "#drawerRow { font-size: 36px; border-bottom: 2px solid #c0c0c0; }"
    "#drawerPageIndicator { font-size: 32px; }";

DrawerView::DrawerView(const QString &title)
    : QWidget(NULL), pageIndicator(NULL), currentPage(0)
{
    setObjectName(QStringLiteral("nickelDrawerView"));
    setStyleSheet(QString::fromLatin1(VIEW_STYLE));
    setAttribute(Qt::WA_StyledBackground, true); // plain QWidget subclasses ignore "background" otherwise
    // Same status-bar properties MoreView sets, so Nickel's pushView does not guess.
    setProperty("showIndicators", true);
    setProperty("showStatusBottomLine", false);
    setProperty("showTitle", true);
    setFocusPolicy(Qt::StrongFocus);

    QVBoxLayout *layout = new QVBoxLayout(this);
    layout->setContentsMargins(SIDE_GUTTER, 0, SIDE_GUTTER, 0);
    layout->setSpacing(0);
    layout->addWidget(createHeader(title));
    layout->addWidget(createRows());
    layout->addStretch(1);
    layout->addWidget(createPageIndicator());

    showPage(0);
}

QWidget *DrawerView::createHeader(const QString &title)
{
    QWidget *header = new QWidget(this);
    header->setFixedHeight(HEADER_HEIGHT);

    QPushButton *back = new QPushButton(QString::fromUtf8("‹"), header);
    back->setObjectName(QStringLiteral("drawerBack"));
    back->setFlat(true);
    back->setFocusPolicy(Qt::NoFocus); // page keys must reach the view, not the button
    connect(back, SIGNAL(clicked()), this, SIGNAL(backRequested()));

    QLabel *titleLabel = new QLabel(title, header);
    titleLabel->setObjectName(QStringLiteral("drawerTitle"));

    QHBoxLayout *layout = new QHBoxLayout(header);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->addWidget(back);
    layout->addWidget(titleLabel, 1);
    return header;
}

QWidget *DrawerView::createRows()
{
    QWidget *list = new QWidget(this);
    QVBoxLayout *layout = new QVBoxLayout(list);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(0);
    for (int i = 0; i < ROWS_PER_PAGE; i++) {
        QLabel *row = new QLabel(list);
        row->setObjectName(QStringLiteral("drawerRow"));
        row->setFixedHeight(ROW_HEIGHT);
        layout->addWidget(row);
        rows.append(row);
    }
    return list;
}

QLabel *DrawerView::createPageIndicator()
{
    pageIndicator = new QLabel(this);
    pageIndicator->setObjectName(QStringLiteral("drawerPageIndicator"));
    pageIndicator->setAlignment(Qt::AlignCenter);
    pageIndicator->setFixedHeight(HEADER_HEIGHT);
    return pageIndicator;
}

void DrawerView::showPage(int page)
{
    currentPage = page;
    for (int i = 0; i < rows.size(); i++) {
        int placeholderNumber = page * ROWS_PER_PAGE + i + 1;
        rows[i]->setText(QStringLiteral("Placeholder %1").arg(placeholderNumber));
    }
    pageIndicator->setText(QStringLiteral("%1 / %2").arg(page + 1).arg(PAGE_COUNT));
}

void DrawerView::requestFullRefresh()
{
    setAttribute(FULL_REFRESH_ATTRIBUTE, true);
    repaint();
    setAttribute(FULL_REFRESH_ATTRIBUTE, false);
}

void DrawerView::keyPressEvent(QKeyEvent *event)
{
    int key = event->key();
    bool isPrevious = key == Qt::Key_Left || key == Qt::Key_Up;
    bool isNext = key == Qt::Key_Right || key == Qt::Key_Down;
    if (!isPrevious && !isNext) {
        nh_log("drawer: key 0x%x ignored", key);
        QWidget::keyPressEvent(event);
        return;
    }

    int target = currentPage + (isNext ? 1 : -1);
    bool isInRange = target >= 0 && target < PAGE_COUNT;
    nh_log("drawer: key 0x%x (%s) page %d -> %d%s", key, isNext ? "next" : "previous", currentPage + 1,
           (isInRange ? target : currentPage) + 1, isInRange ? "" : " (at edge)");
    if (isInRange) {
        showPage(target);
        requestFullRefresh();
    }
    event->accept();
}
