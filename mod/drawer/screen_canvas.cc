#include "screen_canvas.h"

#include <QEvent>
#include <QFontMetrics>
#include <QKeyEvent>
#include <QPaintEvent>
#include <QPainter>
#include <QTimer>

#include "debug_log.h"

#include "fallback_strings.h"
#include "nickel_bridge.h"
#include "nickel_fonts.h"
#include "open_popups.h"

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

static const char *targetName(TapTarget::Kind kind)
{
    switch (kind) {
    case TapTarget::TAP_PREVIOUS_PAGE:
        return "previous page";
    case TapTarget::TAP_NEXT_PAGE:
        return "next page";
    case TapTarget::TAP_COMPONENT:
        break;
    }
    return "component";
}

ScreenCanvas::ScreenCanvas(QWidget *drawerView, QWidget *parent)
    : PressArea(parent), host(drawerView), hasScreen(false), pageIndex(0), pendingPage(-1), isInputBlocked(false)
{
    DrawerLogic::resetRefreshPolicy(&refreshPolicy);
    setObjectName(QStringLiteral("nickelDrawerScreenCanvas"));
    setAttribute(Qt::WA_OpaquePaintEvent, true); // paintEvent covers every pixel
    host->installEventFilter(this);
    connect(this, SIGNAL(tapped()), this, SLOT(onTapped()));
}

const ScreenComponent *ScreenCanvas::component(int index) const
{
    return index >= 0 && index < layout.componentCount() ? &layout.component(index) : NULL;
}

void ScreenCanvas::showScreen(const ScreenModel &next)
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
        pageIndex = pendingPage < 0 ? 0 : qBound(0, pendingPage, layout.pageCount() - 1);
        pendingPage = -1;
        applyPage(pageIndex, true);
        return;
    }
    pageIndex = qBound(0, pageIndex, layout.pageCount() - 1);
    applyPage(pageIndex, false);
}

void ScreenCanvas::setInputBlocked(bool isBlocked)
{
    isInputBlocked = isBlocked;
    setAcceptsPresses(!isBlocked);
}

void ScreenCanvas::resizeEvent(QResizeEvent *event)
{
    PressArea::resizeEvent(event);
    relayout();
}

void ScreenCanvas::relayout()
{
    if (!hasScreen) {
        return;
    }
    layout = ScreenLayout(model, height(), hostTexts());
    pageIndex = qBound(0, pageIndex, layout.pageCount() - 1);
    applyPage(pageIndex, true);
}

// ---- Refresh ----

void ScreenCanvas::applyPage(int index, bool isFullRefresh)
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
QRegion ScreenCanvas::changedRegion(const ScreenPage &next, bool *isSameStructure) const
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

void ScreenCanvas::scheduleRefresh(DrawerLogic::RefreshKind kind, const QRegion &region)
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

// The whole screen flashes, header included: the header is a sibling, so the
// parent (the ScreenView) is repainted.
void ScreenCanvas::flashNow()
{
    NickelBridge::requestFullRefresh(parentWidget());
}

// ---- Input ----

bool ScreenCanvas::eventFilter(QObject *watched, QEvent *event)
{
    if (watched == host && event->type() == QEvent::KeyPress) {
        return handlePageKey(static_cast<QKeyEvent *>(event)->key());
    }
    return false;
}

// Page keys reach the drawer view (Nickel's current view); while a screen is
// open they page the screen and never the drawer underneath.
bool ScreenCanvas::handlePageKey(int key)
{
    DrawerLogic::PageKey pageKey = DrawerLogic::pageKeyFor(key);
    if (pageKey == DrawerLogic::PAGE_KEY_NONE) {
        return false;
    }
    if (OpenPopups::isAnyOpen()) {
        OpenPopups::closeAll("page key on app screen"); // closes the popup instead of paging under it
        return true;
    }
    if (isInputBlocked) {
        DebugLog::write("screen: page key ignored while an action is pending");
        return true;
    }
    goToPage(DrawerLogic::pageAfterKey(pageIndex, pageKey, layout.pageCount(), ONE_PAGE_PER_STEP));
    return true;
}

void ScreenCanvas::goToPage(int index)
{
    if (index == pageIndex) {
        return;
    }
    OpenPopups::closeAll("app screen page change");
    DebugLog::write("screen: page %d -> %d", pageIndex + 1, index + 1);
    pageIndex = index;
    applyPage(index, true);
}

// The topmost target under position (later targets lie over earlier ones), or NULL.
const TapTarget *ScreenCanvas::targetAt(const QPoint &position) const
{
    for (int i = page.targets.size() - 1; i >= 0; i--) {
        if (page.targets[i].rect.contains(position)) {
            return &page.targets[i];
        }
    }
    return NULL;
}

// Only the pressed target shows the pressed state, inverted.
PressArea::PressFeedback ScreenCanvas::feedbackAt(const QPoint &position) const
{
    PressFeedback feedback;
    feedback.highlight = HIGHLIGHT_NONE;
    const TapTarget *target = isInputBlocked ? NULL : targetAt(position);
    if (target) {
        feedback.rect = target->rect;
        feedback.highlight = HIGHLIGHT_INVERT;
    }
    return feedback;
}

// PressArea emits tapped() from the event loop, after the touch handler
// returned (re/notes/protoA-crash.md), debounced, and clears the pressed
// state after this.
void ScreenCanvas::onTapped()
{
    QPoint position = pressPosition();
    const TapTarget *found = isInputBlocked ? NULL : targetAt(position);
    if (!found) {
        DebugLog::write("screen: tap at %d,%d ignored (%s)", position.x(), position.y(),
                        isInputBlocked ? "input blocked" : "no target: text or a disabled control");
        return;
    }
    TapTarget target = *found; // the action may relayout the page
    DebugLog::write("screen: tap on %s %d (\"%s\") at %d,%d", targetName(target.kind), target.component,
                    qPrintable(target.value), position.x(), position.y());
    switch (target.kind) {
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

// ---- Painting (cmd/ndpreview/render_png.go) ----

// Borders are drawn in the item's ink (a disabled button's outline is dim).
static void paintRect(QPainter *painter, const PaintItem &item)
{
    const QRect &r = item.rect;
    if (item.fill != NO_FILL) {
        painter->fillRect(r, grey(item.fill));
    }
    if (item.border > 0) {
        QColor ink = grey(item.ink);
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
        painter->setPen(QPen(grey(item.ink), item.border));
        qreal inset = item.border / 2.0;
        painter->drawEllipse(QRectF(item.rect).adjusted(inset, inset, -inset, -inset));
    } else {
        painter->setPen(Qt::NoPen);
        painter->drawEllipse(item.rect);
    }
    painter->restore();
}

// Screens name a family kind; the families themselves are Nickel's (never generic ones).
static NickelFonts::Role fontRoleFor(ScreenLogic::Family family)
{
    switch (family) {
    case ScreenLogic::FAMILY_SERIF:
        return NickelFonts::ROLE_TITLE;
    case ScreenLogic::FAMILY_MONO:
        return NickelFonts::ROLE_MONO;
    case ScreenLogic::FAMILY_SANS:
        break;
    }
    return NickelFonts::ROLE_BODY;
}

// Lines are vertically centred in their line height. The layout already
// elided them with the preview's estimate; a real font that is still wider
// is elided again rather than clipped mid-glyph.
static void paintText(QPainter *painter, const PaintItem &item)
{
    painter->save();
    QFont font = NickelFonts::font(fontRoleFor(item.family), item.size, item.isBold);
    font.setUnderline(item.isUnderlined);
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

static void paintItem(QPainter *painter, const PaintItem &item)
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

void ScreenCanvas::paintEvent(QPaintEvent *event)
{
    QPainter painter(this);
    painter.fillRect(event->rect(), grey(PAPER_WHITE));
    for (int i = 0; i < page.items.size(); i++) {
        if (page.items[i].rect.intersects(event->rect())) {
            paintItem(&painter, page.items[i]);
        }
    }
}
