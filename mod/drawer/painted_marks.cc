#include "painted_marks.h"

#include <QPainter>
#include <QPolygonF>
#include <QPixmap>
#include <QLineF>

#include "nickel_fonts.h"

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
    setPressHighlight(HIGHLIGHT_CIRCLE); // an icon button, like Nickel's *_hit icons
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

// ---- Header icons: black strokes on white, inside a 64 px glyph box ----

static const int HEADER_ICON_SIDE = 120;
static const int HEADER_GLYPH_SIDE = 64;
static const int HEADER_STROKE = 4;
static const int GEAR_TEETH = 8;
static const qreal GEAR_TIP_RADIUS = 29;
static const qreal GEAR_ROOT_RADIUS = 21;
static const qreal GEAR_HUB_RADIUS = 8;
static const qreal GEAR_TOOTH_HALF_DEGREES = 9;  // half the tooth's angle at the tip
static const qreal GEAR_FLANK_DEGREES = 5;       // extra spread where the tooth meets the root

static void paintImportIcon(QPainter *painter, const QRectF &box)
{
    qreal centre = box.center().x();
    painter->drawLine(QPointF(centre, box.top()), QPointF(centre, box.top() + 0.62 * box.height()));
    QPolygonF head;
    head << QPointF(centre - 0.2 * box.width(), box.top() + 0.42 * box.height())
         << QPointF(centre, box.top() + 0.64 * box.height()) << QPointF(centre + 0.2 * box.width(), box.top() + 0.42 * box.height());
    painter->drawPolyline(head);
    QPolygonF tray;
    tray << QPointF(box.left(), box.top() + 0.66 * box.height()) << QPointF(box.left(), box.bottom())
         << QPointF(box.right(), box.bottom()) << QPointF(box.right(), box.top() + 0.66 * box.height());
    painter->drawPolyline(tray);
}

static QPointF pointOnCircle(const QPointF &centre, qreal radius, qreal degrees)
{
    // QLineF does the trigonometry inside QtCore, so the mod does not need libm.
    return centre + QLineF::fromPolar(radius, degrees).p2();
}

// A thin outline gear like Nickel's line icons: eight trapezoid teeth and a round hub, drawn with
// the header stroke. cmd/ndpreview/icons.go gearShapes approximates it for previews; change both.
static void paintGearIcon(QPainter *painter, const QRectF &box)
{
    QPointF centre = box.center();
    QPolygonF outline;
    qreal step = 360.0 / GEAR_TEETH;
    for (int tooth = 0; tooth < GEAR_TEETH; tooth++) {
        qreal middle = tooth * step - 90;
        outline << pointOnCircle(centre, GEAR_ROOT_RADIUS, middle - GEAR_TOOTH_HALF_DEGREES - GEAR_FLANK_DEGREES)
                << pointOnCircle(centre, GEAR_TIP_RADIUS, middle - GEAR_TOOTH_HALF_DEGREES)
                << pointOnCircle(centre, GEAR_TIP_RADIUS, middle + GEAR_TOOTH_HALF_DEGREES)
                << pointOnCircle(centre, GEAR_ROOT_RADIUS, middle + GEAR_TOOTH_HALF_DEGREES + GEAR_FLANK_DEGREES);
    }
    painter->drawPolygon(outline);
    painter->drawEllipse(centre, GEAR_HUB_RADIUS, GEAR_HUB_RADIUS);
}

static void paintRefreshIcon(QPainter *painter, const QRectF &box)
{
    QRectF ring = box.adjusted(4, 4, -4, -4);
    painter->drawArc(ring, 40 * 16, 280 * 16);
    // The arrow head at the arc's start (upper right), pointing along it.
    QPointF tip(ring.center().x() + ring.width() / 2 * 0.77, ring.center().y() - ring.height() / 2 * 0.64);
    QPolygonF head;
    head << QPointF(tip.x() + 2, tip.y() - 14) << tip << QPointF(tip.x() - 16, tip.y() - 2);
    painter->drawPolyline(head);
}

static void paintAddIcon(QPainter *painter, const QRectF &box)
{
    painter->drawLine(QPointF(box.center().x(), box.top()), QPointF(box.center().x(), box.bottom()));
    painter->drawLine(QPointF(box.left(), box.center().y()), QPointF(box.right(), box.center().y()));
}

bool HeaderIconButton::iconNamed(const QString &name, Icon *icon)
{
    if (name == QLatin1String("import")) {
        *icon = ICON_IMPORT;
    } else if (name == QLatin1String("settings")) {
        *icon = ICON_SETTINGS;
    } else if (name == QLatin1String("refresh")) {
        *icon = ICON_REFRESH;
    } else if (name == QLatin1String("add")) {
        *icon = ICON_ADD;
    } else {
        return false;
    }
    return true;
}

HeaderIconButton::HeaderIconButton(Icon iconShape, QWidget *parent) : PressArea(parent), icon(iconShape)
{
    setFixedSize(HEADER_ICON_SIDE, HEADER_ICON_SIDE);
    setPressHighlight(HIGHLIGHT_CIRCLE);
}

void HeaderIconButton::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), Qt::white);
    painter.setRenderHint(QPainter::Antialiasing, true);
    QPen pen(Qt::black, HEADER_STROKE);
    pen.setJoinStyle(Qt::RoundJoin);
    pen.setCapStyle(Qt::RoundCap);
    painter.setPen(pen);
    painter.setBrush(Qt::NoBrush);
    QRectF box((width() - HEADER_GLYPH_SIDE) / 2.0, (height() - HEADER_GLYPH_SIDE) / 2.0, HEADER_GLYPH_SIDE, HEADER_GLYPH_SIDE);
    switch (icon) {
    case ICON_IMPORT:
        paintImportIcon(&painter, box);
        break;
    case ICON_SETTINGS:
        paintGearIcon(&painter, box);
        break;
    case ICON_REFRESH:
        paintRefreshIcon(&painter, box);
        break;
    case ICON_ADD:
        paintAddIcon(&painter, box);
        break;
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
    painter.setFont(NickelFonts::font(NickelFonts::ROLE_BODY, height() / 2, false));
    painter.drawText(rect(), Qt::AlignCenter, initial);
}
