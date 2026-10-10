#include "item_widgets.h"

#include <QHBoxLayout>
#include <QLabel>
#include <QPainter>
#include <QVBoxLayout>

#include <QVector>

#include "drawer_logic.h"
#include "nickel_fonts.h"
#include "painted_marks.h"
#include "text_meter.h"

// Sizes in device pixels: the CopperUI tokens (docs/copperui-tokens.md).
static const int ROW_HEIGHT = copperui::v1::ROW_DRAWER;
static const int ROW_ICON_SIDE = copperui::v1::LEGACY_DRAWER_ROW_ICON; // icon.drawerRow (64) in fix 2.9
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

ItemWidget::ItemWidget(const DrawerItem &item, QWidget *parent)
    : PressArea(parent), statusLabel(NULL), updateDot(NULL), id(item.id)
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
        if (statusLabel->text() != item.status) {
            statusLabel->setText(item.status);
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
    texts->addWidget(createLabel(item.name, item.isProblem ? "drawerProblemName" : "drawerItemName", this));
    texts->addWidget(statusLabel);
    texts->addStretch(1);
    statusLabel->setWordWrap(item.isProblem); // problem messages may need two lines

    QHBoxLayout *layout = new QHBoxLayout(this);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(SPACING);
    if (!item.isProblem) {
        layout->addWidget(new AppMark(item.iconPath, ROW_ICON_SIDE, this), 0, Qt::AlignVCenter);
    }
    layout->addLayout(texts, 1);
    layout->addWidget(updateDot, 0, Qt::AlignVCenter);
    if (item.hasPrimaryAction()) {
        layout->addWidget(createLabel(QString::fromUtf8("›"), "drawerChevron", this), 0, Qt::AlignVCenter);
    }
    applyLiveState(item);
}

// The shared wrap rule (screen_logic.h): at most maxLines lines, the last one elided.
static QStringList wrapName(const QString &name, int width, int maxLines)
{
    QByteArray utf8 = name.toUtf8();
    QVector<char> out(utf8.size() * 2 + 16);
    ScreenLogic::TextRun run = {TILE_NAME_SIZE, ScreenLogic::FAMILY_SANS, false};
    int count = ScreenLogic::wrapText(utf8.constData(), width, run, QtTextMeter::meter(), maxLines, out.data(), out.size());
    return count == 0 ? QStringList() : QString::fromUtf8(out.constData()).split(QLatin1Char('\n'));
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
    QLabel *label = createLabel(title, "drawerSectionLabel", parent);
    label->setFixedHeight(SECTION_LABEL_HEIGHT);
    label->setAlignment(Qt::AlignLeft | Qt::AlignBottom);
    return label;
}
