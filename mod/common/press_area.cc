#include "press_area.h"

#include <QGesture>
#include <QGestureEvent>
#include <QMouseEvent>
#include <QTimer>
#include <QTouchEvent>

#include "debug_log.h"

// Qt registers custom gesture types from Qt::CustomGesture (0x100) up.
static const int FIRST_CUSTOM_GESTURE_TYPE = Qt::CustomGesture;

static const char *const SOURCE_GESTURE = "nickel tap gesture";
static const char *const SOURCE_TOUCH = "touch";
static const char *const SOURCE_MOUSE = "mouse";

static PressArea::TapGestureTypeProvider tapGestureTypeProvider = NULL;
static bool hasLoggedGestureChoice = false;

void PressArea::setTapGestureTypeProvider(TapGestureTypeProvider provider)
{
    tapGestureTypeProvider = provider;
}

PressArea::PressArea(QWidget *parent)
    : QWidget(parent), longPressTimer(new QTimer(this)), tapGestureType(0), isPressed(false), hasFiredLongPress(false),
      acceptsPresses(true)
{
    // Qt 5.2's QElapsedTimer is not invalid when default-constructed; without this
    // it holds garbage and every first tap is dropped as "too soon" (seen on device).
    sinceLastTap.invalidate();
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

bool PressArea::event(QEvent *event)
{
    switch (event->type()) {
    case QEvent::Gesture:
        return handleGestureEvent(static_cast<QGestureEvent *>(event));
    case QEvent::GestureOverride:
        return handleGestureOverride(static_cast<QGestureEvent *>(event));
    case QEvent::TouchBegin:
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

void PressArea::mouseReleaseEvent(QMouseEvent *event)
{
    logEvent(SOURCE_MOUSE, "release", event->pos());
    endPress(SOURCE_MOUSE, rect().contains(event->pos()));
    event->accept();
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
        return;
    }
    sinceLastTap.start();
    DebugLog::write("press area %p (%s): tap via %s", static_cast<void *>(this), metaObject()->className(), source);
    QTimer::singleShot(0, this, SLOT(emitTapped()));
}

void PressArea::cancelPress(const char *source)
{
    if (isPressed) {
        DebugLog::write("press area %p: press canceled (%s)", static_cast<void *>(this), source);
    }
    isPressed = false;
    longPressTimer->stop();
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

void PressArea::emitTapped()
{
    emit tapped();
}

void PressArea::emitLongPressed()
{
    emit longPressed();
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
