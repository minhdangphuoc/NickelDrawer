#ifndef NICKELDRAWER_MORE_ROW_H
#define NICKELDRAWER_MORE_ROW_H

#include <QString>

class QWidget;

// One row of ours in Nickel's More tab ("Thêm"), built as Nickel builds its
// own (re/notes/G2-findings.md Q1): a Nickel IconLeftButton with a 50x50 icon
// and a title, inserted in MoreView's "moreContainerLayout". Shared by the
// Apps row (EntryRow) and the Tweaks row (TweaksRow).
namespace MoreRow {

struct Spec {
    const char *objectName;   // e.g. "nickelDrawerButton"
    const char *iconPath;     // our icon on the user partition (installed by koboapps)
    const char *fallbackIcon; // Nickel resource used when ours is missing
    const char *anchorName;   // the row goes right after this row; missing: appended
};

// Builds and inserts the row; its tapped() signal is checked. NULL (logged)
// when IconLeftButton, the layout or the signal is missing: the More tab then
// stays as Nickel built it.
QWidget *insert(QWidget *moreView, const Spec &spec, const QString &title);

} // namespace MoreRow

#endif
