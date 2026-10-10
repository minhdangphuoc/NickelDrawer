#include "tweaks_panel.h"

#include <QEvent>
#include <QKeyEvent>
#include <QLabel>
#include <QPainter>
#include <QVBoxLayout>

#include "debug_log.h"
#include "drawer_logic.h"
#include "fallback_strings.h"
#include "nickel_bridge.h"
#include "nickel_fonts.h"
#include "tweaks_logic.h"

// Device pixels (Storm class; Nickel's More rows are 120-130 px). The modal's own metrics are
// "legacy" tokens until it moves to the dialog tokens (fix 3.2).
static const int CELL_HEIGHT = copperui::v1::LEGACY_TWEAKS_CELL_HEIGHT;
static const int DIVIDER_WIDTH = copperui::v1::STROKE_HAIRLINE;
static const int CELL_SIDE_PADDING = copperui::v1::LEGACY_TWEAKS_CELL_PADDING;
static const int MESSAGE_HEIGHT = copperui::v1::LEGACY_TWEAKS_MESSAGE_HEIGHT;
static const int PAGE_LABEL_HEIGHT = copperui::v1::LEGACY_TWEAKS_PAGE_LABEL_HEIGHT;

// Label: Georgia italic like Settings' rows (MenuTextItem[italics=true] in
// NickelTouchMenu.qss, N3ButtonLabel at 34 px on Storm). Rule and dividers: the row line
// grey (Nickel's TouchMenu::separator and Wishlist's divider).
static const char *const PANEL_STYLE =
    "#tweaksPanel { background: " COPPERUI_GREY_PAPER_WHITE "; }"
    "QLabel { color: " COPPERUI_GREY_INK_BLACK "; background: transparent; }"
    "#tweaksRule { background: " COPPERUI_GREY_LINE_ROW "; }"
    "#tweakLabel { font-family: @TITLE_FAMILY@; font-style: italic; font-size: " COPPERUI_PX_LEGACY_TWEAKS_LABEL_SIZE "; }"
    "#tweakValue { font-family: @BODY_FAMILY@; font-size: " COPPERUI_PX_LEGACY_TWEAKS_VALUE_SIZE "; }"
    "#tweaksMessage { font-family: @BODY_FAMILY@; font-size: " COPPERUI_PX_LEGACY_TWEAKS_MESSAGE_SIZE "; }"
    "#tweaksPage { font-family: @BODY_FAMILY@; font-size: " COPPERUI_PX_LEGACY_TWEAKS_VALUE_SIZE "; }";
static const QColor DIVIDER_COLOR(copperui::v1::LINE_ROW, copperui::v1::LINE_ROW, copperui::v1::LINE_ROW);

static QString valueText(const TweakItem &item)
{
    switch (item.kind) {
    case TweaksLogic::KIND_TOGGLE:
        return FallbackStrings::text(item.isOn ? FallbackStrings::SCREEN_ON : FallbackStrings::SCREEN_OFF);
    case TweaksLogic::KIND_INFO:
        return item.infoText;
    case TweaksLogic::KIND_ACTION:
        break;
    }
    return QString();
}

static QLabel *createLabel(const QString &text, const char *objectName, QWidget *parent)
{
    QLabel *label = new QLabel(text, parent);
    label->setObjectName(QString::fromLatin1(objectName));
    label->setTextFormat(Qt::PlainText);
    label->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    return label;
}

// ---- Cell ----

TweakCell::TweakCell(const TweakItem &item, QWidget *parent) : PressArea(parent), id(item.id), valueLabel(NULL)
{
    setObjectName(QStringLiteral("tweakCell"));
    setAttribute(Qt::WA_StyledBackground, true);
    QLabel *label = createLabel(item.label, "tweakLabel", this);
    label->setWordWrap(true); // long vi labels take two lines instead of clipping
    valueLabel = createLabel(QString(), "tweakValue", this);
    QVBoxLayout *layout = new QVBoxLayout(this);
    layout->setContentsMargins(CELL_SIDE_PADDING, 0, CELL_SIDE_PADDING, 0);
    layout->setSpacing(0);
    layout->addStretch(1);
    layout->addWidget(label);
    layout->addWidget(valueLabel);
    layout->addStretch(1);
    if (!item.isTappable()) {
        setPressHighlight(HIGHLIGHT_NONE);
        setAcceptsPresses(false);
    }
    showValue(item);
}

void TweakCell::showValue(const TweakItem &item)
{
    // A cell without a value keeps an empty value line, so every label sits at the same height.
    static const QChar NO_BREAK_SPACE(0x00A0);
    QString text = valueText(item);
    if (text.isEmpty()) {
        text = NO_BREAK_SPACE;
    }
    if (valueLabel->text() != text) {
        valueLabel->setText(text);
    }
}

// ---- Panel ----

TweaksPanel::TweaksPanel(QWidget *parent)
    : QWidget(parent), grid(NULL), messageLabel(NULL), pageLabel(NULL), currentPage(0), isInputEnabled(true)
{
    setObjectName(QStringLiteral("tweaksPanel"));
    setAttribute(Qt::WA_StyledBackground, true);
    setStyleSheet(NickelFonts::styleSheet(PANEL_STYLE));
    setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Fixed);

    QWidget *rule = new QWidget(this);
    rule->setObjectName(QStringLiteral("tweaksRule"));
    rule->setAttribute(Qt::WA_StyledBackground, true);
    rule->setFixedHeight(DIVIDER_WIDTH);
    grid = new QWidget(this);
    grid->installEventFilter(this);
    messageLabel = createLabel(QString(), "tweaksMessage", this);
    messageLabel->setAlignment(Qt::AlignCenter);
    messageLabel->setWordWrap(true);
    messageLabel->setFixedHeight(MESSAGE_HEIGHT);
    pageLabel = createLabel(QString(), "tweaksPage", this);
    pageLabel->setAlignment(Qt::AlignCenter);
    pageLabel->setFixedHeight(PAGE_LABEL_HEIGHT);

    QVBoxLayout *layout = new QVBoxLayout(this);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(0);
    layout->addWidget(rule);
    layout->addWidget(messageLabel);
    layout->addWidget(grid);
    layout->addWidget(pageLabel);
    showMessage(FallbackStrings::text(FallbackStrings::LOADING));
}

void TweaksPanel::showMessage(const QString &text)
{
    qDeleteAll(cells);
    cells.clear();
    allItems.clear();
    grid->hide();
    pageLabel->hide();
    messageLabel->setText(text);
    messageLabel->show();
}

void TweaksPanel::showItems(const QList<TweakItem> &items)
{
    allItems = items;
    messageLabel->hide();
    grid->show();
    showPage(0);
    DebugLog::write("tweaks panel: %d tweak(s), %d page(s)", allItems.size(),
                    DrawerLogic::pageCount(allItems.size(), itemsPerPage()));
}

int TweaksPanel::itemsPerPage() const
{
    return TweaksLogic::COLUMN_COUNT * TweaksLogic::ROWS_PER_PAGE;
}

void TweaksPanel::showPage(int page)
{
    qDeleteAll(cells);
    cells.clear();
    currentPage = DrawerLogic::clampPage(page, allItems.size(), itemsPerPage());
    int first = DrawerLogic::firstIndexOnPage(currentPage, itemsPerPage());
    int count = DrawerLogic::itemCountOnPage(currentPage, allItems.size(), itemsPerPage());
    for (int i = 0; i < count; i++) {
        TweakCell *cell = new TweakCell(allItems[first + i], grid);
        cell->setAcceptsPresses(isInputEnabled && allItems[first + i].isTappable());
        connect(cell, SIGNAL(tapped()), this, SLOT(onCellTapped()));
        cell->show();
        cells.append(cell);
    }
    // Every page as high as the first, so the modal does not jump between pages.
    int itemsOnFirstPage = DrawerLogic::itemCountOnPage(0, allItems.size(), itemsPerPage());
    grid->setFixedHeight(TweaksLogic::gridHeight(itemsOnFirstPage, CELL_HEIGHT, DIVIDER_WIDTH));
    placeCells();
    updatePageLabel();
    grid->update();
}

void TweaksPanel::placeCells()
{
    for (int i = 0; i < cells.size(); i++) {
        TweaksLogic::Rect cell = TweaksLogic::cellRect(i, grid->width(), CELL_HEIGHT, DIVIDER_WIDTH);
        cells[i]->setGeometry(cell.x, cell.y, cell.width, cell.height);
    }
}

void TweaksPanel::updatePageLabel()
{
    int pages = DrawerLogic::pageCount(allItems.size(), itemsPerPage());
    pageLabel->setText(QStringLiteral("%1 / %2").arg(currentPage + 1).arg(pages));
    pageLabel->setVisible(pages > 1);
}

void TweaksPanel::updateItem(const TweakItem &updated)
{
    for (int i = 0; i < allItems.size(); i++) {
        if (allItems[i].id == updated.id) {
            allItems[i] = updated;
        }
    }
    for (int i = 0; i < cells.size(); i++) {
        if (cells[i]->tweakId() == updated.id) {
            cells[i]->showValue(updated);
        }
    }
}

const TweakItem *TweaksPanel::item(const QString &tweakId) const
{
    for (int i = 0; i < allItems.size(); i++) {
        if (allItems[i].id == tweakId) {
            return &allItems[i];
        }
    }
    return NULL;
}

void TweaksPanel::setInputEnabled(bool isEnabled)
{
    isInputEnabled = isEnabled;
    for (int i = 0; i < cells.size(); i++) {
        const TweakItem *shown = item(cells[i]->tweakId());
        cells[i]->setAcceptsPresses(isEnabled && shown && shown->isTappable());
    }
}

bool TweaksPanel::handlePageKey(int qtKey)
{
    DrawerLogic::PageKey key = DrawerLogic::pageKeyFor(qtKey);
    if (key == DrawerLogic::PAGE_KEY_NONE) {
        return false;
    }
    int target = DrawerLogic::pageAfterKey(currentPage, key, allItems.size(), itemsPerPage());
    if (target != currentPage && isInputEnabled) {
        DebugLog::write("tweaks panel: page %d -> %d", currentPage + 1, target + 1);
        showPage(target);
        NickelBridge::requestFullRefresh(this);
    }
    return true;
}

void TweaksPanel::keyPressEvent(QKeyEvent *event)
{
    if (!handlePageKey(event->key())) {
        QWidget::keyPressEvent(event);
    }
}

// The grid: cells placed on resize, dividers painted under them.
bool TweaksPanel::eventFilter(QObject *watched, QEvent *event)
{
    if (watched != grid) {
        return QWidget::eventFilter(watched, event);
    }
    if (event->type() == QEvent::Resize) {
        placeCells();
    } else if (event->type() == QEvent::Paint) {
        QPainter painter(grid);
        painter.fillRect(grid->rect(), Qt::white);
        for (int row = 0; row < TweaksLogic::rowCount(cells.size()); row++) {
            TweaksLogic::Rect left = TweaksLogic::cellRect(row * TweaksLogic::COLUMN_COUNT, grid->width(), CELL_HEIGHT,
                                                           DIVIDER_WIDTH);
            if (TweaksLogic::rowHasRightCell(row, cells.size())) {
                painter.fillRect(left.x + left.width, left.y, DIVIDER_WIDTH, left.height, DIVIDER_COLOR);
            }
            painter.fillRect(0, left.y + left.height, grid->width(), DIVIDER_WIDTH, DIVIDER_COLOR);
        }
        return true;
    }
    return QWidget::eventFilter(watched, event);
}

void TweaksPanel::onCellTapped()
{
    QWidget *tappedCell = qobject_cast<QWidget *>(sender());
    for (int i = 0; i < cells.size(); i++) {
        if (cells[i] == tappedCell) {
            DebugLog::write("tweaks panel: tap on \"%s\"", qPrintable(cells[i]->tweakId()));
            emit tweakTapped(cells[i]->tweakId());
            return;
        }
    }
}
