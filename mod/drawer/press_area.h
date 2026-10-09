#ifndef NICKELDRAWER_PRESS_AREA_H
#define NICKELDRAWER_PRESS_AREA_H

#include <QWidget>

class QTimer;

// A plain Qt widget that reports a short tap or a long press (~600 ms,
// docs/spec/ui-kit.md §5.4). Accepts both touch events and the mouse events
// Qt synthesizes from touch, because which one Nickel's input delivers to a
// plain widget is not verified yet.
class PressArea : public QWidget {
    Q_OBJECT
public:
    explicit PressArea(QWidget *parent);

    // While disabled for input (dialog open, action pending), presses are ignored.
    void setAcceptsPresses(bool isAccepting);
    // Where the last press started, in widget coordinates (for areas with many targets).
    QPoint pressPosition() const { return lastPressPosition; }

signals:
    void tapped();
    void longPressed();

protected:
    bool event(QEvent *event);
    void mousePressEvent(QMouseEvent *event);
    void mouseReleaseEvent(QMouseEvent *event);

private slots:
    void onLongPressTimeout();

private:
    void beginPress(const QPoint &position);
    void endPress(bool isInside);
    void cancelPress();

    QTimer *longPressTimer;
    QPoint lastPressPosition;
    bool isPressed;
    bool hasFiredLongPress;
    bool acceptsPresses;
};

#endif
