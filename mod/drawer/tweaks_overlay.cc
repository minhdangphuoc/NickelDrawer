#include "tweaks_overlay.h"

#include <QApplication>
#include <QDesktopWidget>
#include <QHBoxLayout>
#include <QLabel>
#include <QPainter>
#include <QVBoxLayout>

#include "copperui/tokens.h"
#include "debug_log.h"
#include "nickel_fonts.h"
#include "plain_dialog.h"
#include "screen_logic.h"
#include "system_icon.h"
#include "text_meter.h"
#include "tweaks_logic.h"

// The dialog tokens (docs/design-system.md A6): 1040 wide, 3 px black border, 120 px title row with
// the serif dialog title between two 120 px squares (the Nickel close icon sits in the right one).
static const int BOX_WIDTH = TweaksLogic::BOX_WIDTH;
static const int BORDER_WIDTH = TweaksLogic::BOX_BORDER;
static const int CLOSE_SIDE = TweaksLogic::TITLE_ROW_HEIGHT;

static const char *const OVERLAY_STYLE =
    "#tweaksOverlayBox { background: " COPPERUI_GREY_PAPER_WHITE "; border: " COPPERUI_PX_STROKE_BUTTON " solid "
    COPPERUI_GREY_INK_BLACK "; }"
    "#tweaksOverlayTitle { font-family: @TITLE_FAMILY@; font-size: " COPPERUI_PX_TYPE_DIALOG_TITLE_SIZE "; font-weight: bold; color: "
    COPPERUI_GREY_INK_BLACK "; background: transparent; }";

CloseCrossButton::CloseCrossButton(QWidget *parent) : PressArea(parent)
{
    setObjectName(QStringLiteral("tweaksClose"));
    setFixedSize(CLOSE_SIDE, CLOSE_SIDE);
    const SystemIcon::Icon &cross = SystemIcon::icon(SystemIcon::CLOSE);
    setPressHighlight(cross.pressed.isNull() ? HIGHLIGHT_CIRCLE : HIGHLIGHT_SELF_PAINTED);
}

bool CloseCrossButton::hasIcon()
{
    return !SystemIcon::icon(SystemIcon::CLOSE).isNull();
}

void CloseCrossButton::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), Qt::white);
    const SystemIcon::Icon &cross = SystemIcon::icon(SystemIcon::CLOSE);
    bool isPressedImage = isPressedStateShown() && !cross.pressed.isNull();
    SystemIcon::drawCentred(&painter, rect(), isPressedImage ? cross.pressed : cross.normal);
}

TweaksOverlay::TweaksOverlay(const QString &title, QWidget *content)
    : QDialog(NULL, Qt::Dialog | Qt::FramelessWindowHint), outsideArea(NULL)
{
    setAttribute(Qt::WA_DeleteOnClose, true);
    setStyleSheet(NickelFonts::styleSheet(OVERLAY_STYLE));

    // First child, so it lies under the box: every tap outside the box lands here.
    PressArea *outside = new PressArea(this);
    outside->setObjectName(QStringLiteral("tweaksOverlayOutside"));
    outside->setPressHighlight(PressArea::HIGHLIGHT_NONE);
    connect(outside, SIGNAL(tapped()), this, SLOT(reject()));
    outsideArea = outside;

    QWidget *box = new QWidget(this);
    box->setObjectName(QStringLiteral("tweaksOverlayBox"));
    box->setAttribute(Qt::WA_StyledBackground, true);
    box->setFixedWidth(BOX_WIDTH);
    QVBoxLayout *boxLayout = new QVBoxLayout(box);
    boxLayout->setContentsMargins(BORDER_WIDTH, BORDER_WIDTH, BORDER_WIDTH, BORDER_WIDTH);
    boxLayout->setSpacing(0);
    boxLayout->addWidget(createTitleRow(title, box));
    content->setParent(box);
    boxLayout->addWidget(content);

    QVBoxLayout *screenLayout = new QVBoxLayout(this);
    screenLayout->setContentsMargins(0, 0, 0, 0);
    screenLayout->addWidget(box, 0, Qt::AlignCenter);
}

// N3Dialog's top row: a title centred between two equal squares, ✕ on the right.
QWidget *TweaksOverlay::createTitleRow(const QString &title, QWidget *box)
{
    QWidget *row = new QWidget(box);
    row->setFixedHeight(TweaksLogic::TITLE_ROW_HEIGHT);
    QHBoxLayout *layout = new QHBoxLayout(row);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(0);
    layout->addSpacing(CLOSE_SIDE);
    // The title is one measured line between the two squares, elided if it cannot fit.
    ScreenLogic::TextRun titleRun = {ScreenLogic::tokens::TYPE_DIALOG_TITLE_SIZE, ScreenLogic::FAMILY_SERIF, true};
    int titleWidth = BOX_WIDTH - 2 * BORDER_WIDTH - 2 * CLOSE_SIDE;
    QLabel *titleLabel = new QLabel(QtTextMeter::elideLine(title, titleWidth, titleRun), row);
    titleLabel->setObjectName(QStringLiteral("tweaksOverlayTitle"));
    titleLabel->setAlignment(Qt::AlignCenter);
    titleLabel->setTextFormat(Qt::PlainText);
    layout->addWidget(titleLabel, 1);
    if (!CloseCrossButton::hasIcon()) {
        DebugLog::write("tweaks overlay: close icon not installed, no close button (tap outside closes)");
        layout->addSpacing(CLOSE_SIDE);
        return row;
    }
    CloseCrossButton *close = new CloseCrossButton(row);
    connect(close, SIGNAL(tapped()), this, SLOT(reject()));
    layout->addWidget(close);
    return row;
}

void TweaksOverlay::present()
{
    backdrop = PlainDialog::grabPageUnder(this);
    setGeometry(QApplication::desktop()->screenGeometry());
    open();
}

void TweaksOverlay::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    if (backdrop.isNull()) {
        painter.fillRect(rect(), Qt::white);
    } else {
        painter.drawPixmap(rect(), backdrop);
    }
}

void TweaksOverlay::resizeEvent(QResizeEvent *event)
{
    QDialog::resizeEvent(event);
    outsideArea->setGeometry(rect());
}
