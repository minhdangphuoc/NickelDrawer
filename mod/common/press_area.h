#ifndef NICKELDRAWER_PRESS_AREA_H
#define NICKELDRAWER_PRESS_AREA_H

#include <QElapsedTimer>
#include <QWidget>

class QGestureEvent;
class QTimer;

// The one tappable widget of both mods: reports a short tap or a long press
// (~600 ms, docs/spec/ui-kit.md §5.4). Shared by mod/drawer-proto and mod/drawer.
//
// On the device a plain QPushButton never saw clicked(): Nickel does not
// synthesize mouse events from touch. Nickel's own buttons (IconButtonBase,
// TouchLabel) grabGesture() the type TapGestureRecognizer registered and get
// QEvent::Gesture (TapGesture: started on finger down, finished on release,
// canceled on drag). So a press area listens on three paths at once, whichever
// Nickel delivers: Nickel's tap gesture, raw touch events and mouse events.
// One physical tap can arrive on several paths; a press is begun and ended
// once and a second tap within the debounce interval is dropped. Every event
// received is written to the debug log, so a device test shows the path used.
//
// Signals are emitted from the event loop (0 ms timer), never from inside the
// gesture or touch handler: the slot may pop and delete the page.
class PressArea : public QWidget {
    Q_OBJECT
public:
    // Returns the gesture type Nickel's TapGestureRecognizer registered, or 0
    // when unknown. Each mod sets one at init (its libnickel symbol lookup);
    // it is read when a press area is constructed, after Nickel registered it.
    typedef int (*TapGestureTypeProvider)();
    static void setTapGestureTypeProvider(TapGestureTypeProvider provider);

    static const int LONG_PRESS_MILLISECONDS = 600;
    static const int DEBOUNCE_MILLISECONDS = 400;

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
    void emitTapped();
    void emitLongPressed();

private:
    void grabNickelTapGesture();
    bool handleGestureEvent(QGestureEvent *event);
    bool handleGestureOverride(QGestureEvent *event);
    bool handleTouchEvent(QEvent *event);
    void beginPress(const char *source, const QPoint &position);
    void endPress(const char *source, bool isInside);
    void cancelPress(const char *source);
    void logEvent(const char *source, const char *phase, const QPoint &position) const;

    QTimer *longPressTimer;
    QElapsedTimer sinceLastTap;
    QPoint lastPressPosition;
    int tapGestureType; // 0: Nickel's tap gesture not grabbed
    bool isPressed;
    bool hasFiredLongPress;
    bool acceptsPresses;
};

#endif
