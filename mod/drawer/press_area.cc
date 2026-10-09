#include "press_area.h"

#include <QMouseEvent>
#include <QTimer>
#include <QTouchEvent>

#include "drawer_logic.h"

PressArea::PressArea(QWidget *parent)
    : QWidget(parent), longPressTimer(new QTimer(this)), isPressed(false), hasFiredLongPress(false),
      acceptsPresses(true)
{
    setAttribute(Qt::WA_AcceptTouchEvents, true);
    setFocusPolicy(Qt::NoFocus); // page keys must reach the drawer view
    longPressTimer->setSingleShot(true);
    longPressTimer->setInterval(DrawerLogic::LONG_PRESS_MILLISECONDS);
    connect(longPressTimer, SIGNAL(timeout()), this, SLOT(onLongPressTimeout()));
}

void PressArea::setAcceptsPresses(bool isAccepting)
{
    acceptsPresses = isAccepting;
    if (!isAccepting) {
        cancelPress();
    }
}

bool PressArea::event(QEvent *event)
{
    switch (event->type()) {
    case QEvent::TouchBegin: {
        QTouchEvent *touch = static_cast<QTouchEvent *>(event);
        beginPress(touch->touchPoints().isEmpty() ? QPoint() : touch->touchPoints().first().pos().toPoint());
        event->accept();
        return true;
    }
    case QEvent::TouchEnd: {
        QTouchEvent *touch = static_cast<QTouchEvent *>(event);
        bool isInside = touch->touchPoints().isEmpty() || rect().contains(touch->touchPoints().first().pos().toPoint());
        endPress(isInside);
        event->accept();
        return true;
    }
    case QEvent::TouchCancel:
        cancelPress();
        return true;
    default:
        return QWidget::event(event);
    }
}

void PressArea::mousePressEvent(QMouseEvent *event)
{
    beginPress(event->pos());
    event->accept();
}

void PressArea::mouseReleaseEvent(QMouseEvent *event)
{
    endPress(rect().contains(event->pos()));
    event->accept();
}

void PressArea::beginPress(const QPoint &position)
{
    if (!acceptsPresses || isPressed) {
        return;
    }
    lastPressPosition = position;
    isPressed = true;
    hasFiredLongPress = false;
    longPressTimer->start();
}

void PressArea::endPress(bool isInside)
{
    if (!isPressed) {
        return;
    }
    bool isTap = isInside && !hasFiredLongPress;
    cancelPress();
    if (isTap) {
        emit tapped();
    }
}

void PressArea::cancelPress()
{
    isPressed = false;
    longPressTimer->stop();
}

void PressArea::onLongPressTimeout()
{
    if (!isPressed || !acceptsPresses) {
        return;
    }
    hasFiredLongPress = true;
    emit longPressed();
}
