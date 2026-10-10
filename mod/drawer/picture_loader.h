#ifndef NICKELDRAWER_PICTURE_LOADER_H
#define NICKELDRAWER_PICTURE_LOADER_H

#include <QPixmap>
#include <QSize>
#include <QString>

// Reads one picture for the gallery or the viewer. The file is read whole and
// closed at once (nothing stays open on /mnt/onboard), then decoded from
// memory scaled to fit the box, so a big file never exists decoded at full
// size next to its scaled copy. Files over ScreenLogic::MAX_PICTURE_FILE_BYTES
// are refused. No QtSvg.
namespace PictureLoader {

// A null pixmap when the file is missing, too big or not an image; the reason
// goes to the debug log.
QPixmap load(const QString &path, const QSize &box);

} // namespace PictureLoader

#endif
