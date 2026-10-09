#include "screen_layout.h"

#include <QVector>

#include "qr_encoder.h"

using namespace ScreenLogic;

static const char *const CHEVRON_GLYPH = "\xE2\x80\xBA"; // ›
static const int HEADER_LINE_Y = 52;
static const int HEADER_LINE_HEIGHT = 44;
static const int HEADER_LINE_STEP = 48;
static const int BACK_TAP_WIDTH = GUTTER + 120;
static const int FIELD_CAPTION_GAP = 8;
static const int SWITCH_BORDER = 3;
static const int SWITCH_KNOB = 40;
static const int SWITCH_KNOB_INSET = 6;
static const int RADIO_RING_BORDER = 4;
static const int RADIO_DOT = 24;
static const int BUTTON_BORDER = 3;
static const int ROW_DETAIL_WIDTH = 280;
static const int ROW_DETAIL_RIGHT = 320;
static const int TAP_ROW_TEXT_INSET = 360;
static const int TOGGLE_WORD_WIDTH = 140;
static const int FIELD_BORDER = 2;
static const int QR_FRAME_BORDER = 2;

// ---- Text helpers (Qt strings over the UTF-8 rules of screen_logic) ----

static QStringList wrap(const QString &text, int width, int size, Family family)
{
    QByteArray utf8 = text.toUtf8();
    QVector<char> out(utf8.size() * 2 + 16);
    int count = wrapText(utf8.constData(), width, size, family, out.data(), out.size());
    return count == 0 ? QStringList() : QString::fromUtf8(out.constData()).split(QLatin1Char('\n'));
}

static QString elided(const QString &text, int width, int size, Family family, bool isMiddle)
{
    QByteArray utf8 = text.toUtf8();
    QVector<char> out(utf8.size() + 16);
    elideText(utf8.constData(), width, size, family, isMiddle, out.data(), out.size());
    return QString::fromUtf8(out.constData());
}

static bool isFileName(const QString &text)
{
    return looksLikeFileName(text.toUtf8().constData());
}

static PaintItem::Align alignOf(const QString &align)
{
    if (align == QLatin1String("center")) {
        return PaintItem::ALIGN_CENTER;
    }
    return align == QLatin1String("end") ? PaintItem::ALIGN_END : PaintItem::ALIGN_START;
}

static TextStyle styleOf(const QString &style)
{
    if (style == QLatin1String("caption")) {
        return STYLE_CAPTION;
    }
    return style == QLatin1String("heading") ? STYLE_HEADING : STYLE_BODY;
}

// ---- Paint items ----

bool PaintItem::operator==(const PaintItem &other) const
{
    return shape == other.shape && rect == other.rect && fill == other.fill && border == other.border &&
           lines == other.lines && size == other.size && lineHeight == other.lineHeight && family == other.family &&
           isBold == other.isBold && isUnderlined == other.isUnderlined && align == other.align && ink == other.ink &&
           qrSide == other.qrSide && qrModules == other.qrModules;
}

static PaintItem rectItem(int x, int y, int width, int height, int fill, int border)
{
    PaintItem item;
    item.shape = PaintItem::SHAPE_RECT;
    item.rect = QRect(x, y, width, height);
    item.fill = fill;
    item.border = border;
    return item;
}

static PaintItem ellipseItem(int x, int y, int side, int fill, int border)
{
    PaintItem item = rectItem(x, y, side, side, fill, border);
    item.shape = PaintItem::SHAPE_ELLIPSE;
    return item;
}

static PaintItem textItem(int x, int y, int width, int height, const QString &text, int size, Family family,
                          PaintItem::Align align)
{
    PaintItem item;
    item.shape = PaintItem::SHAPE_TEXT;
    item.rect = QRect(x, y, width, height);
    item.lines << text;
    item.size = size;
    item.lineHeight = height;
    item.family = family;
    item.align = align;
    return item;
}

static PaintItem separatorItem(int y)
{
    return rectItem(GUTTER, y, CONTENT_WIDTH, SEPARATOR, INK_BLACK, 0);
}

static void addTarget(ScreenPage *page, const QRect &rect, int component, const QString &value)
{
    TapTarget target;
    target.kind = TapTarget::TAP_COMPONENT;
    target.rect = rect;
    target.component = component;
    target.value = value;
    page->targets.append(target);
}

// ---- Blocks (components.go) ----

static LayoutBlock block(LayoutBlock::Kind kind, int component, int height, int gap)
{
    LayoutBlock result;
    result.kind = kind;
    result.component = component;
    result.size.height = height;
    result.size.gap = gap;
    return result;
}

static LayoutBlock paragraph(int component, const QString &text, TextStyle style, Family family, bool isBold,
                             PaintItem::Align align, int ink)
{
    LayoutBlock result = block(LayoutBlock::KIND_PARAGRAPH, component, 0, RHYTHM);
    result.textSize = textSize(style);
    result.lineHeight = lineHeight(style);
    result.lines = wrap(text, CONTENT_WIDTH, result.textSize, family);
    result.size.height = textBlockHeight(result.lines.size(), result.lineHeight);
    result.family = family;
    result.isBold = isBold;
    result.align = align;
    result.ink = ink;
    return result;
}

static LayoutBlock sectionBlock(int component, const QString &label)
{
    LayoutBlock result = block(LayoutBlock::KIND_SECTION, component, SECTION_HEIGHT, 0);
    result.primary = label;
    return result;
}

static LayoutBlock valueRow(int component, const QString &key, const QString &value, bool isMiddleElided)
{
    LayoutBlock result = block(LayoutBlock::KIND_VALUE_ROW, component, ROW_HEIGHT, 0);
    result.primary = key;
    result.secondary = value;
    result.isMiddleElided = isMiddleElided;
    return result;
}

static LayoutBlock singleLine(int component, LayoutBlock::Kind kind, const QString &text, TextStyle style,
                              Family family, int ink)
{
    LayoutBlock result = block(kind, component, lineHeight(style), RHYTHM);
    result.primary = text;
    result.textSize = textSize(style);
    result.lineHeight = lineHeight(style);
    result.family = family;
    result.ink = ink;
    return result;
}

static void withTrailingGap(QList<LayoutBlock> *blocks, int first)
{
    if (blocks->size() > first) {
        (*blocks)[blocks->size() - 1].size.gap = RHYTHM;
    }
}

static LayoutBlock bigCodeBlock(int component, const QString &code, int digits)
{
    LayoutBlock result = block(LayoutBlock::KIND_BIG_CODE, component, BIG_CODE_HEIGHT, RHYTHM);
    result.primary = code.isEmpty() ? QString(digits > 0 ? digits : 4, QLatin1Char('-')) : code;
    return result;
}

static LayoutBlock qrBlock(int component, const QString &value)
{
    LayoutBlock result = block(LayoutBlock::KIND_QR, component, QR_SIDE, RHYTHM);
    static QrEncoder::QrCode code; // ~9 KB, GUI thread only
    QByteArray utf8 = value.toUtf8();
    if (!value.isEmpty() &&
        QrEncoder::encode(reinterpret_cast<const unsigned char *>(utf8.constData()), utf8.size(), &code)) {
        result.qrSide = code.side;
        result.qrModules = QByteArray(reinterpret_cast<const char *>(code.modules), code.side * code.side);
    }
    return result;
}

static void addQrBlocks(int component, const QString &address, const QString &caption, QList<LayoutBlock> *blocks)
{
    blocks->append(qrBlock(component, address));
    if (!address.isEmpty()) {
        blocks->append(singleLine(component, LayoutBlock::KIND_LINE, address, STYLE_CAPTION, FAMILY_MONO, INK_BLACK));
        blocks->last().align = PaintItem::ALIGN_CENTER;
    }
    if (!caption.isEmpty()) {
        blocks->append(paragraph(component, caption, STYLE_CAPTION, FAMILY_SANS, false, PaintItem::ALIGN_CENTER, INK_MUTED));
    }
}

static LayoutBlock buttonBlock(int component, const QString &label, bool isPrimary, bool isEnabled)
{
    LayoutBlock result = block(LayoutBlock::KIND_BUTTON, component, 0, RHYTHM);
    result.lines = wrap(label, CONTENT_WIDTH - 2 * RHYTHM, BODY_SIZE, FAMILY_SANS);
    result.size.height = buttonHeight(result.lines.size());
    result.isOn = isPrimary;
    result.isTappable = isEnabled;
    return result;
}

static LayoutBlock noticeBlock(int component, const QString &text, const QString &title)
{
    LayoutBlock result = block(LayoutBlock::KIND_NOTICE, component, 0, RHYTHM);
    int inner = CONTENT_WIDTH - 2 * NOTICE_PADDING - 2 * NOTICE_BORDER;
    result.lines = wrap(text, inner, BODY_SIZE, FAMILY_SANS);
    result.title = title;
    result.size.height = noticeHeight(result.lines.size(), !title.isEmpty());
    return result;
}

static void addKeyValueBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    int first = blocks->size();
    for (int i = 0; i < c.rows.size(); i++) {
        blocks->append(valueRow(component, c.rows[i].first, c.rows[i].second, isFileName(c.rows[i].second)));
    }
    withTrailingGap(blocks, first);
}

static void addRadioBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    int first = blocks->size();
    blocks->append(sectionBlock(component, c.text));
    for (int i = 0; i < c.options.size(); i++) {
        LayoutBlock option = block(LayoutBlock::KIND_RADIO_OPTION, component, ROW_HEIGHT, 0);
        option.primary = c.options[i].second;
        option.tapValue = c.options[i].first;
        option.isOn = c.options[i].first == c.value;
        option.isTappable = c.isEnabled;
        blocks->append(option);
    }
    withTrailingGap(blocks, first);
}

static void addFieldBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    LayoutBlock caption = paragraph(component, c.text, STYLE_CAPTION, FAMILY_SANS, false, PaintItem::ALIGN_START, INK_MUTED);
    caption.size.gap = FIELD_CAPTION_GAP;
    blocks->append(caption);
    LayoutBlock box = block(LayoutBlock::KIND_FIELD_BOX, component, FIELD_HEIGHT, RHYTHM);
    bool isEmpty = c.value.isEmpty();
    box.primary = isEmpty ? c.placeholder : c.value;
    box.ink = isEmpty ? INK_DIM : INK_BLACK;
    box.isTappable = c.isEnabled;
    blocks->append(box);
}

static void addPairBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    if (!c.title.isEmpty()) {
        blocks->append(paragraph(component, c.title, STYLE_HEADING, FAMILY_SANS, true, PaintItem::ALIGN_CENTER, INK_BLACK));
    }
    addQrBlocks(component, c.url, QString(), blocks);
    blocks->append(bigCodeBlock(component, c.code, c.digits));
    QStringList captions;
    captions << c.hint << c.expires;
    for (int i = 0; i < captions.size(); i++) {
        if (!captions[i].isEmpty()) {
            blocks->append(paragraph(component, captions[i], STYLE_CAPTION, FAMILY_SANS, false, PaintItem::ALIGN_CENTER, INK_MUTED));
        }
    }
}

static void addStepsBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    QString headline = QStringLiteral("%1 %2 / %3").arg(c.label, c.done, c.total);
    LayoutBlock line = singleLine(component, LayoutBlock::KIND_LINE, headline, STYLE_BODY, FAMILY_SANS, INK_BLACK);
    line.size.gap = c.detail.isEmpty() ? RHYTHM : 0;
    blocks->append(line);
    if (!c.detail.isEmpty()) {
        blocks->append(singleLine(component, LayoutBlock::KIND_DETAIL_LINE, c.detail, STYLE_CAPTION, FAMILY_SANS, INK_MUTED));
    }
}

static void addListBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    if (!c.title.isEmpty()) {
        blocks->append(sectionBlock(component, c.title));
    }
    if (c.items.isEmpty()) {
        LayoutBlock empty = block(LayoutBlock::KIND_CENTERED_LINE, component, ROW_HEIGHT, RHYTHM);
        empty.primary = c.emptyText;
        blocks->append(empty);
        return;
    }
    int first = blocks->size();
    for (int i = 0; i < c.items.size(); i++) {
        const QString &primary = c.items[i].first;
        LayoutBlock row = valueRow(component, primary, c.items[i].second, isFileName(primary));
        row.isTappable = !c.tap.isEmpty();
        row.tapValue = primary;
        blocks->append(row);
    }
    withTrailingGap(blocks, first);
}

static void addEmptyBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    blocks->append(block(LayoutBlock::KIND_SPACER, component, EMPTY_SPACER_HEIGHT, 0));
    blocks->append(paragraph(component, c.text, STYLE_HEADING, FAMILY_SANS, true, PaintItem::ALIGN_CENTER, INK_BLACK));
    if (!c.caption.isEmpty()) {
        blocks->append(paragraph(component, c.caption, STYLE_CAPTION, FAMILY_SANS, false, PaintItem::ALIGN_CENTER, INK_MUTED));
    }
    if (!c.button.isEmpty()) {
        blocks->append(buttonBlock(component, c.button, true, true));
    }
}

void ScreenLayout::addComponentBlocks(int index, QList<LayoutBlock> *blocks) const
{
    const ScreenComponent &c = components[index];
    const QString &type = c.type;
    if (type == QLatin1String("text")) {
        TextStyle style = styleOf(c.style);
        blocks->append(paragraph(index, c.text, style, FAMILY_SANS, style == STYLE_HEADING, alignOf(c.align),
                                 style == STYLE_CAPTION ? INK_MUTED : INK_BLACK));
    } else if (type == QLatin1String("section")) {
        blocks->append(sectionBlock(index, c.text));
    } else if (type == QLatin1String("kv")) {
        addKeyValueBlocks(index, c, blocks);
    } else if (type == QLatin1String("row")) {
        LayoutBlock row = block(LayoutBlock::KIND_TAP_ROW, index, TAP_ROW_HEIGHT, RHYTHM);
        row.primary = c.text;
        row.secondary = c.detail;
        row.isTappable = !c.action.isEmpty();
        blocks->append(row);
    } else if (type == QLatin1String("toggle")) {
        LayoutBlock toggle = block(LayoutBlock::KIND_TOGGLE, index, ROW_HEIGHT, RHYTHM);
        toggle.primary = c.text;
        toggle.isOn = c.isOn;
        toggle.isTappable = c.isEnabled;
        blocks->append(toggle);
    } else if (type == QLatin1String("radio")) {
        addRadioBlocks(index, c, blocks);
    } else if (type == QLatin1String("field")) {
        addFieldBlocks(index, c, blocks);
    } else if (type == QLatin1String("bigcode")) {
        blocks->append(bigCodeBlock(index, c.text, c.digits));
    } else if (type == QLatin1String("qr")) {
        addQrBlocks(index, c.text, c.caption, blocks);
    } else if (type == QLatin1String("pair")) {
        addPairBlocks(index, c, blocks);
    } else if (type == QLatin1String("steps")) {
        addStepsBlocks(index, c, blocks);
    } else if (type == QLatin1String("notice")) {
        blocks->append(noticeBlock(index, c.text, c.title));
    } else if (type == QLatin1String("button")) {
        blocks->append(buttonBlock(index, c.text, c.isPrimaryButton(), c.isEnabled));
    } else if (type == QLatin1String("list")) {
        addListBlocks(index, c, blocks);
    } else if (type == QLatin1String("empty")) {
        addEmptyBlocks(index, c, blocks);
    }
}

// ---- Layout ----

static QVector<Block> sizesOf(const QList<LayoutBlock> &blocks)
{
    QVector<Block> sizes(blocks.size());
    for (int i = 0; i < blocks.size(); i++) {
        sizes[i] = blocks[i].size;
    }
    return sizes;
}

ScreenLayout::ScreenLayout() : height(SCREEN_HEIGHT)
{
    frame.top = frame.bottom = frame.footerTop = frame.pageBarTop = 0;
    frame.pageCount = 1;
}

ScreenLayout::ScreenLayout(const ScreenModel &screenModel, int viewHeight, const HostTexts &texts)
    : model(screenModel), hostTexts(texts), height(viewHeight)
{
    components = model.body + model.footer;
    for (int i = 0; i < model.body.size(); i++) {
        addComponentBlocks(i, &body);
    }
    for (int i = 0; i < model.footer.size(); i++) {
        addComponentBlocks(model.body.size() + i, &footer);
    }
    QVector<Block> footerSizes = sizesOf(footer);
    int footerHeight = blocksHeight(footerSizes.constData(), 0, footerSizes.size());
    QVector<Block> bodySizes = sizesOf(body);
    QVector<int> pages(bodySizes.size() + 1);
    frame = layoutBody(bodySizes.constData(), bodySizes.size(), footerHeight, height, pages.data());
    for (int i = 0; i < bodySizes.size(); i++) {
        pageOfBlock.append(pages[i]);
    }
}

ScreenPage ScreenLayout::page(int index) const
{
    ScreenPage result;
    drawHeader(&result);
    int y = frame.top;
    for (int i = 0; i < body.size(); i++) {
        if (pageOfBlock[i] == index) {
            drawBlock(body[i], y, &result);
            y += body[i].size.height + body[i].size.gap;
        }
    }
    if (frame.pageCount > 1) {
        drawPageBar(index, &result);
    }
    y = frame.footerTop;
    for (int i = 0; i < footer.size(); i++) {
        drawBlock(footer[i], y, &result);
        y += footer[i].size.height + footer[i].size.gap;
    }
    return result;
}

void ScreenLayout::drawHeader(ScreenPage *page) const
{
    PaintItem arrow;
    arrow.shape = PaintItem::SHAPE_BACK_ARROW;
    arrow.rect = QRect(GUTTER, 40, 64, 88); // the preview's "‹" box (layout.go)
    page->items.append(arrow);
    TapTarget back;
    back.kind = TapTarget::TAP_BACK;
    back.rect = QRect(0, 0, BACK_TAP_WIDTH, HEADER_HEIGHT);
    page->targets.append(back);

    int titleWidth = CONTENT_WIDTH - BACK_WIDTH - HEADER_SIDE_WIDTH;
    PaintItem title = textItem(GUTTER + BACK_WIDTH, 44, titleWidth, 80,
                               elided(model.title, titleWidth, HEADER_TITLE_SIZE, FAMILY_SERIF, false), HEADER_TITLE_SIZE,
                               FAMILY_SERIF, PaintItem::ALIGN_START);
    title.isBold = true;
    page->items.append(title);

    int sideX = SCREEN_WIDTH - GUTTER - HEADER_SIDE_WIDTH;
    int y = HEADER_LINE_Y;
    if (!model.status.isEmpty()) {
        PaintItem status = textItem(sideX, y, HEADER_SIDE_WIDTH, HEADER_LINE_HEIGHT,
                                    elided(model.status, HEADER_SIDE_WIDTH, HEADER_STATUS_SIZE, FAMILY_SANS, false),
                                    HEADER_STATUS_SIZE, FAMILY_SANS, PaintItem::ALIGN_END);
        status.ink = INK_MUTED;
        page->items.append(status);
        y += HEADER_LINE_STEP;
    }
    if (!model.headerLabel.isEmpty()) {
        PaintItem action = textItem(sideX, y, HEADER_SIDE_WIDTH, HEADER_LINE_HEIGHT,
                                    elided(model.headerLabel, HEADER_SIDE_WIDTH, HEADER_STATUS_SIZE, FAMILY_SANS, false),
                                    HEADER_STATUS_SIZE, FAMILY_SANS, PaintItem::ALIGN_END);
        action.isUnderlined = true;
        page->items.append(action);
        TapTarget target;
        target.kind = TapTarget::TAP_HEADER_ACTION;
        target.rect = QRect(sideX, 0, HEADER_SIDE_WIDTH + GUTTER, HEADER_HEIGHT);
        page->targets.append(target);
    }
    page->items.append(separatorItem(HEADER_HEIGHT));
}

void ScreenLayout::drawPageBar(int index, ScreenPage *page) const
{
    PaintItem bar = textItem(GUTTER, frame.pageBarTop, CONTENT_WIDTH, PAGE_BAR_HEIGHT,
                             QStringLiteral("%1 / %2").arg(index + 1).arg(frame.pageCount), CAPTION_SIZE, FAMILY_SANS,
                             PaintItem::ALIGN_CENTER);
    bar.ink = INK_MUTED;
    page->items.append(bar);
    // Touch paging: the left and right halves of the page bar (ui-kit §5.3).
    TapTarget previous;
    previous.kind = TapTarget::TAP_PREVIOUS_PAGE;
    previous.rect = QRect(GUTTER, frame.pageBarTop, CONTENT_WIDTH / 2, PAGE_BAR_HEIGHT + RHYTHM);
    TapTarget next = previous;
    next.kind = TapTarget::TAP_NEXT_PAGE;
    next.rect.moveLeft(GUTTER + CONTENT_WIDTH / 2);
    page->targets << previous << next;
}

// ---- Drawing one block (components.go draw functions) ----

static void drawValueRow(const LayoutBlock &b, int y, ScreenPage *page)
{
    int valueWidth = CONTENT_WIDTH - KEY_COLUMN_WIDTH - RHYTHM;
    page->items.append(textItem(GUTTER, y + 24, KEY_COLUMN_WIDTH, 56,
                                elided(b.primary, KEY_COLUMN_WIDTH, ROW_TEXT_SIZE, FAMILY_SANS, false), ROW_TEXT_SIZE,
                                FAMILY_SANS, PaintItem::ALIGN_START));
    PaintItem value = textItem(GUTTER + KEY_COLUMN_WIDTH + RHYTHM, y + 24, valueWidth, 56,
                               elided(b.secondary, valueWidth, ROW_TEXT_SIZE, FAMILY_SANS, b.isMiddleElided),
                               ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_END);
    value.isBold = true;
    page->items.append(value);
    page->items.append(separatorItem(y + ROW_HEIGHT - SEPARATOR));
    if (b.isTappable) {
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, ROW_HEIGHT), b.component, b.tapValue);
    }
}

static void drawSection(const LayoutBlock &b, int y, ScreenPage *page)
{
    PaintItem label = textItem(GUTTER, y + 16, CONTENT_WIDTH, 40,
                               elided(b.primary, CONTENT_WIDTH, SECTION_TEXT_SIZE, FAMILY_SANS, false).toUpper(),
                               SECTION_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_START);
    label.ink = INK_MUTED;
    label.isBold = true;
    page->items << label << separatorItem(y + SECTION_HEIGHT - SEPARATOR);
}

static void drawTapRow(const LayoutBlock &b, int y, ScreenPage *page)
{
    int textWidth = CONTENT_WIDTH - TAP_ROW_TEXT_INSET;
    page->items << textItem(GUTTER, y + 36, textWidth, 56, elided(b.primary, textWidth, ROW_TEXT_SIZE, FAMILY_SANS, false),
                            ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_START)
                << separatorItem(y + TAP_ROW_HEIGHT - SEPARATOR);
    if (!b.secondary.isEmpty()) {
        PaintItem detail = textItem(SCREEN_WIDTH - GUTTER - ROW_DETAIL_RIGHT, y + 36, ROW_DETAIL_WIDTH, 56,
                                    elided(b.secondary, ROW_DETAIL_WIDTH, ROW_TEXT_SIZE, FAMILY_SANS, false),
                                    ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_END);
        detail.ink = INK_MUTED;
        page->items.append(detail);
    }
    if (b.isTappable) {
        page->items.append(textItem(SCREEN_WIDTH - GUTTER - 32, y + 36, 32, 56, QString::fromUtf8(CHEVRON_GLYPH),
                                    ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_END));
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, TAP_ROW_HEIGHT), b.component, QString());
    }
}

// A 96 x 52 switch: outline with the knob left (off), or filled with the knob right (on).
static void drawSwitch(int x, int y, bool isOn, ScreenPage *page)
{
    PaintItem track = rectItem(x, y, SWITCH_WIDTH, SWITCH_HEIGHT, isOn ? INK_BLACK : NO_FILL, SWITCH_BORDER);
    int knobX = isOn ? x + SWITCH_WIDTH - SWITCH_KNOB_INSET - SWITCH_KNOB : x + SWITCH_KNOB_INSET;
    page->items << track << ellipseItem(knobX, y + SWITCH_KNOB_INSET, SWITCH_KNOB, PAPER_WHITE, SWITCH_BORDER);
}

static void drawToggle(const LayoutBlock &b, int y, const HostTexts &texts, ScreenPage *page)
{
    int textWidth = CONTENT_WIDTH - TAP_ROW_TEXT_INSET;
    PaintItem word = textItem(SCREEN_WIDTH - GUTTER - TOGGLE_WORD_WIDTH, y + 24, TOGGLE_WORD_WIDTH, 56,
                              b.isOn ? texts.on : texts.off, ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_END);
    word.isBold = b.isOn;
    page->items << textItem(GUTTER, y + 24, textWidth, 56, elided(b.primary, textWidth, ROW_TEXT_SIZE, FAMILY_SANS, false),
                            ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_START)
                << word << separatorItem(y + ROW_HEIGHT - SEPARATOR);
    drawSwitch(SCREEN_WIDTH - GUTTER - TOGGLE_WORD_WIDTH - RHYTHM - SWITCH_WIDTH, y + 26, b.isOn, page);
    if (b.isTappable) {
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, ROW_HEIGHT), b.component, QString());
    }
}

static void drawRadioOption(const LayoutBlock &b, int y, ScreenPage *page)
{
    page->items.append(ellipseItem(GUTTER, y + 26, RADIO_RING_SIDE, NO_FILL, RADIO_RING_BORDER));
    if (b.isOn) {
        page->items.append(ellipseItem(GUTTER + 14, y + 40, RADIO_DOT, INK_BLACK, 0));
    }
    int textWidth = CONTENT_WIDTH - RADIO_RING_SIDE - RHYTHM;
    PaintItem label = textItem(GUTTER + RADIO_RING_SIDE + RHYTHM, y + 24, textWidth, 56,
                               elided(b.primary, textWidth, ROW_TEXT_SIZE, FAMILY_SANS, false), ROW_TEXT_SIZE,
                               FAMILY_SANS, PaintItem::ALIGN_START);
    label.isBold = b.isOn;
    page->items << label << separatorItem(y + ROW_HEIGHT - SEPARATOR);
    if (b.isTappable) {
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, ROW_HEIGHT), b.component, b.tapValue);
    }
}

static void drawFieldBox(const LayoutBlock &b, int y, ScreenPage *page)
{
    int textWidth = CONTENT_WIDTH - 2 * RHYTHM;
    PaintItem text = textItem(GUTTER + RHYTHM, y + 30, textWidth, 60,
                              elided(b.primary, textWidth, ROW_TEXT_SIZE, FAMILY_SANS, false), ROW_TEXT_SIZE,
                              FAMILY_SANS, PaintItem::ALIGN_START);
    text.ink = b.ink;
    page->items << rectItem(GUTTER, y, CONTENT_WIDTH, FIELD_HEIGHT, NO_FILL, FIELD_BORDER) << text;
    if (b.isTappable) {
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, FIELD_HEIGHT), b.component, QString());
    }
}

static void drawQr(const LayoutBlock &b, int y, ScreenPage *page)
{
    int x = (SCREEN_WIDTH - QR_SIDE) / 2;
    if (b.qrSide == 0) {
        // Nothing to encode (empty or too long): an empty frame keeps the layout.
        page->items.append(rectItem(x, y, QR_SIDE, QR_SIDE, NO_FILL, QR_FRAME_BORDER));
        return;
    }
    PaintItem code = rectItem(x, y, QR_SIDE, QR_SIDE, PAPER_WHITE, 0);
    code.shape = PaintItem::SHAPE_QR;
    code.qrModules = b.qrModules;
    code.qrSide = b.qrSide;
    page->items.append(code);
}

static void drawNotice(const LayoutBlock &b, int y, ScreenPage *page)
{
    int inner = CONTENT_WIDTH - 2 * NOTICE_PADDING - 2 * NOTICE_BORDER;
    page->items.append(rectItem(GUTTER, y, CONTENT_WIDTH, b.size.height, PAPER_GREY, NOTICE_BORDER));
    int top = y + NOTICE_PADDING;
    if (!b.title.isEmpty()) {
        PaintItem heading = textItem(GUTTER + NOTICE_PADDING, top, inner, BODY_LINE_HEIGHT,
                                     elided(b.title, inner, BODY_SIZE, FAMILY_SANS, false), BODY_SIZE, FAMILY_SANS,
                                     PaintItem::ALIGN_START);
        heading.isBold = true;
        page->items.append(heading);
        top += BODY_LINE_HEIGHT;
    }
    PaintItem text = textItem(GUTTER + NOTICE_PADDING, top, inner, b.lines.size() * BODY_LINE_HEIGHT, QString(),
                              BODY_SIZE, FAMILY_SANS, PaintItem::ALIGN_START);
    text.lines = b.lines;
    text.lineHeight = BODY_LINE_HEIGHT;
    page->items.append(text);
}

static void drawButton(const LayoutBlock &b, int y, ScreenPage *page)
{
    int height = b.size.height;
    PaintItem box = rectItem(GUTTER, y, CONTENT_WIDTH, height, NO_FILL, BUTTON_BORDER);
    int ink = INK_BLACK;
    if (b.isOn) {
        box.fill = INK_BLACK;
        ink = PAPER_WHITE;
    }
    if (!b.isTappable) {
        box.fill = NO_FILL;
        ink = INK_DIM;
    }
    int textHeight = b.lines.size() * BODY_LINE_HEIGHT;
    PaintItem text = textItem(GUTTER + RHYTHM, y + (height - textHeight) / 2, CONTENT_WIDTH - 2 * RHYTHM, textHeight,
                              QString(), BODY_SIZE, FAMILY_SANS, PaintItem::ALIGN_CENTER);
    text.lines = b.lines;
    text.lineHeight = BODY_LINE_HEIGHT;
    text.isBold = true;
    text.ink = ink;
    page->items << box << text;
    if (b.isTappable) {
        addTarget(page, box.rect, b.component, QString());
    }
}

static void drawLine(const LayoutBlock &b, int y, bool isMiddleElided, ScreenPage *page)
{
    PaintItem line = textItem(GUTTER, y, CONTENT_WIDTH, b.lineHeight,
                              elided(b.primary, CONTENT_WIDTH, b.textSize, b.family, isMiddleElided), b.textSize, b.family,
                              b.align);
    line.ink = b.ink;
    page->items.append(line);
}

void ScreenLayout::drawBlock(const LayoutBlock &b, int y, ScreenPage *page) const
{
    switch (b.kind) {
    case LayoutBlock::KIND_SPACER:
        break;
    case LayoutBlock::KIND_PARAGRAPH: {
        PaintItem text = textItem(GUTTER, y, CONTENT_WIDTH, b.size.height, QString(), b.textSize, b.family, b.align);
        text.lines = b.lines;
        text.lineHeight = b.lineHeight;
        text.isBold = b.isBold;
        text.ink = b.ink;
        page->items.append(text);
        break;
    }
    case LayoutBlock::KIND_SECTION:
        drawSection(b, y, page);
        break;
    case LayoutBlock::KIND_VALUE_ROW:
        drawValueRow(b, y, page);
        break;
    case LayoutBlock::KIND_TAP_ROW:
        drawTapRow(b, y, page);
        break;
    case LayoutBlock::KIND_TOGGLE:
        drawToggle(b, y, hostTexts, page);
        break;
    case LayoutBlock::KIND_RADIO_OPTION:
        drawRadioOption(b, y, page);
        break;
    case LayoutBlock::KIND_FIELD_BOX:
        drawFieldBox(b, y, page);
        break;
    case LayoutBlock::KIND_BIG_CODE: {
        PaintItem code = textItem(GUTTER, y, CONTENT_WIDTH, BIG_CODE_HEIGHT, b.primary, BIG_CODE_SIZE, FAMILY_MONO,
                                  PaintItem::ALIGN_CENTER);
        code.isBold = true;
        page->items.append(code);
        break;
    }
    case LayoutBlock::KIND_QR:
        drawQr(b, y, page);
        break;
    case LayoutBlock::KIND_LINE:
        drawLine(b, y, false, page);
        break;
    case LayoutBlock::KIND_DETAIL_LINE:
        drawLine(b, y, true, page);
        break;
    case LayoutBlock::KIND_CENTERED_LINE: {
        PaintItem line = textItem(GUTTER, y + 24, CONTENT_WIDTH, 56,
                                  elided(b.primary, CONTENT_WIDTH, ROW_TEXT_SIZE, FAMILY_SANS, false), ROW_TEXT_SIZE,
                                  FAMILY_SANS, PaintItem::ALIGN_CENTER);
        line.ink = INK_MUTED;
        page->items.append(line);
        break;
    }
    case LayoutBlock::KIND_NOTICE:
        drawNotice(b, y, page);
        break;
    case LayoutBlock::KIND_BUTTON:
        drawButton(b, y, page);
        break;
    }
}
