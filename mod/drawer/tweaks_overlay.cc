#include "tweaks_overlay.h"

#include <QApplication>
#include <QDesktopWidget>
#include <QHBoxLayout>
#include <QLabel>
#include <QPainter>
#include <QVBoxLayout>

#include "debug_log.h"
#include "nickel_fonts.h"
#include "plain_dialog.h"
#include "system_icon.h"

static const int BOX_WIDTH = 1140;
static const int BORDER_WIDTH = 1;
static const int CLOSE_SIDE = 110; // N3Dialog.qss #closeContainer on Storm

static const char *const OVERLAY_STYLE = "#tweaksOverlayBox { background: white; border: 1px solid black; }"
                                         "#tweaksOverlayTitle { font-family: @BODY_FAMILY@; font-size: 30px; "
                                         "color: black; background: transparent; }";

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
    QHBoxLayout *layout = new QHBoxLayout(row);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(0);
    layout->addSpacing(CLOSE_SIDE);
    QLabel *titleLabel = new QLabel(title, row);
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
