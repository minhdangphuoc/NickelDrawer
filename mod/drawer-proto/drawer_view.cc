#include "drawer_view.h"

#include <QHBoxLayout>
#include <QKeyEvent>
#include <QLabel>
#include <QTimer>
#include <QVBoxLayout>

#include "debug_log.h"
#include "nickel_fonts.h"
#include "press_area.h"
#include "status_bar_inset.h"

// Sizes in device pixels (docs/spec/ui-kit.md §2).
static const int SIDE_GUTTER = 72;
static const int HEADER_HEIGHT = 120;
static const int BACK_BUTTON_SIDE = 120; // minimum tap target (ui-kit §1.5)
static const int ROW_HEIGHT = 152;
static const int ROWS_PER_PAGE = 3;
static const int PAGE_COUNT = 2;

// Kobo's patched Qt reads custom widget attributes as e-ink refresh hints;
// 0x80 asks for a flashing full update (re/notes/G2-findings.md Q4).
static const Qt::WidgetAttribute FULL_REFRESH_ATTRIBUTE = static_cast<Qt::WidgetAttribute>(0x80);

// Families are Nickel's own, filled in at run time (nickel_fonts.h): the
// generic "serif" had no Vietnamese and drew "Ứng dụng" as "□ng d□ng".
static const char *const VIEW_STYLE =
    "DrawerView { background: white; }"
    "QWidget { font-family: @BODY_FAMILY@; }"
    "QLabel { color: black; background: white; border: none; }"
    "#drawerHeader { background: white; }"
    "#drawerTitle { font-family: @TITLE_FAMILY@; font-size: 44px; font-weight: bold; }"
    "#drawerBackGlyph { font-size: 64px; }"
    "#drawerRow { font-size: 36px; border-bottom: 2px solid #c0c0c0; }"
    "#drawerPageIndicator { font-size: 32px; }";

DrawerView::DrawerView(const QString &title)
    : QWidget(NULL), rootLayout(NULL), header(NULL), pageIndicator(NULL), currentPage(0), hasLoggedLayout(false)
{
    setObjectName(QStringLiteral("nickelDrawerView"));
    setStyleSheet(NickelFonts::styleSheet(VIEW_STYLE));
    setAttribute(Qt::WA_StyledBackground, true); // plain QWidget subclasses ignore "background" otherwise
    // Same status-bar properties MoreView sets, so Nickel's pushView does not guess.
    setProperty("showIndicators", true);
    setProperty("showStatusBottomLine", false);
    setProperty("showTitle", true);
    setFocusPolicy(Qt::StrongFocus);

    // Top margin is the status-bar inset, known only once Nickel shows the view (showEvent).
    rootLayout = new QVBoxLayout(this);
    rootLayout->setContentsMargins(SIDE_GUTTER, StatusBarInset::FALLBACK_TOP_INSET, SIDE_GUTTER, 0);
    rootLayout->setSpacing(0);
    header = createHeader(title);
    rootLayout->addWidget(header);
    rootLayout->addWidget(createRows());
    rootLayout->addStretch(1);
    rootLayout->addWidget(createPageIndicator());

    showPage(0);
    DebugLog::write("drawer view: constructed %p, header %p", static_cast<void *>(this), static_cast<void *>(header));
}

QWidget *DrawerView::createHeader(const QString &title)
{
    QWidget *header = new QWidget(this);
    header->setObjectName(QStringLiteral("drawerHeader"));
    header->setAttribute(Qt::WA_StyledBackground, true);
    header->setFixedHeight(HEADER_HEIGHT);

    QLabel *titleLabel = new QLabel(title, header);
    titleLabel->setObjectName(QStringLiteral("drawerTitle"));

    QHBoxLayout *layout = new QHBoxLayout(header);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->addWidget(createBackButton(header));
    layout->addWidget(titleLabel, 1);
    DebugLog::write("drawer view: header created (back + title \"%s\")", qPrintable(title));
    return header;
}

// A plain QPushButton never got clicked() on the device (Nickel delivers taps
// as its own gesture, not as mouse clicks); PressArea takes every path.
PressArea *DrawerView::createBackButton(QWidget *parent)
{
    PressArea *back = new PressArea(parent);
    back->setObjectName(QStringLiteral("drawerBack"));
    back->setFixedSize(BACK_BUTTON_SIDE, BACK_BUTTON_SIDE);
    QLabel *glyph = new QLabel(QString::fromUtf8("‹"), back);
    glyph->setObjectName(QStringLiteral("drawerBackGlyph"));
    glyph->setAlignment(Qt::AlignCenter);
    glyph->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    QHBoxLayout *layout = new QHBoxLayout(back);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->addWidget(glyph);
    connect(back, SIGNAL(tapped()), this, SLOT(onBackTapped()));
    return back;
}

void DrawerView::onBackTapped()
{
    DebugLog::write("drawer view: back tapped, emitting backRequested");
    emit backRequested();
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

void DrawerView::showEvent(QShowEvent *event)
{
    QWidget::showEvent(event);
    applyStatusBarInset();
    StatusBarInset::logGeometry("after show", this, header);
    if (!hasLoggedLayout) {
        hasLoggedLayout = true;
        QTimer::singleShot(0, this, SLOT(logLayoutGeometry())); // runs after the layout pass
    }
}

void DrawerView::applyStatusBarInset()
{
    int inset = StatusBarInset::topInsetFor(this);
    rootLayout->setContentsMargins(SIDE_GUTTER, inset, SIDE_GUTTER, 0);
    rootLayout->activate();
    DebugLog::write("drawer view: top inset %d", inset);
}

void DrawerView::logLayoutGeometry()
{
    StatusBarInset::logGeometry("after first layout", this, header);
}

void DrawerView::keyPressEvent(QKeyEvent *event)
{
    int key = event->key();
    bool isPrevious = key == Qt::Key_Left || key == Qt::Key_Up;
    bool isNext = key == Qt::Key_Right || key == Qt::Key_Down;
    if (!isPrevious && !isNext) {
        DebugLog::write("drawer view: key 0x%x ignored", key);
        QWidget::keyPressEvent(event);
        return;
    }

    int target = currentPage + (isNext ? 1 : -1);
    bool isInRange = target >= 0 && target < PAGE_COUNT;
    DebugLog::write("drawer view: key 0x%x (%s) page %d -> %d%s", key, isNext ? "next" : "previous", currentPage + 1,
           (isInRange ? target : currentPage) + 1, isInRange ? "" : " (at edge)");
    if (isInRange) {
        showPage(target);
        requestFullRefresh();
    }
    event->accept();
}
