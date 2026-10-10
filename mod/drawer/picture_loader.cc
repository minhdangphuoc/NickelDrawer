#include "picture_loader.h"

#include <QBuffer>
#include <QByteArray>
#include <QFile>
#include <QImage>
#include <QImageReader>

#include "debug_log.h"

#include "screen_logic.h"

static bool readWhole(const QString &path, QByteArray *bytes)
{
    QFile file(path);
    if (!file.open(QIODevice::ReadOnly)) {
        DebugLog::write("picture: cannot open %s: %s", qPrintable(path), qPrintable(file.errorString()));
        return false;
    }
    if (file.size() > ScreenLogic::MAX_PICTURE_FILE_BYTES) {
        DebugLog::write("picture: %s is %lld bytes, over the %d byte limit; not decoded", qPrintable(path),
                        static_cast<long long>(file.size()), ScreenLogic::MAX_PICTURE_FILE_BYTES);
        return false;
    }
    *bytes = file.readAll();
    file.close();
    return !bytes->isEmpty();
}

namespace PictureLoader {

QPixmap load(const QString &path, const QSize &box)
{
    if (path.isEmpty() || box.isEmpty()) {
        return QPixmap();
    }
    QByteArray bytes;
    if (!readWhole(path, &bytes)) {
        return QPixmap();
    }
    QBuffer buffer(&bytes);
    buffer.open(QIODevice::ReadOnly);
    QImageReader reader(&buffer);
    QSize source = reader.size();
    if (!source.isValid()) {
        DebugLog::write("picture: %s is not an image Qt can read", qPrintable(path));
        return QPixmap();
    }
    int width = 0;
    int height = 0;
    ScreenLogic::fitInside(source.width(), source.height(), box.width(), box.height(), &width, &height);
    if (width != source.width() || height != source.height()) {
        reader.setScaledSize(QSize(width, height));
    }
    QImage image = reader.read();
    if (image.isNull()) {
        DebugLog::write("picture: decoding %s failed: %s", qPrintable(path), qPrintable(reader.errorString()));
        return QPixmap();
    }
    return QPixmap::fromImage(image);
}

} // namespace PictureLoader
