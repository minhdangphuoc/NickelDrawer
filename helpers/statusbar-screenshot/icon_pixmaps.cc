#include "icon_pixmaps.h"

#include <QImage>
#include <QPainter>

#include <NickelHook.h>

#include "helper_flags.h"
#include "helper_log.h"

static const char GLYPH_FILE[] = "/icons/screenshot-64.png";
// The glyph fills this share of the image, so its farthest corner still sits inside the pressed disc
// (the Tabler glyph reaches 21/24 of its box, a corner is at 0.53 * box from the centre).
static const double GLYPH_SHARE = 0.8;

static QPixmap whiteInk(const QPixmap &image)
{
    QImage inverted = image.toImage().convertToFormat(QImage::Format_ARGB32);
    inverted.invertPixels(QImage::InvertRgb);
    return QPixmap::fromImage(inverted);
}

static QPixmap transparentSquare(int size)
{
    QPixmap square(size, size);
    square.fill(Qt::transparent);
    return square;
}

static QPixmap withGlyph(const QPixmap &glyph, bool isOnBlackDisc, int size)
{
    QPixmap result = transparentSquare(size);
    QPainter painter(&result);
    painter.setRenderHint(QPainter::Antialiasing, true);
    painter.setRenderHint(QPainter::SmoothPixmapTransform, true);
    if (isOnBlackDisc) {
        painter.setPen(Qt::NoPen);
        painter.setBrush(Qt::black);
        painter.drawEllipse(QRect(0, 0, size, size));
    }
    int glyphSize = static_cast<int>(size * GLYPH_SHARE);
    int offset = (size - glyphSize) / 2;
    painter.drawPixmap(QRect(offset, offset, glyphSize, glyphSize), glyph);
    return result;
}

namespace IconPixmaps {

Set load(int size)
{
    Set set;
    QString path = QString::fromLatin1(HelperFlags::FOLDER) + QString::fromLatin1(GLYPH_FILE);
    QPixmap glyph(path); // QPixmap reads the file whole and closes it
    if (glyph.isNull() || size <= 0) {
        nh_log("icon: %s missing or unreadable, no button (reinstall the helper)", qPrintable(path));
        return set;
    }
    set.normal = withGlyph(glyph, false, size);
    set.pressed = withGlyph(whiteInk(glyph), true, size);
    set.blank = transparentSquare(size);
    HelperLog::debug("icon: glyph %dx%d drawn at %d px", glyph.width(), glyph.height(), size);
    return set;
}

} // namespace IconPixmaps
