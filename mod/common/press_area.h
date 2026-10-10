#ifndef NICKELDRAWER_PRESS_AREA_H
#define NICKELDRAWER_PRESS_AREA_H

#include <QElapsedTimer>
#include <QRect>
#include <QWidget>

class PressOverlay;
class QGesture;
class QGestureEvent;
class QTimer;

// The one tappable widget of both mods: reports a short tap or a long press
// (~600 ms, docs/spec/ui-kit.md §5.4). Shared by mod/drawer-proto and mod/drawer.
//
// On the device a plain QPushButton never saw clicked(): Nickel does not
// synthesize mouse events from touch. Nickel's own buttons (IconButtonBase,
// TouchLabel) grabGesture() the type TapGestureRecognizer registered and get
// QEvent::Gesture (TapGesture: started on finger down, finished on release,
// canceled on drag). Nickel never sets the gesture's hot spot; the
// finger's point is TapGesture::position() (screen coordinates), which areas
// with many targets (app screens) need: the centre would hit the wrong target. So a press area listens on three paths at once, whichever
// Nickel delivers: Nickel's tap gesture, raw touch events and mouse events.
// One physical tap can arrive on several paths; a press is begun and ended
// once and a second tap within the debounce interval is dropped. Every event
// received is written to the debug log, so a device test shows the path used.
//
// Press feedback copies Nickel's buttons (IconButtonBase::tapGesture): on
// finger down the pressed state is shown at once (the area inverted, or a
// black circle behind an icon) and pushed to the screen with Kobo's
// "repaint now" hint; on release the action runs, then the normal look comes
// back with a plain update. A long press keeps the pressed state until its
// handler returned (the menu is open); a drag out of the area, a cancel or
// blocked input clears it.
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
    // Where Nickel's tap gesture started, in screen coordinates: Nickel never
    // sets the QGesture hot spot, it keeps the point in TapGesture::position().
    // Returns false when unknown (symbol missing, not a TapGesture). Without a
    // provider, an area with many targets takes the point of the touch begin
    // that follows the gesture (the recognizer sees the touch first).
    typedef bool (*TapPositionProvider)(QGesture *gesture, QPoint *screenPosition);
    static void setTapPositionProvider(TapPositionProvider provider);

    static const int LONG_PRESS_MILLISECONDS = 600;
    static const int DEBOUNCE_MILLISECONDS = 400;

    enum PressHighlight {
        HIGHLIGHT_INVERT,       // rows, tiles, dialog buttons: the area inverted (Nickel's More rows)
        HIGHLIGHT_CIRCLE,       // icon buttons: a black circle with the icon in white (Nickel's *_hit icons)
        HIGHLIGHT_SELF_PAINTED, // the subclass paints it: isPressedStateShown() in paintEvent
        HIGHLIGHT_NONE,
    };
    struct PressFeedback {
        QRect rect; // widget coordinates; empty: no feedback
        PressHighlight highlight;
    };

    explicit PressArea(QWidget *parent);

    // While disabled for input (dialog open, action pending), presses are ignored.
    void setAcceptsPresses(bool isAccepting);
    // Where the last press started, in widget coordinates (for areas with many targets).
    QPoint pressPosition() const { return lastPressPosition; }
    // How a press on the whole area is shown (default HIGHLIGHT_INVERT).
    void setPressHighlight(PressHighlight highlight);
    bool isPressedStateShown() const { return isPressedShown; }
    QRect pressedStateRect() const { return pressedFeedback.rect; }
    bool hasNickelTapGesture() const { return tapGestureType != 0; }
    // One debug line: class, object name, geometry (local and global),
    // visibility, gesture grabbed, touch accepted, input blocked.
    void logTapTarget(const char *when) const;

signals:
    void tapped();
    void longPressed();

protected:
    // What to highlight for a press at position; areas with many targets
    // (app screens) return the target's rect. Default: the whole area.
    virtual PressFeedback feedbackAt(const QPoint &position) const;

    bool event(QEvent *event);
    void mousePressEvent(QMouseEvent *event);
    void mouseMoveEvent(QMouseEvent *event);
    void mouseReleaseEvent(QMouseEvent *event);
    void hideEvent(QHideEvent *event);

private slots:
    void onLongPressTimeout();
    void emitTapped();
    void emitLongPressed();

private:
    void grabNickelTapGesture();
    bool handleGestureEvent(QGestureEvent *event);
    bool handleGestureOverride(QGestureEvent *event);
    bool handleTouchEvent(QEvent *event);
    QPoint gesturePosition(QGesture *gesture, const char **origin) const;
    void refineGuessedPress(const QPoint &position);
    void beginPress(const char *source, const QPoint &position);
    void movePress(const char *source, const QPoint &position);
    void endPress(const char *source, bool isInside);
    void cancelPress(const char *source);
    void showPressedState(const char *source);
    void showOverlay(const PressFeedback &feedback);
    void clearPressedState(const char *reason);
    void logEvent(const char *source, const char *phase, const QPoint &position) const;

    QTimer *longPressTimer;
    PressOverlay *overlay; // created on the first inverted press
    bool isWithinDebounce() const;
    QElapsedTimer sinceLastTap;
    QPoint lastPressPosition;
    PressFeedback pressedFeedback;
    PressHighlight pressHighlight;
    int tapGestureType; // 0: Nickel's tap gesture not grabbed
    bool isPressed;
    bool isPressPositionGuessed; // begun by a gesture with no known point: the centre
    bool isPressedShown;
    bool hasFiredLongPress;
    bool acceptsPresses;
};

#endif
