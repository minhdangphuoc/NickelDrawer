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
#include "screen_logic.h"
#include "system_icon.h"
#include "text_meter.h"
#include "tweaks_logic.h"

// Rule and dividers: the row line grey (Nickel's TouchMenu::separator and Wishlist's divider).
// Cells paint themselves; the message is body text and the page label a muted caption.
static const char *const PANEL_STYLE =
    "#tweaksPanel { background: " COPPERUI_GREY_PAPER_WHITE "; }"
    "QLabel { color: " COPPERUI_GREY_INK_BLACK "; background: transparent; }"
    "#tweaksRule { background: " COPPERUI_GREY_LINE_ROW "; }"
    "#tweaksMessage { font-family: @BODY_FAMILY@; font-size: " COPPERUI_PX_TYPE_BODY_SIZE "; }"
    "#tweaksPage { font-family: @BODY_FAMILY@; font-size: " COPPERUI_PX_TYPE_CAPTION_SIZE "; color: "
    COPPERUI_GREY_INK_MUTED "; }";
static const QColor DIVIDER_COLOR(copperui::v1::LINE_ROW, copperui::v1::LINE_ROW, copperui::v1::LINE_ROW);

static QLabel *createLabel(const QString &text, const char *objectName, QWidget *parent)
{
    QLabel *label = new QLabel(text, parent);
    label->setObjectName(QString::fromLatin1(objectName));
    label->setTextFormat(Qt::PlainText);
    label->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    return label;
}

// ---- Cell ----

static QColor grey(int level)
{
    return QColor(level, level, level);
}

TweakCell::TweakCell(const TweakItem &item, QWidget *parent) : PressArea(parent), shown(item)
{
    setObjectName(QStringLiteral("tweakCell"));
    cellGeometry = TweaksLogic::CellLayout();
    if (!item.isTappable()) {
        setPressHighlight(HIGHLIGHT_NONE);
        setAcceptsPresses(false);
    }
}

void TweakCell::showValue(const TweakItem &item)
{
    shown.isOn = item.isOn;
    shown.infoText = item.infoText;
    measureText();
    update();
}

// The shared text-fit rule with the real font: wrapped or elided where the cell's geometry says,
// so nothing wraps left-aligned or overflows the fixed cell. Needs the cell's size.
void TweakCell::measureText()
{
    if (width() <= 0) {
        return;
    }
    using ScreenLogic::TextRun;
    TextRun labelRun = {ScreenLogic::ROW_TEXT_SIZE, ScreenLogic::FAMILY_SANS, false};
    TextRun valueRun = {ScreenLogic::CAPTION_SIZE, ScreenLogic::FAMILY_SANS, false};
    TextRun stateRun = {ScreenLogic::TOGGLE_STATE_SIZE, ScreenLogic::FAMILY_SANS, false};
    int labelWidth = TweaksLogic::labelWidth(shown.kind, width());
    if (shown.kind == TweaksLogic::KIND_INFO) {
        labelLines = QStringList(QtTextMeter::elideLine(shown.label, labelWidth, labelRun));
        valueLine = QtTextMeter::elideLine(shown.infoText, labelWidth, valueRun);
    } else {
        labelLines = QtTextMeter::wrapLines(shown.label, labelWidth, labelRun, TweaksLogic::maxLabelLines(shown.kind));
    }
    if (shown.kind == TweaksLogic::KIND_TOGGLE) {
        QString word = FallbackStrings::text(shown.isOn ? FallbackStrings::SCREEN_ON : FallbackStrings::SCREEN_OFF);
        stateWord = QtTextMeter::elideLine(word.toUpper(), TweaksLogic::TOGGLE_COLUMN_WIDTH, stateRun);
    }
    cellGeometry = TweaksLogic::cellLayout(shown.kind, width(), height(), labelLines.size());
}

void TweakCell::resizeEvent(QResizeEvent *event)
{
    PressArea::resizeEvent(event);
    measureText();
}

static QRect toQRect(const TweaksLogic::Rect &rect)
{
    return QRect(rect.x, rect.y, rect.width, rect.height);
}

void TweakCell::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), grey(ScreenLogic::PAPER_WHITE));
    painter.setPen(grey(ScreenLogic::INK_BLACK));
    painter.setFont(NickelFonts::font(NickelFonts::ROLE_BODY, ScreenLogic::ROW_TEXT_SIZE, false));
    int labelAlignment = (cellGeometry.isCentred ? Qt::AlignHCenter : Qt::AlignLeft) | Qt::AlignVCenter | Qt::TextSingleLine;
    QRect label = toQRect(cellGeometry.label);
    for (int i = 0; i < labelLines.size(); i++) {
        QRect line(label.x(), label.y() + i * TweaksLogic::LABEL_LINE_HEIGHT, label.width(), TweaksLogic::LABEL_LINE_HEIGHT);
        painter.drawText(line, labelAlignment, labelLines[i]);
    }
    if (shown.kind == TweaksLogic::KIND_INFO) {
        painter.setFont(NickelFonts::font(NickelFonts::ROLE_BODY, ScreenLogic::CAPTION_SIZE, false));
        painter.setPen(grey(ScreenLogic::INK_MUTED));
        painter.drawText(toQRect(cellGeometry.value), Qt::AlignLeft | Qt::AlignVCenter | Qt::TextSingleLine, valueLine);
    }
    if (shown.kind == TweaksLogic::KIND_TOGGLE) {
        // Nickel's switch image (never drawn by us); without it the state word alone shows the state.
        SystemIcon::Id image = shown.isOn ? SystemIcon::SWITCH_ON : SystemIcon::SWITCH_OFF;
        SystemIcon::drawCentred(&painter, toQRect(cellGeometry.switchImage), SystemIcon::icon(image).normal);
        painter.setFont(NickelFonts::font(NickelFonts::ROLE_BODY, ScreenLogic::TOGGLE_STATE_SIZE, false));
        painter.drawText(toQRect(cellGeometry.stateWord), Qt::AlignHCenter | Qt::AlignVCenter | Qt::TextSingleLine, stateWord);
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
    rule->setFixedHeight(TweaksLogic::RULE_HEIGHT);
    grid = new QWidget(this);
    grid->installEventFilter(this);
    messageLabel = createLabel(QString(), "tweaksMessage", this);
    messageLabel->setAlignment(Qt::AlignCenter);
    messageLabel->setFixedHeight(TweaksLogic::MESSAGE_HEIGHT);
    pageLabel = createLabel(QString(), "tweaksPage", this);
    pageLabel->setAlignment(Qt::AlignCenter);
    pageLabel->setFixedHeight(TweaksLogic::PAGER_LINE_HEIGHT);

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
    // Wrapped by the shared rule, centred, two lines at most: never clipped or left-aligned.
    ScreenLogic::TextRun run = {ScreenLogic::BODY_SIZE, ScreenLogic::FAMILY_SANS, false};
    int width = TweaksLogic::boxInnerWidth() - 2 * TweaksLogic::CELL_PADDING;
    messageLabel->setText(QtTextMeter::wrapLines(text, width, run, TweaksLogic::LABEL_MAX_LINES).join(QStringLiteral("\n")));
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
    grid->setFixedHeight(TweaksLogic::gridHeight(itemsOnFirstPage, TweaksLogic::CELL_HEIGHT, TweaksLogic::DIVIDER_WIDTH));
    placeCells();
    updatePageLabel();
    grid->update();
}

void TweaksPanel::placeCells()
{
    for (int i = 0; i < cells.size(); i++) {
        TweaksLogic::Rect cell = TweaksLogic::cellRect(i, grid->width(), TweaksLogic::CELL_HEIGHT, TweaksLogic::DIVIDER_WIDTH);
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
            TweaksLogic::Rect left = TweaksLogic::cellRect(row * TweaksLogic::COLUMN_COUNT, grid->width(),
                                                           TweaksLogic::CELL_HEIGHT, TweaksLogic::DIVIDER_WIDTH);
            if (TweaksLogic::rowHasRightCell(row, cells.size())) {
                painter.fillRect(left.x + left.width, left.y, TweaksLogic::DIVIDER_WIDTH, left.height, DIVIDER_COLOR);
            }
            painter.fillRect(0, left.y + left.height, grid->width(), TweaksLogic::DIVIDER_WIDTH, DIVIDER_COLOR);
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
