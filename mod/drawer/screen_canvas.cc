#include "screen_canvas.h"

#include <QEvent>
#include <QFontMetrics>
#include <QKeyEvent>
#include <QPaintEvent>
#include <QPainter>
#include <QVector>
#include <QTimer>

#include "debug_log.h"

#include "fallback_strings.h"
#include "nickel_bridge.h"
#include "nickel_fonts.h"
#include "open_popups.h"
#include "picture_loader.h"
#include "system_icon.h"
#include "text_meter.h"

using namespace ScreenLogic;

static const int RUN_FROM_EVENT_LOOP = 0;
static const int ONE_PAGE_PER_STEP = 1;
static const qreal DASH_LENGTH = 4; // in pen widths
static const int MISSING_GLYPH_WIDTH = 96;
static const int MISSING_GLYPH_HEIGHT = 80;

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
    case TapTarget::TAP_TILE:
        return "tile";
    case TapTarget::TAP_LIST_ROW:
        return "list row";
    case TapTarget::TAP_VIEWER_PREVIOUS:
        return "viewer previous zone";
    case TapTarget::TAP_VIEWER_NEXT:
        return "viewer next zone";
    case TapTarget::TAP_VIEWER_TOGGLE_CHROME:
        return "viewer chrome zone";
    case TapTarget::TAP_VIEWER_BACK:
        return "viewer full-view back button";
    case TapTarget::TAP_COMPONENT:
        break;
    }
    return "component";
}

ScreenCanvas::ScreenCanvas(QWidget *drawerView, QWidget *parent)
    : PressArea(parent), host(drawerView), hasScreen(false), pageIndex(0), pendingPage(-1), isInputBlocked(false),
      isChrome(true)
{
    DrawerLogic::resetRefreshPolicy(&refreshPolicy);
    setObjectName(QStringLiteral("nickelDrawerScreenCanvas"));
    setAttribute(Qt::WA_OpaquePaintEvent, true); // paintEvent covers every pixel
    host->installEventFilter(this);
    connect(this, SIGNAL(tapped()), this, SLOT(onTapped()));
    connect(this, SIGNAL(longPressed()), this, SLOT(onLongPressed()));
}

bool ScreenCanvas::componentAt(int index, ScreenComponent *copy) const
{
    if (index < 0 || index >= layout.componentCount()) {
        return false;
    }
    *copy = layout.component(index);
    return true;
}

QString ScreenCanvas::viewerPictureId() const
{
    const PictureTile *tile = layout.viewerShownTile();
    return tile ? tile->id : QString();
}

bool ScreenCanvas::isViewerPictureCurrent() const
{
    const PictureTile *tile = layout.viewerShownTile();
    return tile && tile->isCurrent;
}

void ScreenCanvas::dropPictures()
{
    if (!pixmaps.isEmpty()) {
        DebugLog::write("screen: %d decoded picture(s) released", pixmaps.size());
        pixmaps.clear();
    }
}

void ScreenCanvas::showScreen(const ScreenModel &next)
{
    bool isSameScreen = hasScreen && next.app == model.app && next.screen == model.screen;
    // The picture on screen is the user's, not the reply's: a reload that was
    // asked before a flip must not jump back.
    QString keptPicture = isSameScreen ? viewerPictureId() : QString();
    model = next;
    hasScreen = true;
    layout = ScreenLayout(model, height(), hostTexts());
    if (layout.viewerComponent() >= 0 && !isSameScreen) {
        isChrome = true;
    }
    restoreViewerState(keptPicture);
    if (!isSameScreen && model.screen.isEmpty()) {
        // A placeholder (no screen id: loading) is drawn without the flash.
        pageIndex = 0;
        page = layout.page(0);
        refreshPictures();
        update();
        return;
    }
    if (!isSameScreen) {
        pageIndex = pendingPage < 0 ? 0 : qBound(0, pendingPage, layout.pageCount() - 1);
        int tilePage = layout.pageOfGalleryTile(pendingTileId);
        pageIndex = tilePage >= 0 ? tilePage : pageIndex;
        pendingPage = -1;
        pendingTileId.clear();
        applyPage(pageIndex, true);
    } else {
        pageIndex = qBound(0, pageIndex, layout.pageCount() - 1);
        applyPage(pageIndex, layout.viewerComponent() >= 0 && viewerPictureId() != keptPicture);
    }
    emit viewerStateChanged();
    if (layout.viewerComponent() >= 0 && !viewerPictureId().isEmpty() && viewerPictureId() != keptPicture) {
        emit viewerPictureChanged(viewerPictureId());
    }
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
    QString keptPicture = viewerPictureId();
    layout = ScreenLayout(model, height(), hostTexts());
    restoreViewerState(keptPicture);
    if (hasViewer()) {
        DebugLog::write("viewer: relayout for canvas %dx%d, chrome %s, picture \"%s\" kept", width(), height(),
                        isChrome ? "shown" : "hidden", qPrintable(viewerPictureId()));
    }
    pageIndex = qBound(0, pageIndex, layout.pageCount() - 1);
    applyPage(pageIndex, true);
}

// A new layout starts with the chrome shown and the reply's picture; the
// viewer keeps the user's own: the chrome as last toggled and the picture on
// screen. Without this a resize showed the chrome on the canvas while the
// header stayed hidden.
void ScreenCanvas::restoreViewerState(const QString &keptPicture)
{
    if (layout.viewerComponent() < 0) {
        return;
    }
    layout.setChromeShown(isChrome);
    int kept = layout.viewerIndexOfTile(keptPicture);
    if (kept >= 0) {
        layout.showViewerPictureAt(kept);
    }
}

// ---- Refresh ----

void ScreenCanvas::applyPage(int index, bool isFullRefresh)
{
    ScreenPage next = layout.page(index);
    if (isFullRefresh) {
        page = next;
        refreshPictures();
        logTargets();
        scheduleRefresh(DrawerLogic::refreshForPageChange(&refreshPolicy), QRegion());
        return;
    }
    bool isSameStructure = false;
    QRegion region = changedRegion(next, &isSameStructure);
    page = next;
    refreshPictures();
    if (!isSameStructure) {
        logTargets();
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

// Keeps only the pictures of the shown page decoded (the previous page's, or
// the previous viewer picture, are released first), then decodes what is new.
// Reads are small and capped (picture_loader.h) and happen when the page is
// laid out, never while painting.
void ScreenCanvas::refreshPictures()
{
    QHash<QString, QSize> wanted;
    for (int i = 0; i < page.items.size(); i++) {
        const PaintItem &item = page.items[i];
        if (item.shape == PaintItem::SHAPE_IMAGE && !item.imagePath.isEmpty()) {
            wanted.insert(item.imagePath, item.rect.size());
        }
    }
    QList<QString> held = pixmaps.keys();
    int released = 0;
    for (int i = 0; i < held.size(); i++) {
        if (!wanted.contains(held[i]) || wanted.value(held[i]) != pixmaps.value(held[i]).box) {
            pixmaps.remove(held[i]);
            released++;
        }
    }
    int decoded = 0;
    for (QHash<QString, QSize>::const_iterator it = wanted.constBegin(); it != wanted.constEnd(); ++it) {
        if (pixmaps.contains(it.key())) {
            continue;
        }
        CachedPicture picture;
        picture.box = it.value();
        picture.pixmap = PictureLoader::load(it.key(), it.value());
        pixmaps.insert(it.key(), picture);
        decoded++;
    }
    if (released > 0 || decoded > 0) {
        DebugLog::write("screen: pictures released %d, decoded %d, held %d", released, decoded, pixmaps.size());
    }
}

const QPixmap *ScreenCanvas::pictureFor(const PaintItem &item) const
{
    QHash<QString, CachedPicture>::const_iterator found = pixmaps.constFind(item.imagePath);
    return found == pixmaps.constEnd() || found.value().pixmap.isNull() ? NULL : &found.value().pixmap;
}

void ScreenCanvas::hideEvent(QHideEvent *event)
{
    PressArea::hideEvent(event);
    dropPictures(); // a hidden screen keeps nothing decoded
}

void ScreenCanvas::showEvent(QShowEvent *event)
{
    PressArea::showEvent(event);
    if (hasScreen && pixmaps.isEmpty()) {
        refreshPictures();
        update();
    }
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

// Once per new page layout (not on live value updates): what a tap can hit.
void ScreenCanvas::logTargets() const
{
    DebugLog::write("screen: \"%s\" page %d/%d, %d tap target(s) on canvas %dx%d%s", qPrintable(model.screen),
                    pageIndex + 1, layout.pageCount(), page.targets.size(), width(), height(),
                    isInputBlocked ? ", input blocked" : "");
    for (int i = 0; i < page.targets.size(); i++) {
        const TapTarget &target = page.targets[i];
        bool hasComponent = target.component >= 0 && target.component < layout.componentCount();
        DebugLog::write("screen: target %d: %s %d (%s \"%s\") at %d,%d %dx%d", i, targetName(target.kind),
                        target.component, hasComponent ? qPrintable(layout.component(target.component).type) : "-", qPrintable(target.value),
                        target.rect.x(), target.rect.y(), target.rect.width(), target.rect.height());
    }
}

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
    if (hasViewer()) {
        stepViewer(pageKey == DrawerLogic::PAGE_KEY_NEXT ? 1 : -1);
        return true;
    }
    goToPage(DrawerLogic::pageAfterKey(pageIndex, pageKey, layout.pageCount(), ONE_PAGE_PER_STEP));
    return true;
}

// Every picture change is a full refresh (GC16); at the ends nothing happens.
void ScreenCanvas::stepViewer(int step)
{
    int next = ScreenLogic::viewerIndexAfterStep(layout.viewerShownIndex(), layout.viewerCount(), step);
    if (next == layout.viewerShownIndex()) {
        DebugLog::write("viewer: no %s picture (%d of %d), nothing changes; chrome %s", step < 0 ? "previous" : "next",
                        layout.viewerShownIndex() + 1, layout.viewerCount(), isChrome ? "shown" : "hidden");
        return;
    }
    OpenPopups::closeAll("viewer picture change");
    DebugLog::write("viewer: picture %d -> %d of %d", layout.viewerShownIndex() + 1, next + 1, layout.viewerCount());
    layout.showViewerPictureAt(next);
    applyPage(0, true);
    emit viewerStateChanged();
    emit viewerPictureChanged(viewerPictureId());
}

void ScreenCanvas::toggleChrome()
{
    isChrome = !isChrome;
    layout.setChromeShown(isChrome);
    applyPage(0, true);
    DebugLog::write("viewer: chrome %s, %d tap target(s), canvas %dx%d", isChrome ? "shown" : "hidden",
                    page.targets.size(), width(), height());
    emit viewerStateChanged();
}

// The full view's back button: one level back, from the full view to the viewer.
void ScreenCanvas::showChrome()
{
    if (isChrome) {
        DebugLog::write("viewer: back button tapped with the chrome already shown, ignored");
        return;
    }
    toggleChrome();
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
    bool isZone = target && (target->kind == TapTarget::TAP_VIEWER_PREVIOUS || target->kind == TapTarget::TAP_VIEWER_NEXT ||
                             target->kind == TapTarget::TAP_VIEWER_TOGGLE_CHROME);
    if (target && !isZone) { // a third of the picture never inverts
        feedback.rect = target->rect;
        // An icon button over a picture: Nickel's black circle, not the whole target inverted.
        feedback.highlight = target->kind == TapTarget::TAP_VIEWER_BACK ? HIGHLIGHT_CIRCLE : HIGHLIGHT_INVERT;
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
        DebugLog::write("screen: tap at %d,%d ignored (%s; %d target(s), canvas %dx%d%s)", position.x(), position.y(),
                        isInputBlocked ? "input blocked" : "no target: text or a disabled control", page.targets.size(),
                        width(), height(), hasViewer() ? (isChrome ? ", viewer chrome shown" : ", viewer full view") : "");
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
    case TapTarget::TAP_VIEWER_PREVIOUS:
        stepViewer(-1);
        return;
    case TapTarget::TAP_VIEWER_NEXT:
        stepViewer(1);
        return;
    case TapTarget::TAP_VIEWER_TOGGLE_CHROME:
        toggleChrome();
        return;
    case TapTarget::TAP_VIEWER_BACK:
        showChrome();
        return;
    case TapTarget::TAP_TILE:
    case TapTarget::TAP_LIST_ROW:
    case TapTarget::TAP_COMPONENT:
        emit componentTapped(target.component, target.value);
        return;
    }
}

// A held gallery tile or list row opens its menu at the target; PressArea keeps
// the pressed state until this returns, and does not also send tapped().
void ScreenCanvas::onLongPressed()
{
    QPoint position = pressPosition();
    const TapTarget *found = isInputBlocked ? NULL : targetAt(position);
    bool isMenuTarget = found && (found->kind == TapTarget::TAP_TILE || found->kind == TapTarget::TAP_LIST_ROW);
    if (!isMenuTarget) {
        DebugLog::write("screen: long press at %d,%d ignored (%s)", position.x(), position.y(),
                        found ? "not a gallery tile or list row" : "no target");
        return;
    }
    TapTarget target = *found;
    QPoint anchor = mapToGlobal(target.rect.center());
    if (target.kind == TapTarget::TAP_LIST_ROW) {
        DebugLog::write("screen: long press on list row %d (\"%s\") at %d,%d", target.row, qPrintable(target.value),
                        position.x(), position.y());
        emit listRowLongPressed(target.component, target.row, target.value, anchor);
        return;
    }
    DebugLog::write("screen: long press on tile \"%s\" at %d,%d", qPrintable(target.value), position.x(), position.y());
    emit tileLongPressed(target.component, target.value, anchor);
}

// ---- Painting (cmd/ndpreview/render_png.go) ----

// Borders are drawn in the item's ink (a disabled button's outline is dim).
static void paintDashedRect(QPainter *painter, const PaintItem &item)
{
    painter->save();
    QPen pen(grey(item.ink), item.border);
    pen.setStyle(Qt::DashLine);
    pen.setDashPattern(QVector<qreal>() << DASH_LENGTH << DASH_LENGTH);
    painter->setPen(pen);
    painter->setBrush(Qt::NoBrush);
    qreal inset = item.border / 2.0;
    painter->drawRect(QRectF(item.rect).adjusted(inset, inset, -inset - 1, -inset - 1));
    painter->restore();
}

static void paintRect(QPainter *painter, const PaintItem &item)
{
    if (item.isDashed) {
        paintDashedRect(painter, item);
        return;
    }
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

// Lines are vertically centred in their line height. The layout already measured
// them with this font (text_meter.h); a line that is still wider, because the
// font changed after the layout, is elided rather than clipped mid-glyph.
static void paintText(QPainter *painter, const PaintItem &item)
{
    painter->save();
    QFont font = NickelFonts::font(QtTextMeter::roleFor(item.family), item.size, item.isBold);
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

// Icons come from system_icon.h (Nickel's, else Tabler PNGs); we never draw
// them. Black icons take a lighter ink as opacity on the white paper.
static void paintIcon(QPainter *painter, const QRect &box, const QPixmap &image, int ink)
{
    painter->save();
    painter->setOpacity(1.0 - ink / 255.0);
    SystemIcon::drawCentred(painter, box, image);
    painter->restore();
}

static void paintPictureIcon(QPainter *painter, const QRect &box, int ink)
{
    paintIcon(painter, box, SystemIcon::icon(SystemIcon::PICTURE).normal, ink);
}

// The current picture's badge: Nickel's check on its black disc (it covers the
// layout's disc), or the Tabler check drawn white on that disc.
static void paintCheck(QPainter *painter, const PaintItem &item)
{
    if (!item.isOn) {
        return;
    }
    const SystemIcon::Icon &check = SystemIcon::icon(SystemIcon::CHECK_BADGE);
    if (check.source != SystemIcon::SOURCE_TABLER) {
        SystemIcon::drawCentred(painter, item.rect, check.normal);
        return;
    }
    static QPixmap whiteCheck = SystemIcon::whiteInk(check.normal);
    SystemIcon::drawCentred(painter, item.rect, whiteCheck);
}

static void paintBackArrow(QPainter *painter, const PaintItem &item)
{
    paintIcon(painter, item.rect, SystemIcon::icon(SystemIcon::BACK).normal, item.ink);
}

// Nickel's switch image at its own size; without it the state word alone shows the state.
static void paintSwitch(QPainter *painter, const PaintItem &item)
{
    SystemIcon::Id id = item.isOn ? SystemIcon::SWITCH_ON : SystemIcon::SWITCH_OFF;
    SystemIcon::drawCentred(painter, item.rect, SystemIcon::icon(id).normal);
}

// The picture centred in its box (it was decoded to fit); the glyph when it is missing.
static void paintImage(QPainter *painter, const PaintItem &item, const QPixmap *picture)
{
    if (picture) {
        painter->drawPixmap(item.rect.x() + (item.rect.width() - picture->width()) / 2,
                            item.rect.y() + (item.rect.height() - picture->height()) / 2, *picture);
        return;
    }
    if (item.showsMissingGlyph) {
        QRect glyph(0, 0, MISSING_GLYPH_WIDTH, MISSING_GLYPH_HEIGHT);
        glyph.moveCenter(item.rect.center());
        paintPictureIcon(painter, glyph, INK_DIM);
    }
}

static void paintItem(QPainter *painter, const PaintItem &item, const QPixmap *picture)
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
    case PaintItem::SHAPE_IMAGE:
        paintImage(painter, item, picture);
        break;
    case PaintItem::SHAPE_CHECK:
        paintCheck(painter, item);
        break;
    case PaintItem::SHAPE_PICTURE_GLYPH:
        paintPictureIcon(painter, item.rect, item.ink);
        break;
    case PaintItem::SHAPE_BACK_ARROW:
        paintBackArrow(painter, item);
        break;
    case PaintItem::SHAPE_SWITCH:
        paintSwitch(painter, item);
        break;
    }
}

void ScreenCanvas::paintEvent(QPaintEvent *event)
{
    QPainter painter(this);
    painter.fillRect(event->rect(), grey(PAPER_WHITE));
    for (int i = 0; i < page.items.size(); i++) {
        if (page.items[i].rect.intersects(event->rect())) {
            const PaintItem &item = page.items[i];
            paintItem(&painter, item, item.shape == PaintItem::SHAPE_IMAGE ? pictureFor(item) : NULL);
        }
    }
}
