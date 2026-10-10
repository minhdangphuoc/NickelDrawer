#include "plain_dialog.h"

#include <QApplication>
#include <QBoxLayout>
#include <QDesktopWidget>
#include <QLabel>
#include <QPainter>
#include <QTimer>

#include "copperui/tokens.h"
#include "debug_log.h"
#include "drawer_logic.h"
#include "nickel_fonts.h"
#include "press_area.h"

// The dialog's width, text size and button border are "legacy" tokens until PlainDialog moves
// to the dialog tokens (fix 2.8).
static const int DIALOG_WIDTH = copperui::v1::LEGACY_DIALOG_WIDTH;
static const int MENU_WIDTH = copperui::v1::MENU_WIDTH;
static const int PADDING = copperui::v1::SPACE_DIALOG_PADDING;
static const int MENU_SIDE_PADDING = copperui::v1::SPACE_MENU_SIDE;
static const int SPACING = copperui::v1::RHYTHM;
static const int BUTTON_HEIGHT = copperui::v1::BUTTON_HEIGHT;
static const int MENU_ROW_HEIGHT = copperui::v1::TARGET_MIN;
static const int DIVIDER_HEIGHT = copperui::v1::STROKE_HAIRLINE;
static const int BOX_BORDER = copperui::v1::STROKE_BUTTON;

// Menu look from Nickel's NickelTouchMenu.qss: 3 px black border, white box,
// 1 px separators in the row line grey; the title row bold.
static const char *const DIALOG_STYLE =
    "QWidget { font-family: @BODY_FAMILY@; }"
    "QLabel { color: " COPPERUI_GREY_INK_BLACK "; background: transparent; }"
    "#plainDialogBox { background: " COPPERUI_GREY_PAPER_WHITE "; border: " COPPERUI_PX_STROKE_BUTTON " solid "
    COPPERUI_GREY_INK_BLACK "; }"
    "#plainDialogTitle { font-family: @TITLE_FAMILY@; font-size: " COPPERUI_PX_TYPE_DIALOG_TITLE_SIZE "; font-weight: bold; }"
    "#plainDialogText { font-size: " COPPERUI_PX_LEGACY_DIALOG_TEXT_SIZE "; }"
    "#plainDialogButton { border: " COPPERUI_PX_LEGACY_DIALOG_BUTTON_BORDER " solid " COPPERUI_GREY_INK_BLACK
    "; background: " COPPERUI_GREY_PAPER_WHITE "; }"
    "#plainDialogButtonLabel { font-size: " COPPERUI_PX_LEGACY_DIALOG_TEXT_SIZE "; }"
    "#plainMenuTitle { font-size: " COPPERUI_PX_TYPE_MENU_SIZE "; font-weight: bold; }"
    "#plainMenuRow { background: " COPPERUI_GREY_PAPER_WHITE "; }"
    "#plainMenuRowLabel { font-size: " COPPERUI_PX_TYPE_MENU_SIZE "; }"
    "#plainMenuDivider { background: " COPPERUI_GREY_LINE_ROW "; }";

static QLabel *createLabel(const QString &text, const char *objectName, QWidget *parent)
{
    QLabel *label = new QLabel(text, parent);
    label->setObjectName(QString::fromLatin1(objectName));
    label->setWordWrap(true); // long vi labels wrap instead of clipping (ui-kit §4)
    return label;
}

PlainDialog::PlainDialog(const QString &title, const QString &text, const QList<DialogChoice> &choices, Layout layout,
                         OutsideTap outsideTap)
    : QDialog(NULL, Qt::Dialog | Qt::FramelessWindowHint), outsideArea(NULL), box(NULL), boxLayout(NULL),
      outsideTapRule(outsideTap),
      hasAnswered(false)
{
    setAttribute(Qt::WA_DeleteOnClose, true);
    setStyleSheet(NickelFonts::styleSheet(DIALOG_STYLE));

    // First child, so it lies under the box: every tap outside the box lands here.
    PressArea *outside = new PressArea(this);
    outside->setObjectName(QStringLiteral("plainDialogOutside"));
    outside->setPressHighlight(PressArea::HIGHLIGHT_NONE);
    connect(outside, SIGNAL(tapped()), this, SLOT(onOutsideTapped()));
    outsideArea = outside;
    QVBoxLayout *screenLayout = new QVBoxLayout(this);
    screenLayout->setContentsMargins(0, 0, 0, 0);

    box = new QWidget(this);
    box->setObjectName(QStringLiteral("plainDialogBox"));
    box->setAttribute(Qt::WA_StyledBackground, true);
    box->setFixedWidth(layout == MENU_ROWS ? MENU_WIDTH : DIALOG_WIDTH);
    boxLayout = new QVBoxLayout(box);
    screenLayout->addWidget(box, 0, Qt::AlignCenter);
    if (layout == MENU_ROWS) {
        addMenuContent(title, choices);
    } else {
        addMessageContent(title, text, choices);
    }

    connect(this, SIGNAL(rejected()), this, SLOT(onDismissed()));
    // ui-kit §3.15: a dialog left alone for 120 s counts as cancel.
    QTimer::singleShot(DrawerLogic::CONFIRM_TIMEOUT_MILLISECONDS, this, SLOT(reject()));
}

void PlainDialog::addMessageContent(const QString &title, const QString &text, const QList<DialogChoice> &choices)
{
    boxLayout->setContentsMargins(PADDING, PADDING, PADDING, PADDING);
    boxLayout->setSpacing(SPACING);
    if (!title.isEmpty()) {
        boxLayout->addWidget(createLabel(title, "plainDialogTitle", box));
    }
    if (!text.isEmpty()) {
        QLabel *textLabel = createLabel(text, "plainDialogText", box);
        textLabel->setTextFormat(Qt::PlainText);
        boxLayout->addWidget(textLabel);
    }
    QHBoxLayout *buttonLayout = new QHBoxLayout();
    buttonLayout->setSpacing(SPACING);
    for (int i = 0; i < choices.size(); i++) {
        buttonLayout->addWidget(createButton(choices[i]));
    }
    boxLayout->addLayout(buttonLayout);
}

void PlainDialog::addMenuContent(const QString &title, const QList<DialogChoice> &choices)
{
    boxLayout->setContentsMargins(BOX_BORDER, BOX_BORDER, BOX_BORDER, BOX_BORDER); // inside the border
    boxLayout->setSpacing(0);
    QLabel *titleRow = createLabel(title, "plainMenuTitle", box);
    titleRow->setFixedHeight(MENU_ROW_HEIGHT);
    titleRow->setContentsMargins(MENU_SIDE_PADDING, 0, MENU_SIDE_PADDING, 0);
    boxLayout->addWidget(titleRow);
    for (int i = 0; i < choices.size(); i++) {
        boxLayout->addWidget(createDivider());
        boxLayout->addWidget(createMenuRow(choices[i]));
    }
}

QWidget *PlainDialog::createButton(const DialogChoice &choice)
{
    PressArea *button = new PressArea(box);
    button->setObjectName(QStringLiteral("plainDialogButton"));
    button->setAttribute(Qt::WA_StyledBackground, true);
    button->setMinimumHeight(BUTTON_HEIGHT);
    QLabel *label = createLabel(choice.label, "plainDialogButtonLabel", button);
    label->setAlignment(Qt::AlignCenter);
    label->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    QHBoxLayout *layout = new QHBoxLayout(button);
    layout->addWidget(label);
    connect(button, SIGNAL(tapped()), this, SLOT(onChoiceTapped()));
    buttons.append(button);
    buttonIds.append(choice.id);
    return button;
}

// A plain text row, no box of its own; pressed, it inverts (PressArea).
QWidget *PlainDialog::createMenuRow(const DialogChoice &choice)
{
    PressArea *row = new PressArea(box);
    row->setObjectName(QStringLiteral("plainMenuRow"));
    row->setAttribute(Qt::WA_StyledBackground, true);
    row->setFixedHeight(MENU_ROW_HEIGHT);
    QLabel *label = createLabel(choice.label, "plainMenuRowLabel", row);
    label->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    QHBoxLayout *layout = new QHBoxLayout(row);
    layout->setContentsMargins(MENU_SIDE_PADDING, 0, MENU_SIDE_PADDING, 0);
    layout->addWidget(label);
    connect(row, SIGNAL(tapped()), this, SLOT(onChoiceTapped()));
    buttons.append(row);
    buttonIds.append(choice.id);
    return row;
}

QWidget *PlainDialog::createDivider()
{
    QWidget *divider = new QWidget(box);
    divider->setObjectName(QStringLiteral("plainMenuDivider"));
    divider->setAttribute(Qt::WA_StyledBackground, true);
    divider->setFixedHeight(DIVIDER_HEIGHT);
    return divider;
}

QPixmap PlainDialog::grabPageUnder(const QWidget *popup)
{
    QWidget *page = NULL;
    QList<QWidget *> windows = QApplication::topLevelWidgets();
    for (int i = 0; i < windows.size(); i++) {
        QWidget *window = windows[i];
        if (window == popup || !window->isVisible() || window->inherits("QMenu") || window->inherits("QDialog")) {
            continue;
        }
        if (!page || window->width() * window->height() > page->width() * page->height()) {
            page = window;
        }
    }
    QPixmap backdrop = page ? page->grab() : QPixmap();
    DebugLog::write("popup backdrop %s", backdrop.isNull() ? "unavailable, white" : "captured");
    return backdrop;
}

void PlainDialog::present()
{
    backdrop = grabPageUnder(this);
    setGeometry(QApplication::desktop()->screenGeometry());
    open();
}

void PlainDialog::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    if (backdrop.isNull()) {
        painter.fillRect(rect(), Qt::white);
    } else {
        painter.drawPixmap(rect(), backdrop);
    }
}

void PlainDialog::resizeEvent(QResizeEvent *event)
{
    QDialog::resizeEvent(event);
    outsideArea->setGeometry(rect());
}

void PlainDialog::onChoiceTapped()
{
    int index = buttons.indexOf(qobject_cast<QWidget *>(sender()));
    if (index < 0 || hasAnswered) {
        return;
    }
    hasAnswered = true;
    emit chosen(buttonIds[index]);
    accept();
}

void PlainDialog::onOutsideTapped()
{
    if (outsideTapRule == OUTSIDE_TAP_IGNORED) {
        DebugLog::write("plain dialog %p: outside tap ignored (destructive question)", static_cast<void *>(this));
        return;
    }
    DebugLog::write("plain dialog %p: outside tap, dismissed", static_cast<void *>(this));
    reject();
}

void PlainDialog::onDismissed()
{
    if (hasAnswered) {
        return;
    }
    hasAnswered = true;
    emit chosen(QString());
}
