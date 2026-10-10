#include "tweaks_overlay.h"

#include <QApplication>
#include <QDesktopWidget>
#include <QHBoxLayout>
#include <QLabel>
#include <QPainter>
#include <QVBoxLayout>

#include "plain_dialog.h"
#include "nickel_fonts.h"

static const int BOX_WIDTH = 1140;
static const int BORDER_WIDTH = 1;
static const int CLOSE_SIDE = 110; // N3Dialog.qss #closeContainer on Storm
static const int CROSS_INSET = 36;
static const int CROSS_PEN_WIDTH = 4;
static const char *const CLOSE_IMAGE = ":/images/search/global_close.png";
static const char *const CLOSE_PRESSED_IMAGE = ":/images/search/global_close_hit.png";

static const char *const OVERLAY_STYLE = "#tweaksOverlayBox { background: white; border: 1px solid black; }"
                                         "#tweaksOverlayTitle { font-family: @BODY_FAMILY@; font-size: 30px; "
                                         "color: black; background: transparent; }";

CloseCrossButton::CloseCrossButton(QWidget *parent) : PressArea(parent)
{
    setObjectName(QStringLiteral("tweaksClose"));
    setFixedSize(CLOSE_SIDE, CLOSE_SIDE);
    setPressHighlight(HIGHLIGHT_SELF_PAINTED);
}

void CloseCrossButton::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), Qt::white);
    QPixmap image(QString::fromLatin1(isPressedStateShown() ? CLOSE_PRESSED_IMAGE : CLOSE_IMAGE));
    if (!image.isNull()) {
        painter.drawPixmap((width() - image.width()) / 2, (height() - image.height()) / 2, image);
        return;
    }
    if (isPressedStateShown()) {
        painter.setBrush(Qt::black);
        painter.setPen(Qt::NoPen);
        painter.drawEllipse(rect().adjusted(CROSS_INSET / 2, CROSS_INSET / 2, -CROSS_INSET / 2, -CROSS_INSET / 2));
    }
    painter.setRenderHint(QPainter::Antialiasing, true);
    painter.setPen(QPen(isPressedStateShown() ? Qt::white : Qt::black, CROSS_PEN_WIDTH));
    QRect cross = rect().adjusted(CROSS_INSET, CROSS_INSET, -CROSS_INSET, -CROSS_INSET);
    painter.drawLine(cross.topLeft(), cross.bottomRight());
    painter.drawLine(cross.topRight(), cross.bottomLeft());
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
