#ifndef NICKELDRAWER_TWEAKS_PANEL_H
#define NICKELDRAWER_TWEAKS_PANEL_H

#include <QList>
#include <QStringList>
#include <QWidget>

#include "press_area.h"
#include "tweaks_logic.h"
#include "tweaks_model.h"

class QLabel;

// One tweak, painted with measured text in Nickel's sans (cell geometry: tweaks_logic.h): an
// action is its label, centred; a toggle its label at the left and, at the right, Nickel's switch
// image with the state word under it; an info item its label and, under it, the value as a
// caption. Toggles and actions invert while pressed; info cells take no presses.
class TweakCell : public PressArea {
public:
    TweakCell(const TweakItem &item, QWidget *parent);

    const QString &tweakId() const { return shown.id; }
    // Changes only the value (state or info text), in place (a partial e-ink update).
    void showValue(const TweakItem &item);

protected:
    void paintEvent(QPaintEvent *event);
    void resizeEvent(QResizeEvent *event);

private:
    void measureText();

    TweakItem shown;
    QStringList labelLines;
    QString valueLine;
    QString stateWord;
    TweaksLogic::CellLayout cellGeometry;
};

// The body of the Tweaks modal (docs/spec/tweaks.md, DECISIONS 2026-10-10):
// a full-width rule, then the tweaks in 2 columns, row by row, with thin
// dividers between rows and columns; page keys flip pages when there are
// more than TweaksLogic::ROWS_PER_PAGE rows. Grid math: TweaksLogic.
// The chrome (title, close button, frame) is Nickel's N3Dialog, or the own
// overlay when that is missing.
class TweaksPanel : public QWidget {
    Q_OBJECT
public:
    explicit TweaksPanel(QWidget *parent);

    // "Loading…" or why the list could not be read.
    void showMessage(const QString &text);
    // Builds the cells of the first page. Never called from a tap handler.
    void showItems(const QList<TweakItem> &items);
    // A toggle's new value, in place.
    void updateItem(const TweakItem &item);
    const TweakItem *item(const QString &tweakId) const;
    // While a tweak runs, every cell ignores presses.
    void setInputEnabled(bool isEnabled);
    // Page keys (Nickel's TactileButtonFilter: Left/Up back, Right/Down on).
    // true when the key was a page key.
    bool handlePageKey(int qtKey);

signals:
    void tweakTapped(const QString &tweakId);

protected:
    void keyPressEvent(QKeyEvent *event);
    bool eventFilter(QObject *watched, QEvent *event);

private slots:
    void onCellTapped();

private:
    void showPage(int page);
    void placeCells();
    void updatePageLabel();
    int itemsPerPage() const;

    QWidget *grid;
    QLabel *messageLabel;
    QLabel *pageLabel;
    QList<TweakItem> allItems;
    QList<TweakCell *> cells; // the current page
    int currentPage;
    bool isInputEnabled;
};

#endif
