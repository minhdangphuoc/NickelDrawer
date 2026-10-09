#include "painted_marks.h"

#include <QPainter>
#include <QPixmap>

static const int DOT_DIAMETER = 20;
static const int TOGGLE_SIDE = 120;      // minimum tap target (ui-kit §1.5)
static const int TOGGLE_GLYPH_SIDE = 44;
static const int TOGGLE_BAR_HEIGHT = 6;
static const int TOGGLE_UNDERLINE_HEIGHT = 4;
static const int TOGGLE_UNDERLINE_GAP = 14;
static const int OUTLINE_WIDTH = 2;

UpdateDot::UpdateDot(QWidget *parent) : QWidget(parent)
{
    setFixedSize(DOT_DIAMETER, DOT_DIAMETER);
    setAttribute(Qt::WA_TransparentForMouseEvents, true);
}

void UpdateDot::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.setRenderHint(QPainter::Antialiasing, true);
    painter.setPen(Qt::NoPen);
    painter.setBrush(Qt::black);
    painter.drawEllipse(rect());
}

ViewToggleIcon::ViewToggleIcon(Shape iconShape, QWidget *parent) : PressArea(parent), shape(iconShape), isActive(false)
{
    setFixedSize(TOGGLE_SIDE, TOGGLE_SIDE);
}

void ViewToggleIcon::setActive(bool active)
{
    if (isActive != active) {
        isActive = active;
        update();
    }
}

void ViewToggleIcon::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), Qt::white);
    QRect glyph((width() - TOGGLE_GLYPH_SIDE) / 2, (height() - TOGGLE_GLYPH_SIDE) / 2, TOGGLE_GLYPH_SIDE,
                TOGGLE_GLYPH_SIDE);
    if (shape == SHAPE_LIST) {
        paintList(&painter, glyph);
    } else {
        paintGrid(&painter, glyph);
    }
    if (isActive) {
        int underlineTop = glyph.bottom() + TOGGLE_UNDERLINE_GAP;
        painter.fillRect(QRect(glyph.left(), underlineTop, glyph.width(), TOGGLE_UNDERLINE_HEIGHT), Qt::black);
    }
}

void ViewToggleIcon::paintList(QPainter *painter, const QRect &area)
{
    int gap = (area.height() - 3 * TOGGLE_BAR_HEIGHT) / 2;
    for (int bar = 0; bar < 3; bar++) {
        int top = area.top() + bar * (TOGGLE_BAR_HEIGHT + gap);
        painter->fillRect(QRect(area.left(), top, area.width(), TOGGLE_BAR_HEIGHT), Qt::black);
    }
}

void ViewToggleIcon::paintGrid(QPainter *painter, const QRect &area)
{
    int gap = TOGGLE_BAR_HEIGHT;
    int cell = (area.width() - gap) / 2;
    painter->setPen(QPen(Qt::black, OUTLINE_WIDTH));
    painter->setBrush(Qt::NoBrush);
    for (int row = 0; row < 2; row++) {
        for (int column = 0; column < 2; column++) {
            painter->drawRect(area.left() + column * (cell + gap), area.top() + row * (cell + gap), cell - 1, cell - 1);
        }
    }
}

AppMark::AppMark(const QString &iconPath, const QString &name, int side, QWidget *parent) : QWidget(parent)
{
    setFixedSize(side, side);
    setAttribute(Qt::WA_TransparentForMouseEvents, true);
    if (!iconPath.isEmpty()) {
        // QPixmap reads the whole file and closes it (SVG needs Nickel's svg
        // image plugin; without it the initial is drawn instead).
        icon = QPixmap(iconPath).scaled(side, side, Qt::KeepAspectRatio, Qt::SmoothTransformation);
    }
    initial = name.left(1).toUpper();
}

void AppMark::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    if (!icon.isNull()) {
        painter.drawPixmap((width() - icon.width()) / 2, (height() - icon.height()) / 2, icon);
        return;
    }
    painter.setPen(QPen(Qt::black, OUTLINE_WIDTH));
    painter.setBrush(Qt::NoBrush);
    painter.drawRect(rect().adjusted(1, 1, -1, -1));
    QFont font = painter.font();
    font.setPixelSize(height() / 2);
    painter.setFont(font);
    painter.drawText(rect(), Qt::AlignCenter, initial);
}
