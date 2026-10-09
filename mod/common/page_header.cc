#include "page_header.h"

#include <stdlib.h>

#include <QHBoxLayout>
#include <QLabel>
#include <QPainter>
#include <QVBoxLayout>

#include "debug_log.h"
#include "nickel_fonts.h"

// Nickel's back arrow pair, as N3Dialog's back button sets them (Wishlist sets
// only the first; its TouchLabel then inverts the image).
static const char *const BACK_ARROW_PATH = ":/images/reading/back_arrow.png";
static const char *const BACK_ARROW_PRESSED_PATH = ":/images/reading/back_arrow_hit.png";

// TextHeader is 0x90 bytes on 23828 (Ui_WishlistView allocates that much);
// allocate generously and zeroed, as for IconLeftButton.
static const size_t TEXT_HEADER_ALLOCATION = 256;

static const int BACK_BUTTON_SIDE = 120;  // minimum tap target (ui-kit §1.5)
static const int DRAWN_ARROW_SIDE = 64;   // Nickel's header icon on the Libra 2 class ("headerIconWidth")
static const int DRAWN_ARROW_STROKE = 5;

// Nickel's QSS sizes the TextHeader title (N3ButtonLabel, 34 px on the Libra 2
// class) but makes it Georgia italic; Wishlist turns it into the sans upright
// (WishlistView.qss "#backLabel #label"). The family is Nickel's own UI sans,
// which has Vietnamese (nickel_fonts.h). The fallback title copies the size.
// Borders inside the TextHeader are off: our one divider runs under the row.
static const char *const HEADER_STYLE =
    "#nickelDrawerTextHeader #label { font-family: @BODY_FAMILY@; font-style: normal; color: black; border: none; }"
    "#nickelDrawerTextHeader #image { border: none; }"
    "#drawerHeaderDivider { background: #666666; }"
    "#drawerHeaderTitle { font-family: @BODY_FAMILY@; font-size: 34px; font-style: normal; color: black;"
    " background: transparent; }";

// ---- Back arrow icon ----

namespace BackArrowIcon {

QPixmap pixmap(bool isPressed)
{
    static QPixmap *images = NULL; // loaded on first use, once Nickel's resources are registered
    if (!images) {
        images = new QPixmap[2];
        images[0] = QPixmap(QString::fromLatin1(BACK_ARROW_PATH));
        images[1] = QPixmap(QString::fromLatin1(BACK_ARROW_PRESSED_PATH));
        for (int i = 0; i < 2; i++) {
            const char *path = i ? BACK_ARROW_PRESSED_PATH : BACK_ARROW_PATH;
            if (images[i].isNull()) {
                DebugLog::write("header: %s missing from Nickel's resources, arrow drawn instead", path);
            } else {
                DebugLog::write("header: %s %dx%d", path, images[i].width(), images[i].height());
            }
        }
    }
    return images[isPressed ? 1 : 0];
}

// "←" in a circle-sized box: black on white, or white on a black circle when pressed.
static void paintDrawnArrow(QPainter *painter, const QRect &area, bool isPressed)
{
    int side = qMin(DRAWN_ARROW_SIDE, qMin(area.width(), area.height()));
    QRect box(0, 0, side, side);
    box.moveCenter(area.center());
    painter->save();
    painter->setRenderHint(QPainter::Antialiasing, true);
    if (isPressed) {
        painter->setPen(Qt::NoPen);
        painter->setBrush(Qt::black);
        painter->drawEllipse(box);
    }
    painter->setPen(QPen(isPressed ? Qt::white : Qt::black, DRAWN_ARROW_STROKE, Qt::SolidLine, Qt::RoundCap));
    qreal reach = side * 0.28; // half the shaft
    QPointF centre = QRectF(box).center();
    QPointF tip(centre.x() - reach, centre.y());
    painter->drawLine(QPointF(centre.x() + reach, centre.y()), tip);
    painter->drawLine(tip, QPointF(tip.x() + reach * 0.8, centre.y() - reach * 0.8));
    painter->drawLine(tip, QPointF(tip.x() + reach * 0.8, centre.y() + reach * 0.8));
    painter->restore();
}

void paint(QPainter *painter, const QRect &area, bool isPressed)
{
    QPixmap image = pixmap(isPressed);
    if (image.isNull()) {
        paintDrawnArrow(painter, area, isPressed);
        return;
    }
    QSize size = image.size();
    if (size.width() > area.width() || size.height() > area.height()) {
        size.scale(area.size(), Qt::KeepAspectRatio);
    }
    QRect target(QPoint(0, 0), size);
    target.moveCenter(area.center());
    painter->drawPixmap(target, image);
}

} // namespace BackArrowIcon

// ---- Fallback back button ----

BackArrowButton::BackArrowButton(QWidget *parent) : PressArea(parent)
{
    setObjectName(QStringLiteral("drawerBack"));
    setFixedSize(BACK_BUTTON_SIDE, BACK_BUTTON_SIDE);
    setPressHighlight(HIGHLIGHT_SELF_PAINTED); // Nickel's own pressed arrow image
}

void BackArrowButton::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), Qt::white);
    // Left-aligned like Nickel's header icon; the rest of the target is padding.
    QRect icon(0, (height() - DRAWN_ARROW_SIDE) / 2, DRAWN_ARROW_SIDE, DRAWN_ARROW_SIDE);
    BackArrowIcon::paint(&painter, icon, isPressedStateShown());
}

// ---- Header ----

PageHeader::PageHeader(const QString &title, const NickelTextHeaderApi &nickelApi, QWidget *parent)
    : QWidget(parent), api(nickelApi), rowLayout(NULL), nickelHeader(NULL), ownBack(NULL), ownTitle(NULL)
{
    setObjectName(QStringLiteral("drawerHeader"));
    setFixedHeight(ROW_HEIGHT + DIVIDER_HEIGHT);
    setStyleSheet(NickelFonts::styleSheet(HEADER_STYLE));

    QWidget *row = new QWidget(this);
    row->setFixedHeight(ROW_HEIGHT);
    rowLayout = new QHBoxLayout(row);
    rowLayout->setContentsMargins(0, 0, 0, 0);
    rowLayout->setSpacing(0);

    QVBoxLayout *column = new QVBoxLayout(this);
    column->setContentsMargins(0, 0, 0, 0);
    column->setSpacing(0);
    column->addWidget(row);
    column->addWidget(createDivider());

    if (!createNickelHeader(title)) {
        createOwnHeader(title);
    }
}

// A separate widget, so nothing in the row (toggles included) can cover it.
QWidget *PageHeader::createDivider()
{
    QWidget *divider = new QWidget(this);
    divider->setObjectName(QStringLiteral("drawerHeaderDivider"));
    divider->setAttribute(Qt::WA_StyledBackground, true);
    divider->setFixedHeight(DIVIDER_HEIGHT);
    return divider;
}

// Nickel's TextHeader as in Ui_WishlistView::setupUi, with N3Dialog's pressed
// arrow. Only the arrow shows the press (setInvertImageOnly: reverseColors
// then swaps the image to the hit image and leaves the title alone; inverting
// the title smeared it on the panel). Its tapped() is queued: Nickel emits it
// inside its gesture handler, which still uses the header afterwards, and our
// slot may delete the page.
bool PageHeader::createNickelHeader(const QString &title)
{
    if (!api.isComplete()) {
        DebugLog::write("header: Nickel TextHeader symbols missing, own header");
        return false;
    }
    QPixmap arrow = BackArrowIcon::pixmap(false);
    if (arrow.isNull()) {
        DebugLog::write("header: no back arrow image for Nickel's TextHeader, own header");
        return false;
    }
    void *memory = calloc(1, TEXT_HEADER_ALLOCATION);
    if (!memory) {
        DebugLog::write("header: out of memory for TextHeader, own header");
        return false;
    }
    DebugLog::write("header: constructing Nickel TextHeader at %p, parent %p", memory, static_cast<void *>(this));
    api.construct(memory, rowLayout->parentWidget());
    QWidget *header = reinterpret_cast<QWidget *>(memory);
    header->setObjectName(QStringLiteral("nickelDrawerTextHeader"));
    api.setDeselectedPixmap(header, arrow);
    QPixmap pressedArrow = BackArrowIcon::pixmap(true);
    if (!pressedArrow.isNull()) {
        api.setSelectedPixmap(header, pressedArrow);
    }
    api.setInvertImageOnly(header, true);
    api.setText(header, title);
    if (!connect(header, SIGNAL(tapped()), this, SLOT(onNickelHeaderTapped()), Qt::QueuedConnection)) {
        DebugLog::write("header: cannot connect %s::tapped(), own header", header->metaObject()->className());
        delete header;
        return false;
    }
    if (!connect(header, SIGNAL(mouseDown()), this, SLOT(onNickelHeaderPressed()))) {
        DebugLog::write("header: no mouseDown() on %s, presses not logged", header->metaObject()->className());
    }
    rowLayout->addWidget(header, 0, Qt::AlignVCenter);
    rowLayout->addStretch(1);
    nickelHeader = header;
    DebugLog::write("header: Nickel %s %p, title \"%s\" left-aligned, pressed arrow only (%s)",
                    header->metaObject()->className(), static_cast<void *>(header), qPrintable(title),
                    pressedArrow.isNull() ? "inverted image" : "hit image");
    return true;
}

void PageHeader::createOwnHeader(const QString &title)
{
    ownBack = new BackArrowButton(this);
    ownTitle = new QLabel(title, this);
    ownTitle->setObjectName(QStringLiteral("drawerHeaderTitle"));
    ownTitle->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    connect(ownBack, SIGNAL(tapped()), this, SLOT(onOwnBackTapped()));
    rowLayout->addWidget(ownBack, 0, Qt::AlignVCenter);
    rowLayout->addWidget(ownTitle, 1, Qt::AlignVCenter);
    DebugLog::write("header: own copy (arrow %s), title \"%s\"",
                    BackArrowIcon::pixmap(false).isNull() ? "drawn" : "Nickel's image", qPrintable(title));
}

void PageHeader::setTitle(const QString &title)
{
    if (nickelHeader) {
        api.setText(nickelHeader, title);
    } else {
        ownTitle->setText(title);
    }
}

void PageHeader::addTrailingWidget(QWidget *widget)
{
    widget->setParent(rowLayout->parentWidget());
    rowLayout->addWidget(widget, 0, Qt::AlignVCenter);
}

// A disabled TextHeaderBase ignores its tap gesture (TextHeaderBase::tapGesture).
void PageHeader::setBackEnabled(bool isEnabled)
{
    if (nickelHeader) {
        nickelHeader->setEnabled(isEnabled);
    } else {
        ownBack->setAcceptsPresses(isEnabled);
    }
}

void PageHeader::onNickelHeaderPressed()
{
    DebugLog::write("header: Nickel back pressed (Nickel shows its pressed arrow)");
}

void PageHeader::onNickelHeaderTapped()
{
    DebugLog::write("header: Nickel back tapped");
    emit backTapped();
}

void PageHeader::onOwnBackTapped()
{
    DebugLog::write("header: own back tapped");
    emit backTapped();
}

void PageHeader::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), Qt::white);
}
