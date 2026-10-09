#include "screen_view.h"

#include <QEvent>
#include <QFontMetrics>
#include <QHBoxLayout>
#include <QLabel>

#include "debug_log.h"

#include "nickel_bridge.h"
#include "nickel_fonts.h"
#include "page_header.h"
#include "screen_canvas.h"
#include "screen_logic.h"
#include "status_bar_inset.h"

using namespace ScreenLogic;

// The layout leaves exactly the header's height free at the top (ndpreview: headerHeight).
static_assert(HEADER_HEIGHT == PageHeader::ROW_HEIGHT + PageHeader::DIVIDER_HEIGHT,
              "screen_logic.h HEADER_HEIGHT must match PageHeader");

// The header row's texts at the right, in Nickel's UI sans (never generic
// families: no Vietnamese). Sizes and greys are cmd/ndpreview's (layout.go).
static const char *const HEADER_SIDE_STYLE =
    "#screenHeaderStatus { font-family: @BODY_FAMILY@; font-size: 32px; color: #404040; background: white; }"
    "#screenHeaderAction QLabel { font-family: @BODY_FAMILY@; font-size: 32px; color: black;"
    " text-decoration: underline; background: white; }"
    "#screenHeaderAction { background: white; }";

// ---- Header action ----

HeaderActionButton::HeaderActionButton(QWidget *parent) : PressArea(parent), label(new QLabel(this))
{
    setObjectName(QStringLiteral("screenHeaderAction"));
    setFixedHeight(PageHeader::ROW_HEIGHT); // the whole row height: a 120 px tap target
    setMaximumWidth(HEADER_SIDE_WIDTH + RHYTHM);
    setPressHighlight(HIGHLIGHT_INVERT);
    label->setAlignment(Qt::AlignRight | Qt::AlignVCenter);
    label->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    QHBoxLayout *layout = new QHBoxLayout(this);
    layout->setContentsMargins(RHYTHM, 0, 0, 0);
    layout->addWidget(label);
}

void HeaderActionButton::setText(const QString &text)
{
    label->ensurePolished(); // the style sheet's font, not the default one
    QFontMetrics metrics(label->font());
    label->setText(metrics.elidedText(text, Qt::ElideRight, HEADER_SIDE_WIDTH));
}

// ---- Screen view ----

ScreenView::ScreenView(QWidget *drawerView)
    : QWidget(drawerView), host(drawerView), canvas(NULL), header(NULL), statusLabel(NULL), actionButton(NULL),
      isInputBlocked(false)
{
    setObjectName(QStringLiteral("nickelDrawerScreen"));
    setStyleSheet(NickelFonts::styleSheet(HEADER_SIDE_STYLE));
    // The canvas first, the header after it: a later sibling lies on top.
    canvas = new ScreenCanvas(host, this);
    connect(canvas, SIGNAL(componentTapped(int, QString)), this, SIGNAL(componentTapped(int, QString)));
    header = createHeader();
    placeBelowStatusBar();
    host->installEventFilter(this);
    raise();
    show();
}

// The drawer's header (page_header.h) with the status text and the header
// action at the right of the same row.
PageHeader *ScreenView::createHeader()
{
    PageHeader *pageHeader = new PageHeader(QString(), NickelBridge::textHeaderApi(), this);
    connect(pageHeader, SIGNAL(backTapped()), this, SLOT(onBackTapped()));
    statusLabel = new QLabel(pageHeader);
    statusLabel->setObjectName(QStringLiteral("screenHeaderStatus"));
    statusLabel->setAlignment(Qt::AlignRight | Qt::AlignVCenter);
    statusLabel->setMaximumWidth(HEADER_SIDE_WIDTH);
    statusLabel->hide();
    actionButton = new HeaderActionButton(pageHeader);
    connect(actionButton, SIGNAL(tapped()), this, SLOT(onHeaderActionTapped()));
    actionButton->hide();
    pageHeader->addTrailingWidget(statusLabel);
    pageHeader->addTrailingWidget(actionButton);
    DebugLog::write("screen view: header created (%s back, title, status, action)",
                    pageHeader->usesNickelHeader() ? "Nickel's" : "own");
    return pageHeader;
}

// The screen covers the drawer page except the strip under Nickel's status bar.
void ScreenView::placeBelowStatusBar()
{
    int inset = StatusBarInset::topInsetFor(host);
    setGeometry(host->rect().adjusted(0, inset, 0, 0));
    StatusBarInset::logGeometry("screen placed", this, header);
}

// Header where the drawer has it (inside the side gutters, at the top); the
// canvas under the whole view, so layout coordinates are the view's.
void ScreenView::resizeEvent(QResizeEvent *event)
{
    QWidget::resizeEvent(event);
    header->setGeometry(GUTTER, 0, CONTENT_WIDTH, header->height());
    canvas->setGeometry(rect());
    header->raise();
}

bool ScreenView::eventFilter(QObject *watched, QEvent *event)
{
    if (watched == host && event->type() == QEvent::Resize) {
        placeBelowStatusBar();
    }
    return false;
}

void ScreenView::showScreen(const ScreenModel &next)
{
    model = next;
    showHeader();
    canvas->showScreen(model);
}

// Title, status and action text; QLabel repaints only what changed (partial).
void ScreenView::showHeader()
{
    header->setTitle(model.title);
    statusLabel->ensurePolished(); // the style sheet's font, not the default one
    QFontMetrics metrics(statusLabel->font());
    statusLabel->setText(metrics.elidedText(model.status, Qt::ElideRight, HEADER_SIDE_WIDTH));
    statusLabel->setVisible(!model.status.isEmpty());
    actionButton->setText(model.headerLabel);
    actionButton->setVisible(!model.headerLabel.isEmpty());
}

int ScreenView::currentPage() const
{
    return canvas->currentPage();
}

void ScreenView::restorePageOnNextScreen(int page)
{
    canvas->restorePageOnNextScreen(page);
}

const ScreenComponent *ScreenView::component(int index) const
{
    return canvas->component(index);
}

static ScreenComponent *componentIn(ScreenModel *model, int index)
{
    if (index >= 0 && index < model->body.size()) {
        return &model->body[index];
    }
    index -= model->body.size();
    return index >= 0 && index < model->footer.size() ? &model->footer[index] : NULL;
}

void ScreenView::showToggleState(int index, bool isOn)
{
    ScreenModel next = model;
    ScreenComponent *toggle = componentIn(&next, index);
    if (toggle) {
        toggle->isOn = isOn;
        showScreen(next);
    }
}

void ScreenView::showRadioValue(int index, const QString &value)
{
    ScreenModel next = model;
    ScreenComponent *radio = componentIn(&next, index);
    if (radio) {
        radio->value = value;
        showScreen(next);
    }
}

void ScreenView::setInputBlocked(bool isBlocked)
{
    isInputBlocked = isBlocked;
    header->setBackEnabled(!isBlocked);
    actionButton->setAcceptsPresses(!isBlocked);
    canvas->setInputBlocked(isBlocked);
}

// PageHeader and PressArea emit from the event loop: the slot may delete this view.
void ScreenView::onBackTapped()
{
    DebugLog::write("screen: back tapped on \"%s\"%s", qPrintable(model.screen),
                    isInputBlocked ? ", ignored while an action is pending" : "");
    if (!isInputBlocked) {
        emit backRequested();
    }
}

void ScreenView::onHeaderActionTapped()
{
    DebugLog::write("screen: header action \"%s\" tapped%s", qPrintable(model.headerLabel),
                    isInputBlocked ? ", ignored while an action is pending" : "");
    if (!isInputBlocked) {
        emit headerActionTapped();
    }
}
