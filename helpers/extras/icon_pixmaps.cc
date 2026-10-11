#include "icon_pixmaps.h"

#include <QImage>
#include <QPainter>

#include <NickelHook.h>

#include "debug_log.h"
#include "helper_flags.h"
#include "icon_metrics.h"

static const char GLYPH_FILE[] = "/screenshot.png";

static QPixmap whiteInk(const QPixmap &image)
{
    QImage inverted = image.toImage().convertToFormat(QImage::Format_ARGB32);
    inverted.invertPixels(QImage::InvertRgb);
    return QPixmap::fromImage(inverted);
}

static QPixmap transparentSquare(int side)
{
    QPixmap square(side, side);
    square.fill(Qt::transparent);
    return square;
}

static QPixmap onBlackDisc(const QPixmap &glyph, int side)
{
    QPixmap result = transparentSquare(side);
    QPainter painter(&result);
    painter.setRenderHint(QPainter::Antialiasing, true);
    painter.setPen(Qt::NoPen);
    painter.setBrush(Qt::black);
    painter.drawEllipse(QRect(0, 0, side, side));
    painter.drawPixmap(0, 0, glyph);
    return result;
}

static QPixmap atSide(const QPixmap &image, int side)
{
    if (image.width() == side && image.height() == side) {
        return image;
    }
    DebugLog::write("icon: glyph %dx%d scaled to %d px (Nickel's icon size)", image.width(), image.height(), side);
    return image.scaled(side, side, Qt::KeepAspectRatio, Qt::SmoothTransformation);
}

namespace IconPixmaps {

Set load(int nickelSide)
{
    Set set;
    QString path = QString::fromLatin1(HelperFlags::ICONS_DIRECTORY) + QString::fromLatin1(GLYPH_FILE);
    QPixmap glyph(path); // QPixmap reads the file whole and closes it
    if (glyph.isNull()) {
        nh_log("icon: %s missing or unreadable, no button (reinstall the helper)", qPrintable(path));
        return set;
    }
    int side = IconMetrics::canvasSide(nickelSide, qMin(glyph.width(), glyph.height()));
    QPixmap normal = atSide(glyph, side);
    set.normal = normal;
    set.pressed = onBlackDisc(whiteInk(normal), side);
    set.blank = transparentSquare(side);
    DebugLog::write("icon: glyph %dx%d, images %d px (Nickel's side %d)", glyph.width(), glyph.height(), side,
                    nickelSide);
    return set;
}

} // namespace IconPixmaps
