#ifndef NICKELDRAWER_DRAWER_SURFACES_H
#define NICKELDRAWER_DRAWER_SURFACES_H

#include <QString>
#include <QStringList>
#include <QWidget>

#include "press_area.h"

// The drawer's own CopperUI surfaces (docs/design-system.md 4.7 and 4.18), painted with text that is
// measured and wrapped by the shared text-fit rule, so nothing clips or overflows.

// A notice: 2 px black border, paper.notice fill, 24 px padding, body text wrapped, then one rhythm
// of space below. The Qt6 upgrade warning.
class NoticeBox : public QWidget {
public:
    NoticeBox(const QString &text, QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);

private:
    QStringList lines;
    int boxHeight;
};

// The standard primary button: content width, black fill, white bold label of at most two centred
// lines, 120 px tall (168 for two lines). Pressed: inverted by PressArea.
class PrimaryButton : public PressArea {
public:
    PrimaryButton(const QString &label, QWidget *parent);

    static QStringList labelLines(const QString &label);

protected:
    void paintEvent(QPaintEvent *event);

private:
    QStringList lines;
};

// The empty state: optional 96 px icon, serif heading, muted caption and at most one primary
// button, stacked by copperui/empty_state.h. "Loading…" is the heading alone.
class EmptyState : public QWidget {
    Q_OBJECT
public:
    // caption empty: none; buttonLabel empty: no button; hasIcon false (or the icon not installed): none.
    EmptyState(const QString &heading, const QString &caption, bool hasIcon, const QString &buttonLabel, QWidget *parent);

    void setInputEnabled(bool isEnabled);

signals:
    void buttonTapped();

protected:
    void paintEvent(QPaintEvent *event);

private:
    QStringList headingLines;
    QStringList captionLines;
    QRect iconRect;
    QRect headingRect;
    QRect captionRect;
    PrimaryButton *button;
};

#endif
