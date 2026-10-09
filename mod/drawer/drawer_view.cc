#include "drawer_view.h"

#include <QGridLayout>
#include <QHBoxLayout>
#include <QKeyEvent>
#include <QLabel>
#include <QTimer>
#include <QVBoxLayout>

#include <NickelHook.h>

#include "fallback_strings.h"
#include "item_widgets.h"
#include "nickel_bridge.h"
#include "painted_marks.h"
#include "view_settings.h"

// Sizes in device pixels (docs/spec/ui-kit.md §2).
static const int SIDE_GUTTER = 72;
static const int HEADER_HEIGHT = 120;
static const int FOOTER_BUTTON_SIDE = 120;
static const int SPACING = 24;
static const int CONTENT_WIDTH = 1120;
static const int GRID_TILE_SIDE =
    (CONTENT_WIDTH - (DrawerLogic::GRID_COLUMNS - 1) * SPACING) / DrawerLogic::GRID_COLUMNS;

// Black on white, no grey fills; separators 2 px (ui-kit §1, §2).
static const char *const VIEW_STYLE =
    "DrawerView { background: white; }"
    "QLabel { color: black; background: transparent; }"
    "#drawerTitle { font-family: serif; font-size: 46px; }"
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
    "#drawerEmptyTitle { font-family: serif; font-size: 40px; }"
    "#drawerEmptyText { font-size: 30px; }";

static QLabel *createTextLabel(const QString &text, const char *objectName, QWidget *parent)
{
    QLabel *label = new QLabel(text, parent);
    label->setObjectName(QString::fromLatin1(objectName));
    return label;
}

DrawerView::DrawerView()
    : QWidget(NULL), titleLabel(NULL), listToggle(NULL), gridToggle(NULL), pageArea(NULL), pageLayout(NULL),
      hintLabel(NULL), pageIndicator(NULL), viewMode(ViewSettings::load()), currentPage(0), isInputBlocked(false),
      hasModel(false)
{
    DrawerLogic::resetRefreshPolicy(&refreshPolicy);
    setObjectName(QStringLiteral("nickelDrawerView"));
    setStyleSheet(QString::fromLatin1(VIEW_STYLE));
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
    hintLabel = createTextLabel(QString(), "drawerHint", this);
    hintLabel->setAlignment(Qt::AlignHCenter);
    hintLabel->setWordWrap(true);

    QVBoxLayout *layout = new QVBoxLayout(this);
    layout->setContentsMargins(SIDE_GUTTER, 0, SIDE_GUTTER, 0);
    layout->setSpacing(0);
    layout->addWidget(createHeader());
    layout->addWidget(pageArea, 1);
    layout->addWidget(hintLabel);
    layout->addWidget(createFooter());

    setViewMode(viewMode);
    renderPage(); // "Loading…" until the session delivers the model
}

PressArea *DrawerView::createArrow(const char *glyph, const char *slotOrSignal, QWidget *parent)
{
    PressArea *button = new PressArea(parent);
    button->setFixedSize(FOOTER_BUTTON_SIDE, FOOTER_BUTTON_SIDE);
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

QWidget *DrawerView::createHeader()
{
    QWidget *header = new QWidget(this);
    header->setFixedHeight(HEADER_HEIGHT);
    titleLabel = createTextLabel(FallbackStrings::text(FallbackStrings::APPS_TITLE), "drawerTitle", header);
    listToggle = new ViewToggleIcon(ViewToggleIcon::SHAPE_LIST, header);
    gridToggle = new ViewToggleIcon(ViewToggleIcon::SHAPE_GRID, header);
    connect(listToggle, SIGNAL(tapped()), this, SLOT(onListToggleTapped()));
    connect(gridToggle, SIGNAL(tapped()), this, SLOT(onGridToggleTapped()));
    pressables << listToggle << gridToggle;

    QHBoxLayout *layout = new QHBoxLayout(header);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(0);
    layout->addWidget(createArrow("‹", SIGNAL(backRequested()), header));
    layout->addWidget(titleLabel, 1);
    layout->addWidget(listToggle);
    layout->addWidget(gridToggle);
    return header;
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
    titleLabel->setText(model.title.isEmpty() ? FallbackStrings::text(FallbackStrings::APPS_TITLE) : model.title);
    hint = model.hint.isEmpty() ? FallbackStrings::text(FallbackStrings::LONG_PRESS_HINT) : model.hint;
    hintLabel->setText(hint);
    flatten(model);
    if (isStatusOnlyChange) {
        applyLiveStates();
        scheduleRefresh(DrawerLogic::refreshForStatusChange(&refreshPolicy));
        return;
    }
    currentPage = DrawerLogic::clampPage(currentPage, items.size(), DrawerLogic::itemsPerPage(viewMode));
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
    int first = DrawerLogic::firstIndexOnPage(currentPage, DrawerLogic::itemsPerPage(viewMode));
    for (int i = 0; i < pageWidgets.size() && first + i < items.size(); i++) {
        pageWidgets[i]->applyLiveState(items[first + i]);
    }
}

void DrawerView::setInputBlocked(bool isBlocked)
{
    isInputBlocked = isBlocked;
    for (int i = 0; i < pressables.size(); i++) {
        pressables[i]->setAcceptsPresses(!isBlocked);
    }
    for (int i = 0; i < pageWidgets.size(); i++) {
        pageWidgets[i]->setAcceptsPresses(!isBlocked);
    }
}

void DrawerView::setViewMode(DrawerLogic::ViewMode mode)
{
    int firstVisible = DrawerLogic::firstIndexOnPage(currentPage, DrawerLogic::itemsPerPage(viewMode));
    viewMode = mode;
    listToggle->setActive(mode == DrawerLogic::VIEW_LIST);
    gridToggle->setActive(mode == DrawerLogic::VIEW_GRID);
    currentPage = DrawerLogic::pageContainingItem(firstVisible, DrawerLogic::itemsPerPage(mode));
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
    goToPage(DrawerLogic::pageAfterKey(currentPage, DrawerLogic::PAGE_KEY_PREVIOUS, items.size(),
                                       DrawerLogic::itemsPerPage(viewMode)));
}

void DrawerView::onNextTapped()
{
    goToPage(DrawerLogic::pageAfterKey(currentPage, DrawerLogic::PAGE_KEY_NEXT, items.size(),
                                       DrawerLogic::itemsPerPage(viewMode)));
}

void DrawerView::goToPage(int page)
{
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
        } else if (child->layout()) {
            // Grid layouts hold the tiles; the tiles are children of pageArea.
            QLayout *nested = child->layout();
            while (QLayoutItem *nestedChild = nested->takeAt(0)) {
                if (nestedChild->widget()) {
                    nestedChild->widget()->hide();
                    nestedChild->widget()->deleteLater();
                }
                delete nestedChild;
            }
        }
        delete child;
    }
}

void DrawerView::renderPage()
{
    clearPage();
    int perPage = DrawerLogic::itemsPerPage(viewMode);
    int first = DrawerLogic::firstIndexOnPage(currentPage, perPage);
    int count = DrawerLogic::itemCountOnPage(currentPage, items.size(), perPage);
    if (items.isEmpty()) {
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
        int index = first + i;
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

void DrawerView::renderGridPage(int first, int count)
{
    QGridLayout *grid = new QGridLayout();
    grid->setContentsMargins(0, SPACING, 0, 0);
    grid->setSpacing(SPACING);
    for (int i = 0; i < count; i++) {
        AppTile *tile = new AppTile(items[first + i], GRID_TILE_SIDE, pageArea);
        connect(tile, SIGNAL(tapped()), this, SLOT(onItemTapped()));
        connect(tile, SIGNAL(longPressed()), this, SLOT(onItemLongPressed()));
        grid->addWidget(tile, i / DrawerLogic::GRID_COLUMNS, i % DrawerLogic::GRID_COLUMNS);
        pageWidgets.append(tile);
    }
    pageLayout->addLayout(grid);
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
    int pages = DrawerLogic::pageCount(items.size(), DrawerLogic::itemsPerPage(viewMode));
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
    setFocus(Qt::OtherFocusReason);
    scheduleRefresh(DrawerLogic::refreshForOpen(&refreshPolicy));
}

const DrawerItem *DrawerView::itemForSender() const
{
    QObject *pressed = sender();
    int first = DrawerLogic::firstIndexOnPage(currentPage, DrawerLogic::itemsPerPage(viewMode));
    for (int i = 0; i < pageWidgets.size(); i++) {
        if (pageWidgets[i] == pressed && first + i < items.size()) {
            return &items[first + i];
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
    if (item) {
        emit itemLongPressed(*item);
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
    if (isInputBlocked) {
        nh_log("drawer: page key ignored while a dialog or action is pending");
        return;
    }
    int perPage = DrawerLogic::itemsPerPage(viewMode);
    int target = DrawerLogic::pageAfterKey(currentPage, key, items.size(), perPage);
    nh_log("drawer: key 0x%x page %d -> %d", event->key(), currentPage + 1, target + 1);
    if (target != currentPage) {
        goToPage(target);
    }
}
