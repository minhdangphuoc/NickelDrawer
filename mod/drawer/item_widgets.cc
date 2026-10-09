#include "item_widgets.h"

#include <QHBoxLayout>
#include <QLabel>
#include <QPainter>
#include <QVBoxLayout>

#include "painted_marks.h"

// Sizes in device pixels (docs/spec/ui-kit.md §2).
static const int ROW_HEIGHT = 152;
static const int ROW_ICON_SIDE = 52;
static const int SPACING = 24;
static const int TILE_MARK_SIDE = 96;
static const int TILE_PADDING = 24;
static const int OUTLINE_WIDTH = 2;
static const int SECTION_LABEL_HEIGHT = 64;

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

void ItemWidget::createStatusAndDot()
{
    statusLabel = createLabel(QString(), "drawerItemStatus", this);
    statusLabel->setWordWrap(false);
    updateDot = new UpdateDot(this);
}

bool ItemWidget::applyLiveState(const DrawerItem &item)
{
    if (item.id != id || !statusLabel || !updateDot) {
        return false;
    }
    if (statusLabel->text() != item.status) {
        statusLabel->setText(item.status);
    }
    statusLabel->setVisible(!item.status.isEmpty());
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
        layout->addWidget(new AppMark(item.iconPath, item.name, ROW_ICON_SIDE, this), 0, Qt::AlignVCenter);
    }
    layout->addLayout(texts, 1);
    layout->addWidget(updateDot, 0, Qt::AlignVCenter);
    if (item.hasPrimaryAction()) {
        layout->addWidget(createLabel(QString::fromUtf8("›"), "drawerChevron", this), 0, Qt::AlignVCenter);
    }
    applyLiveState(item);
}

AppTile::AppTile(const DrawerItem &item, int side, QWidget *parent) : ItemWidget(item, parent)
{
    setObjectName(QStringLiteral("drawerTile"));
    setFixedSize(side, side);
    createStatusAndDot();
    statusLabel->setAlignment(Qt::AlignHCenter);

    QLabel *name = createLabel(item.name, "drawerItemName", this);
    name->setAlignment(Qt::AlignHCenter);
    name->setWordWrap(true);

    QVBoxLayout *layout = new QVBoxLayout(this);
    layout->setContentsMargins(TILE_PADDING, TILE_PADDING, TILE_PADDING, TILE_PADDING);
    layout->setSpacing(SPACING / 2);
    layout->addWidget(new AppMark(item.iconPath, item.name, TILE_MARK_SIDE, this), 0, Qt::AlignHCenter);
    layout->addStretch(1);
    layout->addWidget(statusLabel);
    layout->addWidget(name);
    applyLiveState(item);
}

void AppTile::paintEvent(QPaintEvent *event)
{
    PressArea::paintEvent(event);
    QPainter painter(this);
    painter.setPen(QPen(Qt::black, OUTLINE_WIDTH));
    painter.setBrush(Qt::NoBrush);
    painter.drawRect(rect().adjusted(1, 1, -1, -1));
}

void AppTile::resizeEvent(QResizeEvent *event)
{
    PressArea::resizeEvent(event);
    updateDot->move(width() - TILE_PADDING / 2 - updateDot->width(), TILE_PADDING / 2);
}

QLabel *createSectionLabel(const QString &title, QWidget *parent)
{
    QLabel *label = createLabel(title, "drawerSectionLabel", parent);
    label->setFixedHeight(SECTION_LABEL_HEIGHT);
    label->setAlignment(Qt::AlignLeft | Qt::AlignBottom);
    return label;
}
