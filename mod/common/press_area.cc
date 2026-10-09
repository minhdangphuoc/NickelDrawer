#include "press_area.h"

#include <QGesture>
#include <QGestureEvent>
#include <QMouseEvent>
#include <QPainter>
#include <QPainterPath>
#include <QPixmap>
#include <QPointer>
#include <QTimer>
#include <QTouchEvent>

#include "debug_log.h"

// Qt registers custom gesture types from Qt::CustomGesture (0x100) up.
static const int FIRST_CUSTOM_GESTURE_TYPE = Qt::CustomGesture;

// Kobo's patched Qt reads custom widget attributes as e-ink hints. Nickel's
// buttons push their pressed state to the panel with 0x80 set around a
// synchronous repaint() (IconButtonBase::tapGesture, AddToCollectionDialog::
// flashWidget; re/notes/G2-findings.md "Native header and press feedback").
static const Qt::WidgetAttribute REPAINT_NOW_ATTRIBUTE = static_cast<Qt::WidgetAttribute>(0x80);

// Icon buttons are 120 px tap targets around a ~44-64 px glyph; the circle
// leaves this margin inside the target, like Nickel's *_hit icons.
static const int CIRCLE_MARGIN = 16;
// A finger rolls a few pixels while pressing; only a real drag cancels.
static const int MOVE_TOLERANCE = 24;

static const char *const SOURCE_GESTURE = "nickel tap gesture";
static const char *const SOURCE_TOUCH = "touch";
static const char *const SOURCE_MOUSE = "mouse";

static PressArea::TapGestureTypeProvider tapGestureTypeProvider = NULL;
static bool hasLoggedGestureChoice = false;

// The pressed look of an inverted area: a picture laid over it, so child
// labels need no pressed style of their own. Input passes through to the area.
class PressOverlay : public QWidget {
public:
    explicit PressOverlay(QWidget *parent) : QWidget(parent)
    {
        setAttribute(Qt::WA_TransparentForMouseEvents, true);
        setFocusPolicy(Qt::NoFocus);
        hide();
    }

    void setImage(const QPixmap &pixmap) { image = pixmap; }

protected:
    void paintEvent(QPaintEvent *)
    {
        QPainter painter(this);
        painter.drawPixmap(0, 0, image);
    }

private:
    QPixmap image;
};

static const char *highlightName(PressArea::PressHighlight highlight)
{
    switch (highlight) {
    case PressArea::HIGHLIGHT_INVERT:
        return "inverted";
    case PressArea::HIGHLIGHT_CIRCLE:
        return "circle";
    case PressArea::HIGHLIGHT_SELF_PAINTED:
        return "self-painted";
    case PressArea::HIGHLIGHT_NONE:
        break;
    }
    return "none";
}

static void repaintNow(QWidget *widget, const QRect &area)
{
    widget->setAttribute(REPAINT_NOW_ATTRIBUTE, true);
    widget->repaint(area);
    widget->setAttribute(REPAINT_NOW_ATTRIBUTE, false);
}

// The area as drawn, colours inverted (Nickel: render, then Difference with
// white). White underneath, so a widget without its own background still
// inverts to black.
static QPixmap invertedSnapshot(QWidget *widget, const QRect &area)
{
    QPixmap snapshot(area.size());
    snapshot.fill(Qt::white);
    widget->render(&snapshot, QPoint(), QRegion(area), QWidget::DrawWindowBackground | QWidget::DrawChildren);
    QPainter painter(&snapshot);
    painter.setCompositionMode(QPainter::CompositionMode_Difference);
    painter.fillRect(snapshot.rect(), Qt::white);
    return snapshot;
}

// Only the centred circle of the inverted snapshot; outside it stays transparent.
static QPixmap circleOf(const QPixmap &inverted)
{
    int side = qMin(inverted.width(), inverted.height());
    int diameter = side > 3 * CIRCLE_MARGIN ? side - 2 * CIRCLE_MARGIN : side;
    QRect disc((inverted.width() - diameter) / 2, (inverted.height() - diameter) / 2, diameter, diameter);
    QPixmap circle(inverted.size());
    circle.fill(Qt::transparent);
    QPainter painter(&circle);
    QPainterPath clip;
    clip.addEllipse(disc);
    painter.setClipPath(clip);
    painter.drawPixmap(0, 0, inverted);
    return circle;
}

void PressArea::setTapGestureTypeProvider(TapGestureTypeProvider provider)
{
    tapGestureTypeProvider = provider;
}

PressArea::PressArea(QWidget *parent)
    : QWidget(parent), longPressTimer(new QTimer(this)), overlay(NULL), pressHighlight(HIGHLIGHT_INVERT),
      tapGestureType(0), isPressed(false), isPressedShown(false), hasFiredLongPress(false), acceptsPresses(true)
{
    // Qt 5.2's QElapsedTimer is not invalid when default-constructed; without this
    // it holds garbage and every first tap is dropped as "too soon" (seen on device).
    sinceLastTap.invalidate();
    pressedFeedback.highlight = HIGHLIGHT_NONE;
    setAttribute(Qt::WA_AcceptTouchEvents, true);
    setFocusPolicy(Qt::NoFocus); // page keys must reach the drawer view
    longPressTimer->setSingleShot(true);
    longPressTimer->setInterval(LONG_PRESS_MILLISECONDS);
    connect(longPressTimer, SIGNAL(timeout()), this, SLOT(onLongPressTimeout()));
    grabNickelTapGesture();
}

void PressArea::grabNickelTapGesture()
{
    int type = tapGestureTypeProvider ? tapGestureTypeProvider() : 0;
    bool isUsable = type >= FIRST_CUSTOM_GESTURE_TYPE;
    if (!hasLoggedGestureChoice) {
        hasLoggedGestureChoice = true;
        DebugLog::write("press area: Nickel tap gesture type 0x%x %s", type,
                        isUsable ? "grabbed" : "unusable, touch and mouse paths only");
    }
    if (isUsable) {
        tapGestureType = type;
        grabGesture(static_cast<Qt::GestureType>(type));
    }
}

void PressArea::setAcceptsPresses(bool isAccepting)
{
    acceptsPresses = isAccepting;
    if (!isAccepting) {
        cancelPress("input blocked");
    }
}

void PressArea::setPressHighlight(PressHighlight highlight)
{
    pressHighlight = highlight;
}

PressArea::PressFeedback PressArea::feedbackAt(const QPoint &) const
{
    PressFeedback feedback;
    feedback.rect = rect();
    feedback.highlight = pressHighlight;
    return feedback;
}

bool PressArea::event(QEvent *event)
{
    switch (event->type()) {
    case QEvent::Gesture:
        return handleGestureEvent(static_cast<QGestureEvent *>(event));
    case QEvent::GestureOverride:
        return handleGestureOverride(static_cast<QGestureEvent *>(event));
    case QEvent::TouchBegin:
    case QEvent::TouchUpdate:
    case QEvent::TouchEnd:
    case QEvent::TouchCancel:
        return handleTouchEvent(event);
    default:
        return QWidget::event(event);
    }
}

// An ancestor that grabbed the same gesture would otherwise get it first.
bool PressArea::handleGestureOverride(QGestureEvent *event)
{
    QGesture *gesture = tapGestureType ? event->gesture(static_cast<Qt::GestureType>(tapGestureType)) : NULL;
    if (!gesture) {
        return QWidget::event(event);
    }
    logEvent(SOURCE_GESTURE, "override", QPoint());
    event->accept(gesture);
    return true;
}

bool PressArea::handleGestureEvent(QGestureEvent *event)
{
    QGesture *gesture = tapGestureType ? event->gesture(static_cast<Qt::GestureType>(tapGestureType)) : NULL;
    if (!gesture) {
        return QWidget::event(event);
    }
    // TapGestureRecognizer sets the hot spot (screen coordinates).
    QPoint position = gesture->hasHotSpot() ? mapFromGlobal(gesture->hotSpot().toPoint()) : rect().center();
    if (!gesture->hasHotSpot()) {
        // Areas with many targets (app screens) then hit the centre: the device log must show it.
        DebugLog::write("press area %p: tap gesture without hot spot, using the centre", static_cast<void *>(this));
    }
    switch (gesture->state()) {
    case Qt::GestureStarted:
        logEvent(SOURCE_GESTURE, "started", position);
        beginPress(SOURCE_GESTURE, position);
        break;
    case Qt::GestureFinished:
        logEvent(SOURCE_GESTURE, "finished", position);
        endPress(SOURCE_GESTURE, !gesture->hasHotSpot() || rect().contains(position));
        break;
    case Qt::GestureCanceled:
        logEvent(SOURCE_GESTURE, "canceled", position);
        cancelPress(SOURCE_GESTURE);
        break;
    case Qt::GestureUpdated:
    case Qt::NoGesture:
        break;
    }
    event->accept(gesture);
    return true;
}

bool PressArea::handleTouchEvent(QEvent *event)
{
    QTouchEvent *touch = static_cast<QTouchEvent *>(event);
    bool hasPoint = !touch->touchPoints().isEmpty();
    QPoint position = hasPoint ? touch->touchPoints().first().pos().toPoint() : rect().center();
    switch (event->type()) {
    case QEvent::TouchBegin:
        logEvent(SOURCE_TOUCH, "begin", position);
        beginPress(SOURCE_TOUCH, position);
        break;
    case QEvent::TouchUpdate:
        movePress(SOURCE_TOUCH, position); // not logged: one line per finger movement
        break;
    case QEvent::TouchEnd:
        logEvent(SOURCE_TOUCH, "end", position);
        endPress(SOURCE_TOUCH, !hasPoint || rect().contains(position));
        break;
    default:
        logEvent(SOURCE_TOUCH, "cancel", position);
        cancelPress(SOURCE_TOUCH);
        break;
    }
    event->accept();
    return true;
}

void PressArea::mousePressEvent(QMouseEvent *event)
{
    logEvent(SOURCE_MOUSE, "press", event->pos());
    beginPress(SOURCE_MOUSE, event->pos());
    event->accept();
}

void PressArea::mouseMoveEvent(QMouseEvent *event)
{
    movePress(SOURCE_MOUSE, event->pos());
    event->accept();
}

void PressArea::mouseReleaseEvent(QMouseEvent *event)
{
    logEvent(SOURCE_MOUSE, "release", event->pos());
    endPress(SOURCE_MOUSE, rect().contains(event->pos()));
    event->accept();
}

void PressArea::hideEvent(QHideEvent *event)
{
    clearPressedState("area hidden");
    QWidget::hideEvent(event);
}

void PressArea::beginPress(const char *source, const QPoint &position)
{
    if (!acceptsPresses || isPressed) {
        return;
    }
    if (isWithinDebounce()) {
        DebugLog::write("press area %p: %s press dropped (%lld ms after the last tap)", static_cast<void *>(this),
                        source, static_cast<long long>(sinceLastTap.elapsed()));
        return;
    }
    lastPressPosition = position;
    isPressed = true;
    hasFiredLongPress = false;
    longPressTimer->start();
    DebugLog::write("press area %p (%s): pressed via %s at %d,%d", static_cast<void *>(this),
                    metaObject()->className(), source, position.x(), position.y());
    showPressedState(source);
}

void PressArea::movePress(const char *source, const QPoint &position)
{
    if (!isPressed) {
        return;
    }
    QRect bounds = pressedFeedback.rect.isEmpty() ? rect() : pressedFeedback.rect;
    if (!bounds.adjusted(-MOVE_TOLERANCE, -MOVE_TOLERANCE, MOVE_TOLERANCE, MOVE_TOLERANCE).contains(position)) {
        DebugLog::write("press area %p: %s moved out to %d,%d", static_cast<void *>(this), source, position.x(),
                        position.y());
        cancelPress(source);
    }
}

void PressArea::endPress(const char *source, bool isInside)
{
    if (!isPressed) {
        return;
    }
    bool isTap = isInside && !hasFiredLongPress;
    isPressed = false;
    longPressTimer->stop();
    if (!isTap) {
        DebugLog::write("press area %p: released via %s, %s", static_cast<void *>(this), source,
                        hasFiredLongPress ? "after the long press" : "outside, no tap");
        clearPressedState(hasFiredLongPress ? "released after long press" : "released outside");
        return;
    }
    sinceLastTap.start();
    DebugLog::write("press area %p (%s): tap via %s", static_cast<void *>(this), metaObject()->className(), source);
    QTimer::singleShot(0, this, SLOT(emitTapped())); // the pressed state stays until the action ran
}

void PressArea::cancelPress(const char *source)
{
    if (isPressed) {
        DebugLog::write("press area %p: press canceled (%s)", static_cast<void *>(this), source);
    }
    isPressed = false;
    longPressTimer->stop();
    clearPressedState(source);
}

void PressArea::showPressedState(const char *source)
{
    PressFeedback feedback = feedbackAt(lastPressPosition);
    feedback.rect &= rect();
    if (feedback.highlight == HIGHLIGHT_NONE || feedback.rect.isEmpty()) {
        DebugLog::write("press area %p: no pressed state at %d,%d", static_cast<void *>(this), lastPressPosition.x(),
                        lastPressPosition.y());
        return;
    }
    pressedFeedback = feedback;
    isPressedShown = true;
    if (feedback.highlight == HIGHLIGHT_SELF_PAINTED) {
        repaintNow(this, feedback.rect);
    } else {
        showOverlay(feedback);
    }
    DebugLog::write("press area %p (%s): pressed state shown (%s, %dx%d at %d,%d, repaint now, via %s)",
                    static_cast<void *>(this), metaObject()->className(), highlightName(feedback.highlight),
                    feedback.rect.width(), feedback.rect.height(), feedback.rect.x(), feedback.rect.y(), source);
}

void PressArea::showOverlay(const PressFeedback &feedback)
{
    if (!overlay) {
        overlay = new PressOverlay(this);
    }
    overlay->hide(); // never part of its own snapshot
    QPixmap inverted = invertedSnapshot(this, feedback.rect);
    overlay->setImage(feedback.highlight == HIGHLIGHT_CIRCLE ? circleOf(inverted) : inverted);
    overlay->setGeometry(feedback.rect);
    overlay->show();
    overlay->raise();
    repaintNow(overlay, overlay->rect());
}

// Back to the normal look exactly as Nickel's defaultColors() does it
// (IconButtonBase, TextHeaderBase): no e-ink hint, one plain update() of the
// pressed rect, so Kobo's Qt redraws it with its normal partial waveform and
// the fast press waveform leaves no ghost. Hiding the overlay alone would
// leave the repaint of the area under it to Qt's exposure handling.
void PressArea::clearPressedState(const char *reason)
{
    if (!isPressedShown) {
        return;
    }
    isPressedShown = false;
    if (overlay && overlay->isVisible()) {
        overlay->hide();
    }
    update(pressedFeedback.rect);
    DebugLog::write("press area %p (%s): pressed state cleared (%s), %dx%d at %d,%d redrawn with a plain update",
                    static_cast<void *>(this), metaObject()->className(), reason, pressedFeedback.rect.width(),
                    pressedFeedback.rect.height(), pressedFeedback.rect.x(), pressedFeedback.rect.y());
    pressedFeedback.rect = QRect();
    pressedFeedback.highlight = HIGHLIGHT_NONE;
}

void PressArea::onLongPressTimeout()
{
    if (!isPressed || !acceptsPresses) {
        return;
    }
    hasFiredLongPress = true;
    DebugLog::write("press area %p (%s): long press", static_cast<void *>(this), metaObject()->className());
    QTimer::singleShot(0, this, SLOT(emitLongPressed()));
}

// The slot may delete this area (a tap that pops the page): only a live area
// clears its pressed state.
void PressArea::emitTapped()
{
    QPointer<PressArea> alive(this);
    emit tapped();
    if (alive) {
        clearPressedState("after tap action");
    }
}

void PressArea::emitLongPressed()
{
    QPointer<PressArea> alive(this);
    emit longPressed();
    if (alive) {
        clearPressedState("after long-press action");
    }
}

void PressArea::logEvent(const char *source, const char *phase, const QPoint &position) const
{
    DebugLog::write("press area %p (%s \"%s\"): %s %s at %d,%d%s", static_cast<const void *>(this),
                    metaObject()->className(), qPrintable(objectName()), source, phase, position.x(), position.y(),
                    acceptsPresses ? "" : " (input blocked)");
}

bool PressArea::isWithinDebounce() const
{
    if (!sinceLastTap.isValid())
        return false;
    const qint64 elapsed = sinceLastTap.elapsed();
    return elapsed >= 0 && elapsed < DEBOUNCE_MILLISECONDS; // a negative value means a broken clock, never "too soon"
}
