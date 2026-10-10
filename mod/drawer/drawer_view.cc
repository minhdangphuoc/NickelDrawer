#include "drawer_view.h"

#include <QGridLayout>
#include <QHBoxLayout>
#include <QKeyEvent>
#include <QLabel>
#include <QResizeEvent>
#include <QTimer>
#include <QVBoxLayout>

#include "debug_log.h"

#include "fallback_strings.h"
#include "item_widgets.h"
#include "nickel_bridge.h"
#include "nickel_fonts.h"
#include "open_popups.h"
#include "page_header.h"
#include "screen_logic.h"
#include "system_icon.h"
#include "painted_marks.h"
#include "status_bar_inset.h"
#include "view_settings.h"

// Sizes in device pixels (docs/spec/ui-kit.md §2).
static const int SIDE_GUTTER = ScreenLogic::GUTTER;
static const int CONTENT_WIDTH = ScreenLogic::CONTENT_WIDTH;
static const int FOOTER_BUTTON_SIDE = 120;
static const char *const HOST_REFRESH_ID = "_refresh";
static const char *const HOST_UPDATES_ID = "_updates";

// Black on white, no grey fills; separators 2 px (ui-kit §1, §2). Families are
// Nickel's own, filled in at run time (nickel_fonts.h): never generic ones.
static const char *const VIEW_STYLE =
    "DrawerView { background: white; }"
    "#drawerHeader { background: white; }"
    "QWidget { font-family: @BODY_FAMILY@; }"
    "QLabel { color: black; background: transparent; }"
    "#drawerGlyph { font-size: 64px; }"
    "#drawerRow { border-bottom: 2px solid black; background: white; }"
    "#drawerTile { background: white; }"
    "#drawerItemName { font-size: 36px; }"
    "#drawerProblemName { font-size: 36px; font-weight: bold; }"
    "#drawerItemStatus { font-size: 28px; }"
    "#drawerChevron { font-size: 56px; }"
    "#drawerSectionLabel { font-size: 26px; font-weight: bold; }"
    "#drawerHint { font-size: 28px; }"
    "#drawerPageIndicator { font-size: 32px; }"
    "#drawerEmptyTitle { font-family: @TITLE_FAMILY@; font-size: 40px; font-weight: bold; }"
    "#drawerEmptyText { font-size: 30px; }";

static const int HOST_DOT_INSET = 28;

static QLabel *createTextLabel(const QString &text, const char *objectName, QWidget *parent)
{
    QLabel *label = new QLabel(text, parent);
    label->setObjectName(QString::fromLatin1(objectName));
    return label;
}

DrawerView::DrawerView()
    : QWidget(NULL), rootLayout(NULL), header(NULL), listToggle(NULL), gridToggle(NULL), pageArea(NULL), pageLayout(NULL),
      hintLabel(NULL), pageIndicator(NULL), refreshButton(NULL), updatesButton(NULL), updatesDot(NULL),
      viewMode(ViewSettings::load()), gridRows(DrawerLogic::GRID_DEFAULT_ROWS), currentPage(0), isInputBlocked(false),
      hasModel(false), hasLoggedLayout(false)
{
    DrawerLogic::resetRefreshPolicy(&refreshPolicy);
    setObjectName(QStringLiteral("nickelDrawerView"));
    setStyleSheet(NickelFonts::styleSheet(VIEW_STYLE));
    setAttribute(Qt::WA_StyledBackground, true);
    // Same status-bar properties MoreView sets, so Nickel's pushView does not guess.
    setProperty("showIndicators", true);
    setProperty("showStatusBottomLine", false);
    setProperty("showTitle", true);
    setFocusPolicy(Qt::StrongFocus);

    pageArea = new QWidget(this);
    pageLayout = new QVBoxLayout(pageArea);
    pageLayout->setContentsMargins(0, 0, 0, 0);
    pageLayout->setSpacing(0);
    // The page area takes what the header and footer leave; its content (grid
    // rows) is fitted to that height, never the other way round.
    pageArea->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Ignored);
    hintLabel = createTextLabel(QString(), "drawerHint", this);
    hintLabel->setAlignment(Qt::AlignHCenter);
    hintLabel->setWordWrap(true);

    // Top margin is the status-bar inset, known only once Nickel shows the view (showEvent).
    rootLayout = new QVBoxLayout(this);
    rootLayout->setContentsMargins(SIDE_GUTTER, StatusBarInset::FALLBACK_TOP_INSET, SIDE_GUTTER, 0);
    rootLayout->setSpacing(0);
    header = createHeader();
    rootLayout->addWidget(header);
    rootLayout->addWidget(pageArea, 1);
    rootLayout->addWidget(hintLabel);
    rootLayout->addWidget(createFooter());

    setViewMode(viewMode);
    renderPage(); // "Loading…" until the session delivers the model
    DebugLog::write("drawer view: constructed %p, header %p", static_cast<void *>(this), static_cast<void *>(header));
}

PressArea *DrawerView::createArrow(const char *glyph, const char *slotOrSignal, QWidget *parent)
{
    PressArea *button = new PressArea(parent);
    button->setObjectName(QString::fromUtf8(glyph));
    button->setFixedSize(FOOTER_BUTTON_SIDE, FOOTER_BUTTON_SIDE);
    button->setPressHighlight(PressArea::HIGHLIGHT_CIRCLE); // an icon button, like Nickel's *_hit icons
    QLabel *label = createTextLabel(QString::fromUtf8(glyph), "drawerGlyph", button);
    label->setAlignment(Qt::AlignCenter);
    label->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    QHBoxLayout *layout = new QHBoxLayout(button);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->addWidget(label);
    connect(button, SIGNAL(tapped()), this, slotOrSignal);
    pressables.append(button);
    return button;
}

// We never draw icons: a toggle whose icon is not installed is hidden (the
// drawer stays in the saved view).
static void hideToggleWithoutIcon(ViewToggleIcon *toggle, const char *name)
{
    if (!toggle->hasIcon()) {
        toggle->hide();
        DebugLog::write("drawer view: %s toggle icon not installed, toggle hidden", name);
    }
}

// Nickel's own TextHeader (back arrow, title, divider) when the firmware has
// it, else page_header.h's copy; our list/grid toggles at the right.
PageHeader *DrawerView::createHeader()
{
    PageHeader *pageHeader =
        new PageHeader(FallbackStrings::text(FallbackStrings::APPS_TITLE), NickelBridge::textHeaderApi(), this);
    connect(pageHeader, SIGNAL(backTapped()), this, SIGNAL(backRequested()));
    listToggle = new ViewToggleIcon(ViewToggleIcon::SHAPE_LIST, pageHeader);
    gridToggle = new ViewToggleIcon(ViewToggleIcon::SHAPE_GRID, pageHeader);
    connect(listToggle, SIGNAL(tapped()), this, SLOT(onListToggleTapped()));
    connect(gridToggle, SIGNAL(tapped()), this, SLOT(onGridToggleTapped()));
    refreshButton = createHostButton(HeaderIconButton::ICON_REFRESH, SLOT(onRefreshTapped()), pageHeader);
    updatesButton = createHostButton(HeaderIconButton::ICON_IMPORT, SLOT(onUpdatesTapped()), pageHeader);
    updatesDot = new UpdateDot(updatesButton);
    updatesDot->move(updatesButton->width() - updatesDot->width() - HOST_DOT_INSET, HOST_DOT_INSET);
    updatesDot->hide();
    pressables << listToggle << gridToggle;
    pageHeader->addTrailingWidget(refreshButton);
    pageHeader->addTrailingWidget(updatesButton);
    pageHeader->addTrailingWidget(listToggle);
    pageHeader->addTrailingWidget(gridToggle);
    hideToggleWithoutIcon(listToggle, "list");
    hideToggleWithoutIcon(gridToggle, "grid");
    DebugLog::write("drawer view: header created (%s back, title, list/grid toggles)",
                    pageHeader->usesNickelHeader() ? "Nickel's" : "own");
    return pageHeader;
}

// The grid's stand-ins for the "_refresh" and "_updates" host items. Without
// an icon the button stays hidden (we never draw one) and the log says so.
HeaderIconButton *DrawerView::createHostButton(HeaderIconButton::Icon icon, const char *slot, QWidget *parent)
{
    HeaderIconButton *button = new HeaderIconButton(icon, QString(), parent);
    connect(button, SIGNAL(tapped()), this, slot);
    pressables.append(button);
    button->hide();
    return button;
}

QWidget *DrawerView::createFooter()
{
    QWidget *footer = new QWidget(this);
    pageIndicator = createTextLabel(QString(), "drawerPageIndicator", footer);
    pageIndicator->setAlignment(Qt::AlignCenter);

    QHBoxLayout *layout = new QHBoxLayout(footer);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->addWidget(createArrow("‹", SLOT(onPreviousTapped()), footer));
    layout->addWidget(pageIndicator, 1);
    layout->addWidget(createArrow("›", SLOT(onNextTapped()), footer));
    return footer;
}

void DrawerView::setModel(const DrawerModel &model)
{
    bool isStatusOnlyChange = hasModel && !items.isEmpty() && hasSameItems(model);
    hasModel = true;
    header->setTitle(model.title.isEmpty() ? FallbackStrings::text(FallbackStrings::APPS_TITLE) : model.title);
    hint = model.hint.isEmpty() ? FallbackStrings::text(FallbackStrings::LONG_PRESS_HINT) : model.hint;
    hintLabel->setText(hint);
    flatten(model);
    rebuildShownIndices();
    updateHostButtons();
    if (isStatusOnlyChange) {
        applyLiveStates();
        scheduleRefresh(DrawerLogic::refreshForStatusChange(&refreshPolicy));
        return;
    }
    currentPage = DrawerLogic::clampPage(currentPage, shownIndices.size(), perPage());
    renderPage();
    scheduleRefresh(DrawerLogic::refreshForPageChange(&refreshPolicy));
}

void DrawerView::flatten(const DrawerModel &model)
{
    items.clear();
    sectionOfItem.clear();
    sectionTitles.clear();
    for (int section = 0; section < model.sections.size(); section++) {
        sectionTitles.append(model.sections[section].title);
        const QList<DrawerItem> &sectionItems = model.sections[section].items;
        for (int i = 0; i < sectionItems.size(); i++) {
            items.append(sectionItems[i]);
            sectionOfItem.append(section);
        }
    }
}

bool DrawerView::hasSameItems(const DrawerModel &model) const
{
    int index = 0;
    for (int section = 0; section < model.sections.size(); section++) {
        const QList<DrawerItem> &sectionItems = model.sections[section].items;
        for (int i = 0; i < sectionItems.size(); i++, index++) {
            if (index >= items.size() || items[index].id != sectionItems[i].id ||
                items[index].name != sectionItems[i].name) {
                return false;
            }
        }
    }
    return index == items.size();
}

void DrawerView::applyLiveStates()
{
    int first = DrawerLogic::firstIndexOnPage(currentPage, perPage());
    for (int i = 0; i < pageWidgets.size() && first + i < shownIndices.size(); i++) {
        pageWidgets[i]->applyLiveState(items[shownIndices[first + i]]);
    }
    updateHostButtons();
}

int DrawerView::perPage() const
{
    return DrawerLogic::itemsPerPage(viewMode, gridRows);
}

// The grid hides the host items its header buttons stand for; the list shows all.
void DrawerView::rebuildShownIndices()
{
    shownIndices.clear();
    for (int i = 0; i < items.size(); i++) {
        bool isHidden = viewMode == DrawerLogic::VIEW_GRID && DrawerLogic::isHeaderControlId(qPrintable(items[i].id));
        if (!isHidden) {
            shownIndices.append(i);
        }
    }
}

const DrawerItem *DrawerView::hostItem(const char *id) const
{
    for (int i = 0; i < items.size(); i++) {
        if (items[i].id == QLatin1String(id)) {
            return &items[i];
        }
    }
    return NULL;
}

// Grid mode only: the refresh button, and the updates button (with the dot when
// an update is waiting) when the host offers the item and an icon is installed.
void DrawerView::updateHostButtons()
{
    bool isGrid = viewMode == DrawerLogic::VIEW_GRID;
    const DrawerItem *updates = hostItem(HOST_UPDATES_ID);
    refreshButton->setVisible(isGrid && hostItem(HOST_REFRESH_ID) && !SystemIcon::icon(SystemIcon::REFRESH).isNull());
    updatesButton->setVisible(isGrid && updates && !SystemIcon::icon(SystemIcon::IMPORT).isNull());
    updatesDot->setVisible(updates && updates->hasUpdate);
}

void DrawerView::onRefreshTapped()
{
    const DrawerItem *item = isInputBlocked ? NULL : hostItem(HOST_REFRESH_ID);
    if (item) {
        emit itemTapped(*item);
    }
}

void DrawerView::onUpdatesTapped()
{
    const DrawerItem *item = isInputBlocked ? NULL : hostItem(HOST_UPDATES_ID);
    if (item) {
        emit itemTapped(*item);
    }
}

// Rows that fit the page area under the header; a change re-pages in place,
// keeping the first visible item on screen.
void DrawerView::fitGridRows()
{
    if (pageArea->height() <= 0) {
        return;
    }
    int rows = DrawerLogic::gridRowsThatFit(pageArea->height(), DrawerLogic::GRID_ROW_HEIGHT);
    if (rows == gridRows) {
        return;
    }
    DebugLog::write("drawer view: grid rows %d -> %d (page area %d px)", gridRows, rows, pageArea->height());
    int firstVisible = DrawerLogic::firstIndexOnPage(currentPage, perPage());
    gridRows = rows;
    if (viewMode == DrawerLogic::VIEW_GRID) {
        currentPage = DrawerLogic::pageContainingItem(firstVisible, perPage());
        renderPage();
        scheduleRefresh(DrawerLogic::refreshForPageChange(&refreshPolicy));
    }
}

void DrawerView::setInputBlocked(bool isBlocked)
{
    isInputBlocked = isBlocked;
    header->setBackEnabled(!isBlocked);
    for (int i = 0; i < pressables.size(); i++) {
        pressables[i]->setAcceptsPresses(!isBlocked);
    }
    for (int i = 0; i < pageWidgets.size(); i++) {
        pageWidgets[i]->setAcceptsPresses(!isBlocked);
    }
}

void DrawerView::setViewMode(DrawerLogic::ViewMode mode)
{
    int firstVisible = shownIndices.value(DrawerLogic::firstIndexOnPage(currentPage, perPage()), 0);
    viewMode = mode;
    listToggle->setActive(mode == DrawerLogic::VIEW_LIST);
    gridToggle->setActive(mode == DrawerLogic::VIEW_GRID);
    rebuildShownIndices();
    updateHostButtons();
    // Position of that item among the items the new mode shows.
    int position = 0;
    while (position < shownIndices.size() && shownIndices[position] < firstVisible) {
        position++;
    }
    currentPage = DrawerLogic::pageContainingItem(position, perPage());
}

void DrawerView::restoreState(DrawerLogic::ViewMode mode, int page)
{
    DrawerLogic::ViewMode previousMode = viewMode;
    int previousPage = currentPage;
    setViewMode(mode);
    int clamped = DrawerLogic::clampPage(page, shownIndices.size(), perPage());
    DebugLog::write("drawer view: restore %s, page %d (was %s, page %d)", mode == DrawerLogic::VIEW_GRID ? "grid" : "list",
                    clamped + 1, previousMode == DrawerLogic::VIEW_GRID ? "grid" : "list", previousPage + 1);
    if (mode == previousMode && clamped == previousPage) {
        currentPage = previousPage;
        return;
    }
    goToPage(clamped);
}

void DrawerView::onListToggleTapped()
{
    if (viewMode == DrawerLogic::VIEW_LIST) {
        return;
    }
    setViewMode(DrawerLogic::VIEW_LIST);
    ViewSettings::save(DrawerLogic::VIEW_LIST);
    goToPage(currentPage);
}

void DrawerView::onGridToggleTapped()
{
    if (viewMode == DrawerLogic::VIEW_GRID) {
        return;
    }
    setViewMode(DrawerLogic::VIEW_GRID);
    ViewSettings::save(DrawerLogic::VIEW_GRID);
    goToPage(currentPage);
}

void DrawerView::onPreviousTapped()
{
    goToPage(DrawerLogic::pageAfterKey(currentPage, DrawerLogic::PAGE_KEY_PREVIOUS, shownIndices.size(), perPage()));
}

void DrawerView::onNextTapped()
{
    goToPage(DrawerLogic::pageAfterKey(currentPage, DrawerLogic::PAGE_KEY_NEXT, shownIndices.size(), perPage()));
}

void DrawerView::goToPage(int page)
{
    OpenPopups::closeAll("drawer page change");
    currentPage = page;
    renderPage();
    scheduleRefresh(DrawerLogic::refreshForPageChange(&refreshPolicy));
}

void DrawerView::clearPage()
{
    pageWidgets.clear();
    while (QLayoutItem *child = pageLayout->takeAt(0)) {
        if (child->widget()) {
            child->widget()->hide();
            child->widget()->deleteLater();
        }
        delete child;
    }
}

void DrawerView::renderPage()
{
    clearPage();
    int pageSize = perPage();
    int first = DrawerLogic::firstIndexOnPage(currentPage, pageSize);
    int count = DrawerLogic::itemCountOnPage(currentPage, shownIndices.size(), pageSize);
    if (shownIndices.isEmpty()) {
        renderEmptyState();
    } else if (viewMode == DrawerLogic::VIEW_GRID) {
        renderGridPage(first, count);
    } else {
        renderListPage(first, count);
    }
    pageLayout->addStretch(1);
    updateFooter();
    setInputBlocked(isInputBlocked);
}

void DrawerView::renderListPage(int first, int count)
{
    for (int i = 0; i < count; i++) {
        int index = shownIndices[first + i]; // the list shows every item: the same as first + i
        int previousSection = i > 0 ? sectionOfItem[index - 1] : -1;
        if (DrawerLogic::startsSectionOnPage(i, sectionOfItem[index], previousSection)) {
            pageLayout->addWidget(createSectionLabel(sectionTitles[sectionOfItem[index]], pageArea));
        }
        AppRow *row = new AppRow(items[index], pageArea);
        connect(row, SIGNAL(tapped()), this, SLOT(onItemTapped()));
        connect(row, SIGNAL(longPressed()), this, SLOT(onItemLongPressed()));
        pageLayout->addWidget(row);
        pageWidgets.append(row);
    }
}

// Cells are placed with the pure grid math (drawer_logic), not a layout, so
// all of them have the same size and sit on exact column lines.
void DrawerView::renderGridPage(int first, int count)
{
    QWidget *gridHost = new QWidget(pageArea);
    gridHost->setFixedSize(CONTENT_WIDTH, gridRows * DrawerLogic::GRID_ROW_HEIGHT);
    for (int i = 0; i < count; i++) {
        DrawerLogic::CellRect cell =
            DrawerLogic::gridCellRect(i, DrawerLogic::GRID_COLUMNS, CONTENT_WIDTH, DrawerLogic::GRID_ROW_HEIGHT);
        AppTile *tile = new AppTile(items[shownIndices[first + i]], cell.width, cell.height, gridHost);
        tile->move(cell.x, cell.y);
        connect(tile, SIGNAL(tapped()), this, SLOT(onItemTapped()));
        connect(tile, SIGNAL(longPressed()), this, SLOT(onItemLongPressed()));
        pageWidgets.append(tile);
    }
    pageLayout->addWidget(gridHost);
}

void DrawerView::renderEmptyState()
{
    FallbackStrings::Key heading = hasModel ? FallbackStrings::EMPTY_TITLE : FallbackStrings::LOADING;
    QLabel *title = createTextLabel(FallbackStrings::text(heading), "drawerEmptyTitle", pageArea);
    QLabel *text = createTextLabel(hasModel ? FallbackStrings::text(FallbackStrings::EMPTY_HINT) : QString(),
                                   "drawerEmptyText", pageArea);
    title->setAlignment(Qt::AlignHCenter);
    text->setAlignment(Qt::AlignHCenter);
    text->setWordWrap(true);
    pageLayout->addStretch(1);
    pageLayout->addWidget(title);
    pageLayout->addWidget(text);
}

void DrawerView::updateFooter()
{
    int pages = DrawerLogic::pageCount(shownIndices.size(), perPage());
    pageIndicator->setText(QStringLiteral("%1 / %2").arg(currentPage + 1).arg(pages));
}

void DrawerView::scheduleRefresh(DrawerLogic::RefreshKind kind)
{
    // Partial: Qt repaints only the changed labels by itself. Full: flash once
    // after the new page is laid out.
    if (kind == DrawerLogic::REFRESH_FULL && isVisible()) {
        QTimer::singleShot(0, this, SLOT(flashNow()));
    }
}

void DrawerView::flashNow()
{
    NickelBridge::requestFullRefresh(this);
}

void DrawerView::showEvent(QShowEvent *event)
{
    QWidget::showEvent(event);
    applyStatusBarInset();
    StatusBarInset::logGeometry("after show", this, header);
    if (!hasLoggedLayout) {
        hasLoggedLayout = true;
        QTimer::singleShot(0, this, SLOT(logLayoutGeometry())); // runs after the layout pass
    }
    setFocus(Qt::OtherFocusReason);
    QTimer::singleShot(0, this, SLOT(fitGridRows())); // after the layout pass: the page area has its height
    scheduleRefresh(DrawerLogic::refreshForOpen(&refreshPolicy));
}

// Nickel hides the page when it pops it or shows another tab over it.
void DrawerView::hideEvent(QHideEvent *event)
{
    OpenPopups::closeAll("drawer view hidden");
    QWidget::hideEvent(event);
}

void DrawerView::resizeEvent(QResizeEvent *event)
{
    QWidget::resizeEvent(event);
    fitGridRows();
}

void DrawerView::applyStatusBarInset()
{
    int inset = StatusBarInset::topInsetFor(this);
    rootLayout->setContentsMargins(SIDE_GUTTER, inset, SIDE_GUTTER, 0);
    rootLayout->activate();
    DebugLog::write("drawer view: top inset %d", inset);
}

void DrawerView::logLayoutGeometry()
{
    StatusBarInset::logGeometry("after first layout", this, header);
}

const DrawerItem *DrawerView::itemForSender() const
{
    QObject *pressed = sender();
    int first = DrawerLogic::firstIndexOnPage(currentPage, perPage());
    for (int i = 0; i < pageWidgets.size(); i++) {
        if (pageWidgets[i] == pressed && first + i < shownIndices.size()) {
            return &items[shownIndices[first + i]];
        }
    }
    return NULL;
}

void DrawerView::onItemTapped()
{
    const DrawerItem *item = isInputBlocked ? NULL : itemForSender();
    if (item) {
        emit itemTapped(*item);
    }
}

void DrawerView::onItemLongPressed()
{
    const DrawerItem *item = isInputBlocked ? NULL : itemForSender();
    QWidget *pressed = qobject_cast<QWidget *>(sender());
    if (item && pressed) {
        emit itemLongPressed(*item, pressed->mapToGlobal(pressed->rect().center()));
    }
}

void DrawerView::keyPressEvent(QKeyEvent *event)
{
    DrawerLogic::PageKey key = DrawerLogic::pageKeyFor(event->key());
    if (key == DrawerLogic::PAGE_KEY_NONE) {
        QWidget::keyPressEvent(event);
        return;
    }
    event->accept();
    if (OpenPopups::isAnyOpen()) {
        // The key closes the popup instead of paging under it.
        OpenPopups::closeAll("page key");
        return;
    }
    if (isInputBlocked) {
        DebugLog::write("drawer: page key ignored while an action is pending");
        return;
    }
    int target = DrawerLogic::pageAfterKey(currentPage, key, shownIndices.size(), perPage());
    DebugLog::write("drawer: key 0x%x page %d -> %d", event->key(), currentPage + 1, target + 1);
    if (target != currentPage) {
        goToPage(target);
    }
}
