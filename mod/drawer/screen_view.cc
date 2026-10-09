#include "screen_view.h"

#include <QEvent>
#include <QFontMetrics>
#include <QKeyEvent>
#include <QPaintEvent>
#include <QPainter>
#include <QTimer>

#include <NickelHook.h>

#include "fallback_strings.h"
#include "nickel_bridge.h"
#include "screen_fonts.h"

using namespace ScreenLogic;

static const int RUN_FROM_EVENT_LOOP = 0;
static const int ONE_PAGE_PER_STEP = 1;

static QColor grey(int level)
{
    return QColor(level, level, level);
}

static HostTexts hostTexts()
{
    HostTexts texts;
    texts.on = FallbackStrings::text(FallbackStrings::SCREEN_ON);
    texts.off = FallbackStrings::text(FallbackStrings::SCREEN_OFF);
    return texts;
}

static int qtAlignment(PaintItem::Align align)
{
    switch (align) {
    case PaintItem::ALIGN_CENTER:
        return Qt::AlignHCenter;
    case PaintItem::ALIGN_END:
        return Qt::AlignRight;
    case PaintItem::ALIGN_START:
        break;
    }
    return Qt::AlignLeft;
}

ScreenView::ScreenView(QWidget *drawerView)
    : PressArea(drawerView), host(drawerView), hasScreen(false), pageIndex(0), isInputBlocked(false),
      isTapPending(false)
{
    DrawerLogic::resetRefreshPolicy(&refreshPolicy);
    setObjectName(QStringLiteral("nickelDrawerScreen"));
    setAttribute(Qt::WA_OpaquePaintEvent, true); // paintEvent covers every pixel
    setGeometry(host->rect());
    host->installEventFilter(this);
    connect(this, SIGNAL(tapped()), this, SLOT(onTapped()));
    sinceLastTap.invalidate();
    raise();
    show();
}

const ScreenComponent *ScreenView::component(int index) const
{
    return index >= 0 && index < layout.componentCount() ? &layout.component(index) : NULL;
}

void ScreenView::showScreen(const ScreenModel &next)
{
    bool isSameScreen = hasScreen && next.app == model.app && next.screen == model.screen;
    model = next;
    hasScreen = true;
    layout = ScreenLayout(model, height(), hostTexts());
    if (!isSameScreen && model.screen.isEmpty()) {
        // A placeholder (no screen id: loading) is drawn without the flash.
        pageIndex = 0;
        page = layout.page(0);
        update();
        return;
    }
    if (!isSameScreen) {
        pageIndex = 0;
        applyPage(0, true);
        return;
    }
    pageIndex = qBound(0, pageIndex, layout.pageCount() - 1);
    applyPage(pageIndex, false);
}

static ScreenComponent *componentIn(ScreenModel *model, int index)
{
    if (index >= 0 && index < model->body.size()) {
        return &model->body[index];
    }
    index -= model->body.size();
    return index >= 0 && index < model->footer.size() ? &model->footer[index] : NULL;
}

void ScreenView::showToggleState(int index, bool isOn)
{
    ScreenModel next = model;
    ScreenComponent *toggle = componentIn(&next, index);
    if (toggle) {
        toggle->isOn = isOn;
        showScreen(next);
    }
}

void ScreenView::showRadioValue(int index, const QString &value)
{
    ScreenModel next = model;
    ScreenComponent *radio = componentIn(&next, index);
    if (radio) {
        radio->value = value;
        showScreen(next);
    }
}

void ScreenView::setInputBlocked(bool isBlocked)
{
    isInputBlocked = isBlocked;
    setAcceptsPresses(!isBlocked);
}

void ScreenView::relayout()
{
    if (!hasScreen) {
        return;
    }
    layout = ScreenLayout(model, height(), hostTexts());
    pageIndex = qBound(0, pageIndex, layout.pageCount() - 1);
    applyPage(pageIndex, true);
}

// ---- Refresh ----

void ScreenView::applyPage(int index, bool isFullRefresh)
{
    ScreenPage next = layout.page(index);
    if (isFullRefresh) {
        page = next;
        scheduleRefresh(DrawerLogic::refreshForPageChange(&refreshPolicy), QRegion());
        return;
    }
    bool isSameStructure = false;
    QRegion region = changedRegion(next, &isSameStructure);
    page = next;
    if (!isSameStructure) {
        scheduleRefresh(DrawerLogic::refreshForPageChange(&refreshPolicy), QRegion());
    } else if (!region.isEmpty()) {
        scheduleRefresh(DrawerLogic::refreshForStatusChange(&refreshPolicy), region);
    }
}

// The union of the primitives that changed; the structure is the same when
// every primitive kept its shape and place (only texts, fills or marks changed).
QRegion ScreenView::changedRegion(const ScreenPage &next, bool *isSameStructure) const
{
    QRegion region;
    *isSameStructure = next.items.size() == page.items.size();
    for (int i = 0; *isSameStructure && i < next.items.size(); i++) {
        const PaintItem &before = page.items[i];
        const PaintItem &after = next.items[i];
        if (before.shape != after.shape || before.rect != after.rect) {
            *isSameStructure = false;
        } else if (before != after) {
            region |= after.rect;
        }
    }
    return region;
}

void ScreenView::scheduleRefresh(DrawerLogic::RefreshKind kind, const QRegion &region)
{
    if (kind == DrawerLogic::REFRESH_FULL || region.isEmpty()) {
        update();
        if (isVisible()) {
            QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(flashNow()));
        }
        return;
    }
    update(region); // Qt repaints only this region: a partial e-ink update
}

void ScreenView::flashNow()
{
    NickelBridge::requestFullRefresh(this);
}

// ---- Input ----

bool ScreenView::eventFilter(QObject *watched, QEvent *event)
{
    if (watched != host) {
        return false;
    }
    if (event->type() == QEvent::Resize) {
        setGeometry(host->rect());
        relayout();
        return false;
    }
    if (event->type() == QEvent::KeyPress) {
        return handlePageKey(static_cast<QKeyEvent *>(event)->key());
    }
    return false;
}

// Page keys reach the drawer view (Nickel's current view); while a screen is
// open they page the screen and never the drawer underneath.
bool ScreenView::handlePageKey(int key)
{
    DrawerLogic::PageKey pageKey = DrawerLogic::pageKeyFor(key);
    if (pageKey == DrawerLogic::PAGE_KEY_NONE) {
        return false;
    }
    if (isInputBlocked) {
        nh_log("screen: page key ignored while a dialog or action is pending");
        return true;
    }
    goToPage(DrawerLogic::pageAfterKey(pageIndex, pageKey, layout.pageCount(), ONE_PAGE_PER_STEP));
    return true;
}

void ScreenView::goToPage(int index)
{
    if (index == pageIndex) {
        return;
    }
    nh_log("screen: page %d -> %d", pageIndex + 1, index + 1);
    pageIndex = index;
    applyPage(index, true);
}

void ScreenView::onTapped()
{
    if (isInputBlocked || isTapPending) {
        return;
    }
    pendingTapPosition = pressPosition();
    isTapPending = true;
    // Handled from the event loop, after the touch handler returned (re/notes/protoA-crash.md).
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(handleTap()));
}

void ScreenView::handleTap()
{
    isTapPending = false;
    if (isInputBlocked) {
        return;
    }
    if (sinceLastTap.isValid() && sinceLastTap.elapsed() < TAP_DEBOUNCE_MILLISECONDS) {
        return;
    }
    for (int i = page.targets.size() - 1; i >= 0; i--) {
        const TapTarget &target = page.targets[i];
        if (!target.rect.contains(pendingTapPosition)) {
            continue;
        }
        sinceLastTap.start();
        switch (target.kind) {
        case TapTarget::TAP_BACK:
            emit backRequested();
            return;
        case TapTarget::TAP_HEADER_ACTION:
            emit headerActionTapped();
            return;
        case TapTarget::TAP_PREVIOUS_PAGE:
            goToPage(qMax(0, pageIndex - 1));
            return;
        case TapTarget::TAP_NEXT_PAGE:
            goToPage(qMin(layout.pageCount() - 1, pageIndex + 1));
            return;
        case TapTarget::TAP_COMPONENT:
            emit componentTapped(target.component, target.value);
            return;
        }
    }
}

// ---- Painting (cmd/ndpreview/render_png.go) ----

void ScreenView::paintEvent(QPaintEvent *event)
{
    QPainter painter(this);
    painter.fillRect(event->rect(), grey(PAPER_WHITE));
    for (int i = 0; i < page.items.size(); i++) {
        if (page.items[i].rect.intersects(event->rect())) {
            paintItem(&painter, page.items[i]);
        }
    }
}

static void paintRect(QPainter *painter, const PaintItem &item)
{
    const QRect &r = item.rect;
    if (item.fill != NO_FILL) {
        painter->fillRect(r, grey(item.fill));
    }
    if (item.border > 0) {
        QColor ink = grey(INK_BLACK);
        painter->fillRect(QRect(r.x(), r.y(), r.width(), item.border), ink);
        painter->fillRect(QRect(r.x(), r.bottom() + 1 - item.border, r.width(), item.border), ink);
        painter->fillRect(QRect(r.x(), r.y(), item.border, r.height()), ink);
        painter->fillRect(QRect(r.right() + 1 - item.border, r.y(), item.border, r.height()), ink);
    }
}

// The border is drawn inside the ellipse, like the preview.
static void paintEllipse(QPainter *painter, const PaintItem &item)
{
    painter->save();
    painter->setRenderHint(QPainter::Antialiasing, true);
    painter->setBrush(item.fill == NO_FILL ? QBrush(Qt::NoBrush) : QBrush(grey(item.fill)));
    if (item.border > 0) {
        painter->setPen(QPen(grey(INK_BLACK), item.border));
        qreal inset = item.border / 2.0;
        painter->drawEllipse(QRectF(item.rect).adjusted(inset, inset, -inset, -inset));
    } else {
        painter->setPen(Qt::NoPen);
        painter->drawEllipse(item.rect);
    }
    painter->restore();
}

// Lines are vertically centred in their line height. The layout already
// elided them with the preview's estimate; a real font that is still wider
// is elided again rather than clipped mid-glyph.
static void paintText(QPainter *painter, const PaintItem &item)
{
    painter->save();
    QFont font = ScreenFonts::font(item.family, item.size, item.isBold, item.isUnderlined);
    painter->setFont(font);
    painter->setPen(grey(item.ink));
    painter->setClipRect(item.rect);
    QFontMetrics metrics(font);
    int flags = qtAlignment(item.align) | Qt::AlignVCenter | Qt::TextSingleLine;
    for (int i = 0; i < item.lines.size(); i++) {
        QRect lineRect(item.rect.x(), item.rect.y() + i * item.lineHeight, item.rect.width(), item.lineHeight);
        const QString &line = item.lines[i];
        bool isTooWide = metrics.width(line) > lineRect.width();
        painter->drawText(lineRect, flags, isTooWide ? metrics.elidedText(line, Qt::ElideRight, lineRect.width()) : line);
    }
    painter->restore();
}

// Black modules on white with a 4-module quiet zone, scaled into the square.
static void paintQr(QPainter *painter, const PaintItem &item)
{
    painter->fillRect(item.rect, grey(PAPER_WHITE));
    int modules = item.qrSide + 2 * QR_QUIET_ZONE_MODULES;
    QrPlacement placement = placeQr(modules, item.rect.width());
    if (placement.moduleSize <= 0 || item.qrModules.size() != item.qrSide * item.qrSide) {
        return;
    }
    QColor ink = grey(INK_BLACK);
    int left = item.rect.x() + placement.offset + QR_QUIET_ZONE_MODULES * placement.moduleSize;
    int top = item.rect.y() + placement.offset + QR_QUIET_ZONE_MODULES * placement.moduleSize;
    for (int y = 0; y < item.qrSide; y++) {
        for (int x = 0; x < item.qrSide; x++) {
            if (item.qrModules.at(y * item.qrSide + x)) {
                painter->fillRect(left + x * placement.moduleSize, top + y * placement.moduleSize, placement.moduleSize,
                                  placement.moduleSize, ink);
            }
        }
    }
}

void ScreenView::paintItem(QPainter *painter, const PaintItem &item)
{
    switch (item.shape) {
    case PaintItem::SHAPE_RECT:
        paintRect(painter, item);
        break;
    case PaintItem::SHAPE_ELLIPSE:
        paintEllipse(painter, item);
        break;
    case PaintItem::SHAPE_TEXT:
        paintText(painter, item);
        break;
    case PaintItem::SHAPE_QR:
        paintQr(painter, item);
        break;
    }
}
