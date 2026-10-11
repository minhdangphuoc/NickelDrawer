#include "item_widgets.h"

#include <QHBoxLayout>
#include <QLabel>
#include <QPainter>
#include <QVBoxLayout>

#include "drawer_logic.h"
#include "nickel_fonts.h"
#include "painted_marks.h"
#include "system_icon.h"
#include "screen_logic.h"
#include "text_meter.h"

// Sizes in device pixels: the CopperUI tokens (docs/copperui-tokens.md).
static const int ROW_HEIGHT = copperui::v1::ROW_DRAWER;
static const int ROW_ICON_SIDE = copperui::v1::ICON_DRAWER_ROW; // drawn 1:1
static const int CHEVRON_SIDE = copperui::v1::ICON_CHEVRON;
static const int SPACING = copperui::v1::RHYTHM;
static const int TILE_NAME_SIZE = copperui::v1::TYPE_GRID_NAME_SIZE;
static const int TILE_NAME_INSET = copperui::v1::DRAWER_GRID_NAME_INSET;
static const int SECTION_LABEL_HEIGHT = copperui::v1::SECTION_HEIGHT;

static QLabel *createLabel(const QString &text, const char *objectName, QWidget *parent)
{
    QLabel *label = new QLabel(text, parent);
    label->setObjectName(QString::fromLatin1(objectName));
    label->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    return label;
}

// One line for a fixed-size label: the text, or its measured prefix and "…".
static QString elidedLine(const QString &text, int width, int size, bool isBold)
{
    ScreenLogic::TextRun run = {size, ScreenLogic::FAMILY_SANS, isBold};
    return QtTextMeter::elideLine(text, width, run);
}

ItemWidget::ItemWidget(const DrawerItem &item, QWidget *parent)
    : PressArea(parent), statusLabel(NULL), updateDot(NULL), statusWidth(0), id(item.id)
{
    setAttribute(Qt::WA_StyledBackground, true);
}

void ItemWidget::createDot()
{
    updateDot = new UpdateDot(this);
}

void ItemWidget::createStatusAndDot()
{
    statusLabel = createLabel(QString(), "drawerItemStatus", this);
    statusLabel->setWordWrap(false);
    createDot();
}

bool ItemWidget::applyLiveState(const DrawerItem &item)
{
    if (item.id != id || !updateDot) {
        return false;
    }
    if (statusLabel) { // grid cells have no status line
        QString status = elidedLine(item.status, statusWidth, ScreenLogic::CAPTION_SIZE, false);
        if (statusLabel->text() != status) {
            statusLabel->setText(status);
        }
        statusLabel->setVisible(!item.status.isEmpty());
    }
    updateDot->setVisible(item.hasUpdate);
    return true;
}

AppRow::AppRow(const DrawerItem &item, QWidget *parent) : ItemWidget(item, parent)
{
    setObjectName(QStringLiteral("drawerRow"));
    setFixedHeight(ROW_HEIGHT);
    createStatusAndDot();

    QVBoxLayout *texts = new QVBoxLayout();
    texts->setSpacing(0);
    texts->addStretch(1);
    int textWidth = DrawerLogic::rowTextWidth(!item.isProblem, item.hasPrimaryAction());
    statusWidth = textWidth;
    QString name = elidedLine(item.name, textWidth, ScreenLogic::ROW_TEXT_SIZE, item.isProblem);
    texts->addWidget(createLabel(name, item.isProblem ? "drawerProblemName" : "drawerItemName", this));
    texts->addWidget(statusLabel);
    texts->addStretch(1);

    QHBoxLayout *layout = new QHBoxLayout(this);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(SPACING);
    if (!item.isProblem) {
        layout->addWidget(new AppMark(item.iconPath, ROW_ICON_SIDE, this), 0, Qt::AlignVCenter);
    }
    layout->addLayout(texts, 1);
    layout->addWidget(updateDot, 0, Qt::AlignVCenter);
    if (item.hasPrimaryAction()) {
        layout->addWidget(new IconMark(SystemIcon::CHEVRON_RIGHT, CHEVRON_SIDE, QString::fromUtf8("›"), this), 0,
                          Qt::AlignVCenter);
    }
    applyLiveState(item);
}

// The shared wrap rule (screen_logic.h): at most maxLines lines, the last one elided.
static QStringList wrapName(const QString &name, int width, int maxLines)
{
    ScreenLogic::TextRun run = {TILE_NAME_SIZE, ScreenLogic::FAMILY_SANS, false};
    return QtTextMeter::wrapLines(name, width, run, maxLines);
}

AppTile::AppTile(const DrawerItem &item, int width, int height, QWidget *parent)
    : ItemWidget(item, parent), nameLines(), nameTop(DrawerLogic::gridNameTop(height))
{
    setObjectName(QStringLiteral("drawerTile"));
    setFixedSize(width, height);
    createDot();

    nameLines = wrapName(item.name, width - 2 * TILE_NAME_INSET, DrawerLogic::GRID_NAME_LINES);

    int iconLeft = (width - DrawerLogic::GRID_ICON_SIDE) / 2;
    int iconTop = DrawerLogic::gridIconTop(height);
    AppMark *mark = new AppMark(item.iconPath, DrawerLogic::GRID_ICON_SIDE, this);
    mark->move(iconLeft, iconTop);
    updateDot->move(iconLeft + DrawerLogic::GRID_ICON_SIDE - updateDot->width(), iconTop);
    applyLiveState(item);
}

// The name is painted (not a QLabel) because QLabel word-wraps but cannot elide
// the last line. The press overlay inverts the cell, text included.
void AppTile::paintEvent(QPaintEvent *event)
{
    PressArea::paintEvent(event);
    QPainter painter(this);
    painter.setFont(NickelFonts::font(NickelFonts::ROLE_BODY, TILE_NAME_SIZE, false));
    painter.setPen(Qt::black);
    for (int i = 0; i < nameLines.size(); i++) {
        QRect line(TILE_NAME_INSET, nameTop + i * DrawerLogic::GRID_NAME_LINE_HEIGHT, width() - 2 * TILE_NAME_INSET,
                   DrawerLogic::GRID_NAME_LINE_HEIGHT);
        painter.drawText(line, Qt::AlignHCenter | Qt::AlignTop, nameLines[i]);
    }
}

QLabel *createSectionLabel(const QString &title, QWidget *parent)
{
    // Uppercase first: capitals are wider than the lower case they replace.
    QLabel *label = createLabel(elidedLine(title.toUpper(), ScreenLogic::CONTENT_WIDTH, ScreenLogic::SECTION_TEXT_SIZE, true),
                                "drawerSectionLabel", parent);
    label->setFixedHeight(SECTION_LABEL_HEIGHT);
    label->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    return label;
}
