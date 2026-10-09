#include "plain_dialog.h"

#include <QApplication>
#include <QBoxLayout>
#include <QDesktopWidget>
#include <QLabel>
#include <QPainter>
#include <QTimer>

#include "drawer_logic.h"
#include "nickel_fonts.h"
#include "press_area.h"

static const int DIALOG_WIDTH = 1000;
static const int PADDING = 48;
static const int SPACING = 24;
static const int BUTTON_HEIGHT = 120;
static const int BORDER_WIDTH = 2;

static const char *const DIALOG_STYLE =
    "PlainDialog { background: white; }"
    "QWidget { font-family: @BODY_FAMILY@; }"
    "QLabel { color: black; background: transparent; }"
    "#plainDialogTitle { font-family: @TITLE_FAMILY@; font-size: 42px; font-weight: bold; }"
    "#plainDialogText { font-size: 32px; }"
    "#plainDialogButton { border: 2px solid black; background: white; }"
    "#plainDialogButtonLabel { font-size: 32px; }";

PlainDialog::PlainDialog(const QString &title, const QString &text, const QList<DialogChoice> &choices, Layout layout)
    : QDialog(NULL, Qt::Dialog | Qt::FramelessWindowHint), hasAnswered(false)
{
    setAttribute(Qt::WA_DeleteOnClose, true);
    setStyleSheet(NickelFonts::styleSheet(DIALOG_STYLE));
    setFixedWidth(DIALOG_WIDTH);

    QVBoxLayout *column = new QVBoxLayout(this);
    column->setContentsMargins(PADDING, PADDING, PADDING, PADDING);
    column->setSpacing(SPACING);
    if (!title.isEmpty()) {
        QLabel *titleLabel = new QLabel(title, this);
        titleLabel->setObjectName(QStringLiteral("plainDialogTitle"));
        titleLabel->setWordWrap(true);
        column->addWidget(titleLabel);
    }
    if (!text.isEmpty()) {
        QLabel *textLabel = new QLabel(text, this);
        textLabel->setObjectName(QStringLiteral("plainDialogText"));
        textLabel->setWordWrap(true);
        textLabel->setTextFormat(Qt::PlainText);
        column->addWidget(textLabel);
    }
    QBoxLayout *buttonLayout =
        new QBoxLayout(layout == BUTTONS_STACKED ? QBoxLayout::TopToBottom : QBoxLayout::LeftToRight);
    buttonLayout->setSpacing(SPACING);
    for (int i = 0; i < choices.size(); i++) {
        buttonLayout->addWidget(createButton(choices[i]));
    }
    column->addLayout(buttonLayout);

    connect(this, SIGNAL(rejected()), this, SLOT(onDismissed()));
    // ui-kit §3.15: a dialog left alone for 120 s counts as cancel.
    QTimer::singleShot(DrawerLogic::CONFIRM_TIMEOUT_MILLISECONDS, this, SLOT(reject()));
}

QWidget *PlainDialog::createButton(const DialogChoice &choice)
{
    PressArea *button = new PressArea(this);
    button->setObjectName(QStringLiteral("plainDialogButton"));
    button->setAttribute(Qt::WA_StyledBackground, true);
    button->setMinimumHeight(BUTTON_HEIGHT);
    QLabel *label = new QLabel(choice.label, button);
    label->setObjectName(QStringLiteral("plainDialogButtonLabel"));
    label->setAlignment(Qt::AlignCenter);
    label->setWordWrap(true); // long vi labels wrap instead of clipping (ui-kit §4)
    label->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    QHBoxLayout *layout = new QHBoxLayout(button);
    layout->addWidget(label);
    connect(button, SIGNAL(tapped()), this, SLOT(onChoiceTapped()));
    buttons.append(button);
    buttonIds.append(choice.id);
    return button;
}

void PlainDialog::present()
{
    adjustSize();
    QRect screen = QApplication::desktop()->screenGeometry();
    move(screen.center() - rect().center());
    open();
}

void PlainDialog::paintEvent(QPaintEvent *event)
{
    QDialog::paintEvent(event);
    QPainter painter(this);
    painter.setPen(QPen(Qt::black, BORDER_WIDTH));
    painter.setBrush(Qt::NoBrush);
    painter.drawRect(rect().adjusted(1, 1, -1, -1));
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

void PlainDialog::onDismissed()
{
    if (hasAnswered) {
        return;
    }
    hasAnswered = true;
    emit chosen(QString());
}
