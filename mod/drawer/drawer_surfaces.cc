#include "drawer_surfaces.h"

#include <QPainter>

#include "copperui/empty_state.h"
#include "debug_log.h"
#include "nickel_fonts.h"
#include "screen_logic.h"
#include "system_icon.h"
#include "text_meter.h"

using namespace ScreenLogic;

static QColor grey(int level)
{
    return QColor(level, level, level);
}

static QRect toQRect(const copperui::v1::Rect &rect)
{
    return QRect(rect.x, rect.y, rect.width, rect.height);
}

// Lines centred in a box, each LINE_HEIGHT high, from the box's top.
static void paintCentredLines(QPainter *painter, const QRect &box, const QStringList &lines, int lineHeight)
{
    for (int i = 0; i < lines.size(); i++) {
        QRect line(box.x(), box.y() + i * lineHeight, box.width(), lineHeight);
        painter->drawText(line, Qt::AlignHCenter | Qt::AlignVCenter | Qt::TextSingleLine, lines[i]);
    }
}

// ---- NoticeBox ----

NoticeBox::NoticeBox(const QString &text, QWidget *parent) : QWidget(parent), boxHeight(0)
{
    int inner = CONTENT_WIDTH - 2 * NOTICE_PADDING - 2 * NOTICE_BORDER;
    TextRun run = {BODY_SIZE, FAMILY_SANS, false};
    lines = QtTextMeter::wrapLines(text, inner, run, 0);
    boxHeight = noticeHeight(lines.size(), false);
    setFixedSize(CONTENT_WIDTH, boxHeight + RHYTHM);
}

void NoticeBox::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), grey(PAPER_WHITE));
    painter.fillRect(QRect(0, 0, CONTENT_WIDTH, boxHeight), grey(INK_BLACK));
    painter.fillRect(QRect(NOTICE_BORDER, NOTICE_BORDER, CONTENT_WIDTH - 2 * NOTICE_BORDER, boxHeight - 2 * NOTICE_BORDER),
                     grey(PAPER_GREY));
    painter.setFont(NickelFonts::font(NickelFonts::ROLE_BODY, BODY_SIZE, false));
    painter.setPen(grey(INK_BLACK));
    for (int i = 0; i < lines.size(); i++) {
        QRect line(NOTICE_BORDER + NOTICE_PADDING, NOTICE_BORDER + NOTICE_PADDING + i * BODY_LINE_HEIGHT,
                   CONTENT_WIDTH - 2 * (NOTICE_BORDER + NOTICE_PADDING), BODY_LINE_HEIGHT);
        painter.drawText(line, Qt::AlignLeft | Qt::AlignVCenter | Qt::TextSingleLine, lines[i]);
    }
}

// ---- PrimaryButton ----

QStringList PrimaryButton::labelLines(const QString &label)
{
    TextRun run = {BODY_SIZE, FAMILY_SANS, true};
    return QtTextMeter::wrapLines(label, CONTENT_WIDTH - 2 * RHYTHM, run, BUTTON_MAX_LINES);
}

PrimaryButton::PrimaryButton(const QString &label, QWidget *parent) : PressArea(parent), lines(labelLines(label))
{
    setObjectName(QStringLiteral("drawerPrimaryButton"));
    setFixedSize(CONTENT_WIDTH, buttonHeight(lines.size()));
}

void PrimaryButton::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), grey(INK_BLACK));
    painter.setFont(NickelFonts::font(NickelFonts::ROLE_BODY, BODY_SIZE, true));
    painter.setPen(grey(PAPER_WHITE));
    int textHeight = lines.size() * BODY_LINE_HEIGHT;
    QRect text(RHYTHM, (height() - textHeight) / 2, width() - 2 * RHYTHM, textHeight);
    paintCentredLines(&painter, text, lines, BODY_LINE_HEIGHT);
}

// ---- EmptyState ----

EmptyState::EmptyState(const QString &heading, const QString &caption, bool wantsIcon, const QString &buttonLabel,
                       QWidget *parent)
    : QWidget(parent), button(NULL)
{
    setObjectName(QStringLiteral("drawerEmptyState"));
    TextRun headingRun = {HEADING_SIZE, FAMILY_SERIF, true};
    TextRun captionRun = {CAPTION_SIZE, FAMILY_SANS, false};
    headingLines = QtTextMeter::wrapLines(heading, CONTENT_WIDTH, headingRun, 0);
    captionLines = QtTextMeter::wrapLines(caption, CONTENT_WIDTH, captionRun, 0);
    bool hasIcon = wantsIcon && !SystemIcon::icon(SystemIcon::APP).isNull();
    if (wantsIcon && !hasIcon) {
        DebugLog::write("drawer view: empty-state icon not installed, shown without it");
    }
    QStringList buttonLines = buttonLabel.isEmpty() ? QStringList() : PrimaryButton::labelLines(buttonLabel);
    copperui::v1::EmptyStateContent content = {hasIcon, headingLines.size(), captionLines.size(), !buttonLabel.isEmpty(),
                                               buttonLines.size()};
    copperui::v1::EmptyStateGeometry geometry = copperui::v1::emptyStateGeometry(content, 0, 0, CONTENT_WIDTH);
    setFixedSize(CONTENT_WIDTH, geometry.height);
    iconRect = hasIcon ? toQRect(geometry.icon) : QRect();
    headingRect = toQRect(geometry.heading);
    captionRect = toQRect(geometry.caption);
    if (content.hasButton) {
        button = new PrimaryButton(buttonLabel, this);
        button->move(geometry.button.x, geometry.button.y);
        connect(button, SIGNAL(tapped()), this, SIGNAL(buttonTapped()));
    }
}

void EmptyState::setInputEnabled(bool isEnabled)
{
    if (button) {
        button->setAcceptsPresses(isEnabled);
    }
}

void EmptyState::paintEvent(QPaintEvent *)
{
    QPainter painter(this);
    painter.fillRect(rect(), grey(PAPER_WHITE));
    if (!iconRect.isNull()) {
        SystemIcon::drawCentred(&painter, iconRect, SystemIcon::icon(SystemIcon::APP).normal);
    }
    painter.setFont(NickelFonts::font(NickelFonts::ROLE_TITLE, HEADING_SIZE, true));
    painter.setPen(grey(INK_BLACK));
    paintCentredLines(&painter, headingRect, headingLines, HEADING_LINE_HEIGHT);
    painter.setFont(NickelFonts::font(NickelFonts::ROLE_BODY, CAPTION_SIZE, false));
    painter.setPen(grey(INK_MUTED));
    paintCentredLines(&painter, captionRect, captionLines, CAPTION_LINE_HEIGHT);
}
