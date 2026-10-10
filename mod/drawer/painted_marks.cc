#include "painted_marks.h"

#include <QFontMetrics>
#include <QPainter>
#include <QPixmap>

#include "copperui/tokens.h"
#include "debug_log.h"
#include "nickel_fonts.h"
#include "system_icon.h"

static const int DOT_DIAMETER = copperui::v1::DOT_UPDATE;
static const int TOGGLE_SIDE = copperui::v1::TARGET_MIN;
static const int TOGGLE_ICON_SIDE = copperui::v1::ICON_TOOLBAR;
static const int TOGGLE_UNDERLINE_HEIGHT = copperui::v1::UNDERLINE_HEIGHT;
static const int TOGGLE_UNDERLINE_GAP = copperui::v1::UNDERLINE_GAP;
static const int HEADER_ICON_SIDE = copperui::v1::HEADER_ICON_TARGET;
static const int HEADER_LABEL_PADDING = 2 * copperui::v1::SPACE_UNIT;
static const int HEADER_LABEL_SIZE = copperui::v1::TYPE_HEADER_LABEL_SIZE;

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

static const SystemIcon::Icon &toggleIcon(ViewToggleIcon::Shape shape)
{
    return SystemIcon::icon(shape == ViewToggleIcon::SHAPE_LIST ? SystemIcon::VIEW_LIST : SystemIcon::VIEW_GRID);
}

bool ViewToggleIcon::hasIcon() const
{
    return !toggleIcon(shape).isNull();
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
    QRect area((width() - TOGGLE_ICON_SIDE) / 2, (height() - TOGGLE_ICON_SIDE) / 2, TOGGLE_ICON_SIDE,
               TOGGLE_ICON_SIDE);
    SystemIcon::drawCentred(&painter, area, toggleIcon(shape).normal);
    if (isActive) {
        int underlineTop = area.bottom() + TOGGLE_UNDERLINE_GAP;
        painter.fillRect(QRect(area.left(), underlineTop, area.width(), TOGGLE_UNDERLINE_HEIGHT), Qt::black);
    }
}

static SystemIcon::Id systemIconFor(HeaderIconButton::Icon icon)
{
    switch (icon) {
    case HeaderIconButton::ICON_IMPORT:
        return SystemIcon::IMPORT;
    case HeaderIconButton::ICON_SETTINGS:
        return SystemIcon::SETTINGS;
    case HeaderIconButton::ICON_REFRESH:
        return SystemIcon::REFRESH;
    case HeaderIconButton::ICON_ADD:
        break;
    }
    return SystemIcon::ADD;
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

static QFont headerLabelFont()
{
    return NickelFonts::font(NickelFonts::ROLE_BODY, HEADER_LABEL_SIZE, false);
}

// Nickel's pressed image when it has one; a Tabler icon is inverted in a
// circle; the label fallback is inverted as a whole, like Nickel's text buttons.
HeaderIconButton::HeaderIconButton(Icon iconShape, const QString &actionLabel, QWidget *parent)
    : PressArea(parent), icon(iconShape), label(actionLabel)
{
    const SystemIcon::Icon &image = SystemIcon::icon(systemIconFor(icon));
    if (!image.isNull()) {
        setFixedSize(HEADER_ICON_SIDE, HEADER_ICON_SIDE);
        setPressHighlight(image.pressed.isNull() ? HIGHLIGHT_CIRCLE : HIGHLIGHT_SELF_PAINTED);
        return;
    }
    int labelWidth = QFontMetrics(headerLabelFont()).width(label) + 2 * HEADER_LABEL_PADDING;
    setFixedSize(qMax(HEADER_ICON_SIDE, labelWidth), HEADER_ICON_SIDE);
    setPressHighlight(HIGHLIGHT_INVERT);
    DebugLog::write("header icon: no icon installed, showing the label \"%s\"", qPrintable(label));
}

void HeaderIconButton::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), Qt::white);
    const SystemIcon::Icon &image = SystemIcon::icon(systemIconFor(icon));
    if (image.isNull()) {
        painter.setFont(headerLabelFont());
        painter.setPen(Qt::black);
        painter.drawText(rect(), Qt::AlignCenter, label);
        return;
    }
    bool isPressedImage = isPressedStateShown() && !image.pressed.isNull();
    SystemIcon::drawCentred(&painter, rect(), isPressedImage ? image.pressed : image.normal);
}

AppMark::AppMark(const QString &iconPath, int side, QWidget *parent) : QWidget(parent)
{
    setFixedSize(side, side);
    setAttribute(Qt::WA_TransparentForMouseEvents, true);
    if (!iconPath.isEmpty()) {
        // QPixmap reads the whole file and closes it (SVG needs Nickel's svg
        // image plugin; without it the generic icon is shown instead).
        icon = QPixmap(iconPath);
    }
    if (icon.isNull()) {
        icon = SystemIcon::icon(SystemIcon::APP).normal;
    }
    if (!icon.isNull()) {
        icon = icon.scaled(side, side, Qt::KeepAspectRatio, Qt::SmoothTransformation);
    }
}

void AppMark::paintEvent(QPaintEvent *)
{
    if (icon.isNull()) {
        return;
    }
    QPainter painter(this);
    painter.drawPixmap((width() - icon.width()) / 2, (height() - icon.height()) / 2, icon);
}
