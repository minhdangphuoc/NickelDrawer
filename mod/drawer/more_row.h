#ifndef NICKELDRAWER_MORE_ROW_H
#define NICKELDRAWER_MORE_ROW_H

#include <QString>

class QWidget;

// One row of ours in Nickel's More tab ("Thêm"), built as Nickel builds its
// own (re/notes/G2-findings.md Q1): a Nickel IconLeftButton with a 50x50 icon
// and a title, inserted in MoreView's "moreContainerLayout". Shared by the
// Apps row (EntryRow) and the Tweaks row (TweaksRow).
namespace MoreRow {

// The icon is Nickel's own when it has a fitting one, else a Tabler PNG that
// koboapps installs, else none (the row shows its title only). We never draw it.
struct Spec {
    const char *objectName; // e.g. "nickelDrawerButton"
    const char *nickelIcon; // Nickel resource (as its own More rows use), or NULL
    const char *ownIcon;    // Tabler PNG on the user partition, scaled to Nickel's row icon size
    const char *anchorName; // the row goes right after this row; missing: appended
};

// Builds and inserts the row; its tapped() signal is checked. NULL (logged)
// when IconLeftButton, the layout or the signal is missing: the More tab then
// stays as Nickel built it.
QWidget *insert(QWidget *moreView, const Spec &spec, const QString &title);

} // namespace MoreRow

#endif
