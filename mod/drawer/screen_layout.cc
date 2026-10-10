#include "screen_layout.h"

#include <QVector>

#include "copperui/button.h"
#include "copperui/divider.h"
#include "copperui/list_row.h"
#include "copperui/pager.h"
#include "qr_encoder.h"
#include "row_menu_logic.h"
#include "text_meter.h"

using namespace ScreenLogic;

// Shown only when the Tabler chevron PNGs are not installed (the control keeps working).
static const char *const CHEVRON_NEXT_FALLBACK = "\xE2\x80\xBA"; // ›
static const char *const CHEVRON_PREVIOUS_FALLBACK = "\xE2\x80\xB9"; // ‹
static const int FIELD_CAPTION_GAP = tokens::SPACE_FIELD_CAPTION;
static const int RADIO_RING_BORDER = tokens::STROKE_SELECT;
static const int RADIO_DOT = tokens::RADIO_DOT;
static const int BUTTON_BORDER = tokens::STROKE_BUTTON;
static const int ROW_DETAIL_WIDTH = tokens::ROW_DETAIL_WIDTH;
static const int ROW_DETAIL_RIGHT = tokens::ROW_DETAIL_RIGHT;
static const int TAP_ROW_TEXT_INSET = tokens::ROW_TAP_TEXT_INSET;
static const int FIELD_BORDER = tokens::STROKE_RULE;
static const int QR_FRAME_BORDER = tokens::STROKE_RULE;
static const int GALLERY_EMPTY_SPACER_HEIGHT = tokens::GALLERY_EMPTY_SPACER;
static const int CHEVRON_SIDE = tokens::ICON_CHEVRON;

// ---- Text helpers (Qt strings over the UTF-8 rules of screen_logic) ----

// Paragraphs wrap freely (maxLines 0); a button label keeps BUTTON_MAX_LINES lines and elides the rest.
static int widthOf(const QString &text, int size, Family family, bool isBold = false)
{
    QByteArray utf8 = text.toUtf8();
    TextMeter meter = QtTextMeter::meter();
    TextRun run = {size, family, isBold};
    return meter.measure(meter.context, utf8.constData(), utf8.size(), run);
}

static QStringList wrap(const QString &text, int width, int size, Family family, bool isBold = false, int maxLines = 0)
{
    QByteArray utf8 = text.toUtf8();
    QVector<char> out(utf8.size() * 2 + 16);
    TextRun run = {size, family, isBold};
    int count = wrapText(utf8.constData(), width, run, QtTextMeter::meter(), maxLines, out.data(), out.size());
    return count == 0 ? QStringList() : QString::fromUtf8(out.constData()).split(QLatin1Char('\n'));
}

// One line for a fixed-size control: the text, or its measured prefix and "…".
static QString elided(const QString &text, int width, int size, Family family, bool isMiddle, bool isBold = false)
{
    QByteArray utf8 = text.toUtf8();
    QVector<char> out(utf8.size() + 16);
    TextRun run = {size, family, isBold};
    elideText(utf8.constData(), width, run, QtTextMeter::meter(), isMiddle, out.data(), out.size());
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
           qrSide == other.qrSide && qrModules == other.qrModules && imagePath == other.imagePath &&
           isDashed == other.isDashed && showsMissingGlyph == other.showsMissingGlyph && isOn == other.isOn &&
           iconId == other.iconId;
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

// The two divider kinds of copperui/divider.h as paint items.
static PaintItem dividerItem(copperui::v1::DividerKind kind, int y)
{
    copperui::v1::Divider line = copperui::v1::divider(kind, y);
    return rectItem(line.rect.x, line.rect.y, line.rect.width, line.rect.height, line.ink, 0);
}

// A row divider: 2 px in the row line grey.
static PaintItem separatorItem(int y)
{
    return dividerItem(copperui::v1::DIVIDER_ROW, y);
}

// The 2 px rule under a section label stays black.
static PaintItem sectionRuleItem(int y)
{
    return dividerItem(copperui::v1::DIVIDER_SECTION, y);
}

static PaintItem pictureIcon(const QRect &rect, int ink)
{
    PaintItem item = rectItem(rect.x(), rect.y(), rect.width(), rect.height(), NO_FILL, 0);
    item.shape = PaintItem::SHAPE_PICTURE_ICON;
    item.ink = ink;
    return item;
}

// A Tabler / Nickel icon from system_icon.h at its own size, centred in rect; fallbackText is
// drawn instead when the icon is not installed ("" for nothing).
static PaintItem iconItem(const QRect &rect, SystemIcon::Id id, int ink, const char *fallbackText)
{
    PaintItem item = rectItem(rect.x(), rect.y(), rect.width(), rect.height(), NO_FILL, 0);
    item.shape = PaintItem::SHAPE_ICON;
    item.iconId = id;
    item.ink = ink;
    item.size = ROW_TEXT_SIZE;
    item.lineHeight = rect.height();
    item.align = PaintItem::ALIGN_CENTER;
    item.lines << QString::fromUtf8(fallbackText);
    return item;
}

static QRect toQRect(const IntRect &rect)
{
    return QRect(rect.x, rect.y, rect.width, rect.height);
}

static QRect toQRect(const copperui::v1::Rect &rect)
{
    return QRect(rect.x, rect.y, rect.width, rect.height);
}

static void addTarget(ScreenPage *page, const QRect &rect, int component, const QString &value, int listRow = -1)
{
    TapTarget target;
    target.kind = listRow >= 0 ? TapTarget::TAP_LIST_ROW : TapTarget::TAP_COMPONENT;
    target.rect = rect;
    target.component = component;
    target.value = value;
    target.row = listRow;
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
    result.size.keepWithNext = false;
    return result;
}

// The block stays on the page of the one after it (copperui/pack.h).
static LayoutBlock keptWithNext(LayoutBlock result)
{
    result.size.keepWithNext = true;
    return result;
}

static LayoutBlock paragraph(int component, const QString &text, TextStyle style, Family family, bool isBold,
                             PaintItem::Align align, int ink)
{
    LayoutBlock result = block(LayoutBlock::KIND_PARAGRAPH, component, 0, RHYTHM);
    result.textSize = textSize(style);
    result.lineHeight = lineHeight(style);
    result.lines = wrap(text, CONTENT_WIDTH, result.textSize, family, isBold);
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
    return keptWithNext(result); // a section label is never alone at the bottom of a page
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
    LayoutBlock code = qrBlock(component, address);
    code.size.keepWithNext = !address.isEmpty(); // the code and the address under it
    blocks->append(code);
    if (!address.isEmpty()) {
        blocks->append(singleLine(component, LayoutBlock::KIND_LINE, address, STYLE_CAPTION, FAMILY_MONO, INK_BLACK));
        blocks->last().align = PaintItem::ALIGN_CENTER;
    }
    if (!caption.isEmpty()) {
        blocks->append(paragraph(component, caption, STYLE_CAPTION, FAMILY_SANS, false, PaintItem::ALIGN_CENTER, INK_MUTED));
    }
}

// What decides how a button looks: its flags and the text that replaces its label while it works.
struct ButtonState {
    bool isEnabled;
    bool isPending;
    bool isAsync;
    const QString *pendingLabel;
};

static ButtonState buttonStateOf(const ScreenComponent &c, const HostTexts &texts, bool isEnabled)
{
    ButtonState state = {isEnabled, c.isPending, c.isAsync, &texts.pending};
    return state;
}

// The label wraps to at most BUTTON_MAX_LINES centred lines; a pending quick action shows the
// generic "Working…" instead (A9), in the normal look and without a target.
static LayoutBlock buttonBlock(int component, const QString &label, bool isPrimary, const ButtonState &state)
{
    copperui::v1::ButtonPresentation look =
        copperui::v1::buttonPresentation(state.isEnabled, state.isPending, state.isAsync);
    LayoutBlock result = block(LayoutBlock::KIND_BUTTON, component, 0, RHYTHM);
    const QString &shown = look.showsPendingLabel ? *state.pendingLabel : label;
    result.lines = wrap(shown, CONTENT_WIDTH - 2 * RHYTHM, BODY_SIZE, FAMILY_SANS, true, BUTTON_MAX_LINES);
    result.size.height = buttonHeight(result.lines.size());
    result.isOn = isPrimary;
    result.isTappable = look.isTappable;
    result.isDimmed = look.isDimmed;
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

// The title is a caption above the options (not a section label), kept on the page with them.
static void addRadioBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    int first = blocks->size();
    int titleInk = c.isEnabled ? INK_MUTED : INK_DIM;
    LayoutBlock title = paragraph(component, c.text, STYLE_CAPTION, FAMILY_SANS, false, PaintItem::ALIGN_START, titleInk);
    title.size.gap = FIELD_CAPTION_GAP;
    blocks->append(keptWithNext(title));
    for (int i = 0; i < c.options.size(); i++) {
        QString detail = i < c.optionDetails.size() ? c.optionDetails[i] : QString();
        LayoutBlock option = block(LayoutBlock::KIND_RADIO_OPTION, component, detail.isEmpty() ? ROW_HEIGHT : TAP_ROW_HEIGHT, 0);
        option.primary = c.options[i].second;
        option.secondary = detail;
        option.tapValue = c.options[i].first;
        option.isOn = c.options[i].first == c.value;
        option.isTappable = c.isEnabled;
        option.isDimmed = !c.isEnabled;
        option.size.keepWithNext = i < c.options.size() - 1; // all options stay with the title
        blocks->append(option);
    }
    withTrailingGap(blocks, first);
}

static void addFieldBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    LayoutBlock caption = paragraph(component, c.text, STYLE_CAPTION, FAMILY_SANS, false, PaintItem::ALIGN_START,
                                    c.isEnabled ? INK_MUTED : INK_DIM);
    caption.size.gap = FIELD_CAPTION_GAP;
    blocks->append(keptWithNext(caption)); // the caption and its box are never split
    LayoutBlock box = block(LayoutBlock::KIND_FIELD_BOX, component, FIELD_HEIGHT, RHYTHM);
    bool isEmpty = c.value.isEmpty();
    box.primary = isEmpty ? c.placeholder : c.value;
    box.ink = isEmpty || !c.isEnabled ? INK_DIM : INK_BLACK;
    box.isTappable = c.isEnabled;
    box.isDimmed = !c.isEnabled;
    blocks->append(box);
}

static void addPairBlocks(int component, const ScreenComponent &c, QList<LayoutBlock> *blocks)
{
    if (!c.title.isEmpty()) {
        blocks->append(keptWithNext(paragraph(component, c.title, STYLE_HEADING, FAMILY_SERIF, true, PaintItem::ALIGN_CENTER, INK_BLACK)));
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
    line.size.keepWithNext = !c.detail.isEmpty();
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
        const QString &secondary = c.items[i].second;
        LayoutBlock row = block(LayoutBlock::KIND_LIST_ROW, component, ROW_HEIGHT, 0);
        row.primary = primary;
        row.secondary = secondary;
        row.hasCheckMark = copperui::v1::isCheckMarkText(secondary.toUtf8().constData());
        row.isMiddleElided = isFileName(primary);
        row.isTappable = RowMenuLogic::isRowTarget(!c.tap.isEmpty(), c.menu.size());
        row.tapValue = primary;
        row.listRow = i;
        blocks->append(row);
    }
    withTrailingGap(blocks, first);
}

static void addEmptyBlocks(int component, const ScreenComponent &c, const HostTexts &texts, QList<LayoutBlock> *blocks)
{
    blocks->append(block(LayoutBlock::KIND_SPACER, component, EMPTY_SPACER_HEIGHT, 0));
    blocks->append(paragraph(component, c.text, STYLE_HEADING, FAMILY_SERIF, true, PaintItem::ALIGN_CENTER, INK_BLACK));
    if (!c.caption.isEmpty()) {
        blocks->append(paragraph(component, c.caption, STYLE_CAPTION, FAMILY_SANS, false, PaintItem::ALIGN_CENTER, INK_MUTED));
    }
    if (!c.button.isEmpty()) {
        blocks->append(buttonBlock(component, c.button, true, buttonStateOf(c, texts, true)));
    }
}

// The gallery's own empty state: picture icon, heading, text, primary button.
void ScreenLayout::addGalleryEmptyBlocks(int component, QList<LayoutBlock> *blocks) const
{
    const GalleryEmpty &empty = components[component].galleryEmpty;
    if (empty.title.isEmpty()) {
        return;
    }
    blocks->append(block(LayoutBlock::KIND_SPACER, component, GALLERY_EMPTY_SPACER_HEIGHT, 0));
    blocks->append(block(LayoutBlock::KIND_PICTURE_ICON, component, GALLERY_EMPTY_ICON_SIDE, RHYTHM)); // icon.grid
    blocks->append(paragraph(component, empty.title, STYLE_HEADING, FAMILY_SERIF, true, PaintItem::ALIGN_CENTER, INK_BLACK));
    if (!empty.text.isEmpty()) {
        blocks->append(paragraph(component, empty.text, STYLE_CAPTION, FAMILY_SANS, false, PaintItem::ALIGN_CENTER, INK_MUTED));
    }
    if (!empty.button.isEmpty() && !empty.action.isEmpty()) {
        blocks->append(buttonBlock(component, empty.button, true, buttonStateOf(components[component], hostTexts, true)));
    }
}

void ScreenLayout::addComponentBlocks(int index, QList<LayoutBlock> *blocks) const
{
    const ScreenComponent &c = components[index];
    const QString &type = c.type;
    if (type == QLatin1String("text")) {
        TextStyle style = styleOf(c.style);
        Family family = style == STYLE_HEADING ? FAMILY_SERIF : FAMILY_SANS; // in-page titles are serif
        blocks->append(paragraph(index, c.text, style, family, style == STYLE_HEADING, alignOf(c.align),
                                 style == STYLE_CAPTION ? INK_MUTED : INK_BLACK));
    } else if (type == QLatin1String("section")) {
        blocks->append(sectionBlock(index, c.text));
    } else if (type == QLatin1String("kv")) {
        addKeyValueBlocks(index, c, blocks);
    } else if (type == QLatin1String("row")) {
        LayoutBlock row = block(LayoutBlock::KIND_TAP_ROW, index, TAP_ROW_HEIGHT, RHYTHM);
        bool isTappable = !c.action.isEmpty() || !c.push.isEmpty();
        bool isPendingRow = isTappable && c.isPending && !c.isAsync; // A9: label replaced, no target, no chevron
        row.primary = isPendingRow ? hostTexts.pending : c.text;
        row.secondary = isPendingRow ? QString() : c.detail;
        row.isTappable = isTappable && !isPendingRow;
        blocks->append(row);
    } else if (type == QLatin1String("toggle")) {
        LayoutBlock toggle = block(LayoutBlock::KIND_TOGGLE, index, toggleRow(0, !c.detail.isEmpty()).height, RHYTHM);
        toggle.primary = c.text;
        toggle.secondary = c.detail;
        toggle.isOn = c.isOn;
        toggle.isTappable = c.isEnabled;
        toggle.isDimmed = !c.isEnabled;
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
        blocks->append(buttonBlock(index, c.text, c.isPrimaryButton(), buttonStateOf(c, hostTexts, c.isEnabled)));
    } else if (type == QLatin1String("list")) {
        addListBlocks(index, c, blocks);
    } else if (type == QLatin1String("empty")) {
        addEmptyBlocks(index, c, hostTexts, blocks);
    } else if (type == QLatin1String("gallery")) {
        addGalleryEmptyBlocks(index, blocks); // a gallery with pictures is a grid, not blocks
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

ScreenLayout::ScreenLayout()
    : height(PAGE_HEIGHT), galleryIndex(-1), hasGrid(false), gridTop(0), viewerIndex(-1), shownIndex(-1),
      isChromeShown(true)
{
    frame.top = frame.bottom = frame.footerTop = frame.pageBarTop = 0;
    frame.pageCount = 1;
    gridFrame.columns = GALLERY_DEFAULT_COLUMNS;
    gridFrame.rows = gridFrame.perPage = gridFrame.pageCount = 1;
}

static int indexOfType(const QList<ScreenComponent> &list, const char *type)
{
    for (int i = 0; i < list.size(); i++) {
        if (list[i].type == QLatin1String(type)) {
            return i;
        }
    }
    return -1;
}

ScreenLayout::ScreenLayout(const ScreenModel &screenModel, int viewHeight, const HostTexts &texts)
    : model(screenModel), hostTexts(texts), height(viewHeight), galleryIndex(-1), hasGrid(false), gridTop(0),
      viewerIndex(-1), shownIndex(-1), isChromeShown(true)
{
    components = model.body + model.footer;
    galleryIndex = indexOfType(model.body, "gallery");
    viewerIndex = indexOfType(model.body, "viewer");
    hasGrid = galleryIndex >= 0 && model.body[galleryIndex].tiles.size() + model.body[galleryIndex].pending > 0;
    if (viewerIndex >= 0) {
        const ScreenComponent &viewer = model.body[viewerIndex];
        showViewerPicture(viewer.value);
    }
    // A viewer is alone on its screen; components after a grid are not shown.
    int bodyEnd = hasGrid ? galleryIndex : model.body.size();
    for (int i = 0; i < bodyEnd && viewerIndex < 0; i++) {
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
    gridFrame.columns = GALLERY_DEFAULT_COLUMNS;
    gridFrame.rows = gridFrame.perPage = gridFrame.pageCount = 1;
    if (hasGrid) {
        planGrid(footerHeight, bodySizes);
    }
}

// The grid takes what the blocks before it and the footer leave; every grid
// page repeats those blocks (docs/spec/wallpaper-gallery.md section 2.1).
void ScreenLayout::planGrid(int footerHeight, const QVector<Block> &bodySizes)
{
    const ScreenComponent &gallery = components[galleryIndex];
    gridTop = frame.top;
    for (int i = 0; i < bodySizes.size(); i++) {
        gridTop += bodySizes[i].height + bodySizes[i].gap;
    }
    int bottom = footerHeight > 0 ? frame.footerTop - RHYTHM : height - GALLERY_BOTTOM_MARGIN;
    gridFrame = galleryFrame(gallery.gridColumns, gallery.gridRows, gallery.tiles.size() + gallery.pending, bottom - gridTop);
    frame.pageCount = gridFrame.pageCount;
    frame.pageBarTop = bottom - PAGE_BAR_HEIGHT;
}

int ScreenLayout::pageOfGalleryTile(const QString &id) const
{
    if (!hasGrid) {
        return -1;
    }
    const QList<PictureTile> &tiles = components[galleryIndex].tiles;
    for (int i = 0; i < tiles.size(); i++) {
        if (tiles[i].id == id) {
            return galleryPageOfSlot(i, gridFrame.perPage);
        }
    }
    return -1;
}

const PictureTile *ScreenLayout::viewerShownTile() const
{
    return viewerIndex >= 0 && shownIndex >= 0 && shownIndex < viewerCount() ? &components[viewerIndex].tiles[shownIndex]
                                                                                : NULL;
}

int ScreenLayout::viewerIndexOfTile(const QString &id) const
{
    if (viewerIndex < 0) {
        return -1;
    }
    const QList<PictureTile> &tiles = components[viewerIndex].tiles;
    for (int i = 0; i < tiles.size(); i++) {
        if (tiles[i].id == id) {
            return i;
        }
    }
    return -1;
}

// An unknown id shows the first picture (the backend does the same).
void ScreenLayout::showViewerPicture(const QString &id)
{
    int found = viewerIndexOfTile(id);
    shownIndex = viewerCount() == 0 ? -1 : (found < 0 ? 0 : found);
}

void ScreenLayout::showViewerPictureAt(int index)
{
    shownIndex = index >= 0 && index < viewerCount() ? index : shownIndex;
}

ScreenPage ScreenLayout::page(int index) const
{
    ScreenPage result;
    if (viewerIndex >= 0) {
        drawViewerPage(&result);
        return result;
    }
    int y = frame.top;
    for (int i = 0; i < body.size(); i++) {
        if (hasGrid || pageOfBlock[i] == index) {
            drawBlock(body[i], y, &result);
            y += body[i].size.height + body[i].size.gap;
        }
    }
    if (hasGrid) {
        drawGalleryPage(index, &result);
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

// The pager (copperui/pager.h): "n / N" between two Tabler chevrons; the two halves of the line
// are the touch targets (design-system.md 4.19). Only drawn when the screen has more than one page.
void ScreenLayout::drawPageBar(int index, ScreenPage *page) const
{
    copperui::v1::PagerGeometry pager = copperui::v1::pagerGeometry(frame.pageBarTop);
    PaintItem label = textItem(pager.label.x, pager.label.y, pager.label.width, pager.label.height,
                               QStringLiteral("%1 / %2").arg(index + 1).arg(frame.pageCount), CAPTION_SIZE, FAMILY_SANS,
                               PaintItem::ALIGN_CENTER);
    label.ink = INK_MUTED;
    page->items.append(label);
    page->items.append(iconItem(toQRect(pager.previousArrow), SystemIcon::CHEVRON_LEFT, INK_BLACK, CHEVRON_PREVIOUS_FALLBACK));
    page->items.append(iconItem(toQRect(pager.nextArrow), SystemIcon::CHEVRON_RIGHT, INK_BLACK, CHEVRON_NEXT_FALLBACK));
    TapTarget previous;
    previous.kind = TapTarget::TAP_PREVIOUS_PAGE;
    previous.rect = toQRect(pager.previousTarget);
    TapTarget next;
    next.kind = TapTarget::TAP_NEXT_PAGE;
    next.rect = toQRect(pager.nextTarget);
    page->targets << previous << next;
}

// ---- Drawing one block (components.go draw functions) ----

static void drawValueRow(const LayoutBlock &b, int y, ScreenPage *page)
{
    int valueWidth = CONTENT_WIDTH - KEY_COLUMN_WIDTH - RHYTHM;
    page->items.append(textItem(GUTTER, y + ROW_LABEL_TOP, KEY_COLUMN_WIDTH, ROW_LABEL_HEIGHT,
                                elided(b.primary, KEY_COLUMN_WIDTH, ROW_TEXT_SIZE, FAMILY_SANS, false), ROW_TEXT_SIZE,
                                FAMILY_SANS, PaintItem::ALIGN_START));
    PaintItem value = textItem(GUTTER + KEY_COLUMN_WIDTH + RHYTHM, y + ROW_LABEL_TOP, valueWidth, ROW_LABEL_HEIGHT,
                               elided(b.secondary, valueWidth, ROW_TEXT_SIZE, FAMILY_SANS, b.isMiddleElided, true),
                               ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_END);
    value.isBold = true;
    page->items.append(value);
    page->items.append(separatorItem(y + ROW_HEIGHT - SEPARATOR));
    if (b.isTappable) {
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, ROW_HEIGHT), b.component, b.tapValue, b.listRow);
    }
}

// A list row (copperui/list_row.h): the primary text takes the row less the muted secondary
// caption; a file name is elided in the middle; "✓" is the check icon.
static void drawListRow(const LayoutBlock &b, int y, ScreenPage *page)
{
    int secondaryWidth = b.hasCheckMark || b.secondary.isEmpty() ? 0 : widthOf(b.secondary, CAPTION_SIZE, FAMILY_SANS);
    copperui::v1::ListRowGeometry row = copperui::v1::listRowGeometry(y, secondaryWidth, b.hasCheckMark);
    page->items.append(textItem(row.primary.x, row.primary.y, row.primary.width, row.primary.height,
                                elided(b.primary, row.primary.width, ROW_TEXT_SIZE, FAMILY_SANS, b.isMiddleElided),
                                ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_START));
    if (row.secondary.width > 0) {
        PaintItem secondary = textItem(row.secondary.x, row.secondary.y, row.secondary.width, row.secondary.height,
                                       elided(b.secondary, row.secondary.width, CAPTION_SIZE, FAMILY_SANS, false),
                                       CAPTION_SIZE, FAMILY_SANS, PaintItem::ALIGN_END);
        secondary.ink = INK_MUTED;
        page->items.append(secondary);
    }
    if (row.checkIcon.width > 0) {
        page->items.append(iconItem(toQRect(row.checkIcon), SystemIcon::CHECK, INK_BLACK, ""));
    }
    page->items.append(separatorItem(y + ROW_HEIGHT - SEPARATOR));
    if (b.isTappable) {
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, ROW_HEIGHT), b.component, b.tapValue, b.listRow);
    }
}

static void drawSection(const LayoutBlock &b, int y, ScreenPage *page)
{
    // Uppercase first: capitals are wider than the lower case they replace.
    PaintItem label = textItem(GUTTER, y + SECTION_LABEL_TOP, CONTENT_WIDTH, SECTION_LINE_HEIGHT,
                               elided(b.primary.toUpper(), CONTENT_WIDTH, SECTION_TEXT_SIZE, FAMILY_SANS, false, true),
                               SECTION_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_START);
    label.ink = INK_MUTED;
    label.isBold = true;
    page->items << label << sectionRuleItem(y + SECTION_HEIGHT - SEPARATOR);
}

static void drawTapRow(const LayoutBlock &b, int y, ScreenPage *page)
{
    int textWidth = CONTENT_WIDTH - TAP_ROW_TEXT_INSET;
    page->items << textItem(GUTTER, y + TAP_ROW_TEXT_TOP, textWidth, ROW_LABEL_HEIGHT,
                            elided(b.primary, textWidth, ROW_TEXT_SIZE, FAMILY_SANS, false), ROW_TEXT_SIZE, FAMILY_SANS,
                            PaintItem::ALIGN_START)
                << separatorItem(y + TAP_ROW_HEIGHT - SEPARATOR);
    if (!b.secondary.isEmpty()) {
        PaintItem detail = textItem(SCREEN_WIDTH - GUTTER - ROW_DETAIL_RIGHT, y + TAP_ROW_TEXT_TOP, ROW_DETAIL_WIDTH,
                                    ROW_LABEL_HEIGHT,
                                    elided(b.secondary, ROW_DETAIL_WIDTH, ROW_TEXT_SIZE, FAMILY_SANS, false),
                                    ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_END);
        detail.ink = INK_MUTED;
        page->items.append(detail);
    }
    if (b.isTappable) {
        QRect chevron(SCREEN_WIDTH - GUTTER - CHEVRON_SIDE, y + (TAP_ROW_HEIGHT - CHEVRON_SIDE) / 2, CHEVRON_SIDE,
                      CHEVRON_SIDE);
        page->items.append(iconItem(chevron, SystemIcon::CHEVRON_RIGHT, INK_BLACK, CHEVRON_NEXT_FALLBACK));
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, TAP_ROW_HEIGHT), b.component, QString());
    }
}

// A label (detail line under it, inside the row) and, at the right, Nickel's
// ToggleSwitch: the state word, uppercase like Nickel's, then its switch image.
static void drawToggle(const LayoutBlock &b, int y, const HostTexts &texts, ScreenPage *page)
{
    ToggleRow row = toggleRow(y, !b.secondary.isEmpty());
    int labelInk = b.isDimmed ? INK_DIM : INK_BLACK; // disabled: label, detail, word and switch at ink.dim
    PaintItem label = textItem(row.label.x, row.label.y, row.label.width, row.label.height,
                               elided(b.primary, row.label.width, ROW_TEXT_SIZE, FAMILY_SANS, false), ROW_TEXT_SIZE,
                               FAMILY_SANS, PaintItem::ALIGN_START);
    label.ink = labelInk;
    page->items.append(label);
    if (!b.secondary.isEmpty()) {
        PaintItem detail = textItem(row.detail.x, row.detail.y, row.detail.width, row.detail.height,
                                    elided(b.secondary, row.detail.width, CAPTION_SIZE, FAMILY_SANS, false),
                                    CAPTION_SIZE, FAMILY_SANS, PaintItem::ALIGN_START);
        detail.ink = b.isDimmed ? INK_DIM : INK_MUTED;
        page->items.append(detail);
    }
    QString state = (b.isOn ? texts.on : texts.off).toUpper();
    PaintItem word = textItem(row.state.x, row.state.y, row.state.width, row.state.height,
                              elided(state, row.state.width, TOGGLE_STATE_SIZE, FAMILY_SANS, false), TOGGLE_STATE_SIZE,
                              FAMILY_SANS, PaintItem::ALIGN_END);
    word.ink = labelInk;
    page->items.append(word);
    PaintItem image;
    image.shape = PaintItem::SHAPE_SWITCH;
    image.rect = toQRect(row.switchBox);
    image.isOn = b.isOn;
    image.ink = labelInk;
    page->items << image << separatorItem(y + row.height - SEPARATOR);
    if (b.isTappable) {
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, row.height), b.component, QString());
    }
}

static void drawRadioOption(const LayoutBlock &b, int y, ScreenPage *page)
{
    bool hasDetail = !b.secondary.isEmpty();
    int ink = b.isDimmed ? INK_DIM : INK_BLACK; // disabled: ring, dot, label and detail at ink.dim
    int ringTop = y + (b.size.height - RADIO_RING_SIDE) / 2;
    PaintItem ring = ellipseItem(GUTTER, ringTop, RADIO_RING_SIDE, NO_FILL, RADIO_RING_BORDER);
    ring.ink = ink;
    page->items.append(ring);
    if (b.isOn) {
        page->items.append(ellipseItem(GUTTER + RADIO_DOT_INSET, ringTop + RADIO_DOT_INSET, RADIO_DOT, ink, 0));
    }
    int textWidth = CONTENT_WIDTH - RADIO_RING_SIDE - RHYTHM;
    int textLeft = GUTTER + RADIO_RING_SIDE + RHYTHM;
    PaintItem label = textItem(textLeft, y + (hasDetail ? ROW_LABEL_TOP_WITH_DETAIL : ROW_LABEL_TOP), textWidth,
                               ROW_LABEL_HEIGHT, elided(b.primary, textWidth, ROW_TEXT_SIZE, FAMILY_SANS, false, b.isOn),
                               ROW_TEXT_SIZE, FAMILY_SANS, PaintItem::ALIGN_START);
    label.isBold = b.isOn;
    label.ink = ink;
    page->items.append(label);
    if (hasDetail) {
        PaintItem detail = textItem(textLeft, y + ROW_DETAIL_TOP, textWidth, CAPTION_LINE_HEIGHT,
                                    elided(b.secondary, textWidth, CAPTION_SIZE, FAMILY_SANS, false), CAPTION_SIZE,
                                    FAMILY_SANS, PaintItem::ALIGN_START);
        detail.ink = b.isDimmed ? INK_DIM : INK_MUTED;
        page->items.append(detail);
    }
    page->items.append(separatorItem(y + b.size.height - SEPARATOR));
    if (b.isTappable) {
        addTarget(page, QRect(GUTTER, y, CONTENT_WIDTH, b.size.height), b.component, b.tapValue);
    }
}

static void drawFieldBox(const LayoutBlock &b, int y, ScreenPage *page)
{
    int textWidth = CONTENT_WIDTH - 2 * RHYTHM;
    PaintItem text = textItem(GUTTER + RHYTHM, y + (FIELD_HEIGHT - FIELD_TEXT_HEIGHT) / 2, textWidth, FIELD_TEXT_HEIGHT,
                              elided(b.primary, textWidth, ROW_TEXT_SIZE, FAMILY_SANS, false), ROW_TEXT_SIZE,
                              FAMILY_SANS, PaintItem::ALIGN_START);
    text.ink = b.ink;
    PaintItem box = rectItem(GUTTER, y, CONTENT_WIDTH, FIELD_HEIGHT, NO_FILL, FIELD_BORDER);
    box.ink = b.isDimmed ? INK_DIM : INK_BLACK; // disabled: border and text at ink.dim, no target
    page->items << box << text;
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
                                     elided(b.title, inner, BODY_SIZE, FAMILY_SANS, false, true), BODY_SIZE, FAMILY_SANS,
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

// A bordered button in box: filled black when primary; dimmed when disabled; without a target when
// it is dimmed or pending (the label then already says "Working…"). value tells the session which
// button was tapped.
static void paintButton(const QRect &box, const QStringList &lines, bool isPrimary, bool isDimmed, bool isTappable,
                        int component, const QString &value, ScreenPage *page)
{
    PaintItem outline = rectItem(box.x(), box.y(), box.width(), box.height(), NO_FILL, BUTTON_BORDER);
    int ink = INK_BLACK;
    if (isPrimary) {
        outline.fill = INK_BLACK;
        ink = PAPER_WHITE;
    }
    if (isDimmed) {
        // Disabled: outline and label dimmed, no target (no press, taps ignored).
        outline.fill = NO_FILL;
        outline.ink = INK_DIM;
        ink = INK_DIM;
    }
    int textHeight = lines.size() * BODY_LINE_HEIGHT;
    PaintItem text = textItem(box.x() + RHYTHM, box.y() + (box.height() - textHeight) / 2, box.width() - 2 * RHYTHM,
                              textHeight, QString(), BODY_SIZE, FAMILY_SANS, PaintItem::ALIGN_CENTER);
    text.lines = lines;
    text.lineHeight = BODY_LINE_HEIGHT;
    text.isBold = true;
    text.ink = ink;
    page->items << outline << text;
    if (isTappable) {
        addTarget(page, box, component, value);
    }
}

static void drawButton(const LayoutBlock &b, int y, ScreenPage *page)
{
    paintButton(QRect(GUTTER, y, CONTENT_WIDTH, b.size.height), b.lines, b.isOn, b.isDimmed, b.isTappable, b.component,
                QString(), page);
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
    case LayoutBlock::KIND_LIST_ROW:
        drawListRow(b, y, page);
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
        page->items.append(code); // regular weight (type.bigCode)
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
        PaintItem line = textItem(GUTTER, y + ROW_LABEL_TOP, CONTENT_WIDTH, ROW_LABEL_HEIGHT,
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
    case LayoutBlock::KIND_PICTURE_ICON:
        page->items.append(pictureIcon(QRect((SCREEN_WIDTH - GALLERY_EMPTY_ICON_SIDE) / 2, y, GALLERY_EMPTY_ICON_SIDE,
                                             GALLERY_EMPTY_ICON_SIDE),
                                       INK_DIM));
        break;
    }
}

// ---- Gallery grid ----

// Every slot makes the same four items (frame, picture, badge, check), so a
// thumbnail that appears or a badge that moves changes one tile and the page
// is repainted partially; only a different number of slots is a new structure.
void ScreenLayout::drawGallerySlot(int slotIndex, int slotOnPage, ScreenPage *page) const
{
    const ScreenComponent &gallery = components[galleryIndex];
    bool isPlaceholder = slotIndex >= gallery.tiles.size();
    IntRect slot = galleryTileRect(slotOnPage, gridFrame.columns, gridTop);
    QRect box(slot.x, slot.y, slot.width, slot.height);
    bool isCurrent = !isPlaceholder && gallery.tiles[slotIndex].isCurrent;

    PaintItem frameItem = rectItem(box.x(), box.y(), box.width(), box.height(), NO_FILL,
                                   isPlaceholder ? GALLERY_PLACEHOLDER_BORDER : (isCurrent ? GALLERY_CURRENT_BORDER : GALLERY_BORDER));
    if (isPlaceholder) {
        frameItem.isDashed = true;
        frameItem.ink = INK_DIM;
    }
    page->items.append(frameItem);

    PaintItem picture = rectItem(box.x() + GALLERY_CURRENT_BORDER, box.y() + GALLERY_CURRENT_BORDER,
                                 box.width() - 2 * GALLERY_CURRENT_BORDER, box.height() - 2 * GALLERY_CURRENT_BORDER,
                                 NO_FILL, 0);
    picture.shape = PaintItem::SHAPE_IMAGE;
    picture.showsMissingGlyph = !isPlaceholder;
    picture.imagePath = isPlaceholder ? QString() : gallery.tiles[slotIndex].thumb;
    page->items.append(picture);

    QRect badge(box.right() + 1 - GALLERY_BADGE_INSET - GALLERY_BADGE_SIDE, box.y() + GALLERY_BADGE_INSET,
                GALLERY_BADGE_SIDE, GALLERY_BADGE_SIDE);
    page->items.append(ellipseItem(badge.x(), badge.y(), GALLERY_BADGE_SIDE, isCurrent ? INK_BLACK : NO_FILL, 0));
    PaintItem check = rectItem(badge.x(), badge.y(), badge.width(), badge.height(), NO_FILL, 0);
    check.shape = PaintItem::SHAPE_CHECK;
    check.isOn = isCurrent; // only the current picture shows its check
    check.ink = PAPER_WHITE;
    page->items.append(check);

    if (!isPlaceholder) {
        TapTarget target;
        target.kind = TapTarget::TAP_TILE;
        target.rect = box;
        target.component = galleryIndex;
        target.value = gallery.tiles[slotIndex].id;
        page->targets.append(target);
    }
}

void ScreenLayout::drawGalleryPage(int index, ScreenPage *page) const
{
    const ScreenComponent &gallery = components[galleryIndex];
    int slotCount = gallery.tiles.size() + gallery.pending;
    int first = index * gridFrame.perPage;
    for (int slot = 0; slot < gridFrame.perPage && first + slot < slotCount; slot++) {
        drawGallerySlot(first + slot, slot, page);
    }
}

// ---- Viewer ----

static void addZone(ScreenPage *page, TapTarget::Kind kind, const QRect &rect)
{
    TapTarget zone;
    zone.kind = kind;
    zone.rect = rect;
    page->targets.append(zone);
}

// Full view: a white disc with a thin outline and the header's "←", over
// whatever the picture has there. It lies over the previous-picture zone, so
// it comes after the zones and wins (the canvas takes the topmost target).
void ScreenLayout::drawViewerBackButton(ScreenPage *page) const
{
    QRect disc = toQRect(viewerBackDiscRect());
    page->items.append(ellipseItem(disc.x(), disc.y(), disc.width(), PAPER_WHITE, VIEWER_BACK_OUTLINE));
    QRect arrowBox = toQRect(viewerBackArrowRect());
    PaintItem arrow = rectItem(arrowBox.x(), arrowBox.y(), arrowBox.width(), arrowBox.height(), NO_FILL,
                               VIEWER_BACK_ARROW_STROKE);
    arrow.shape = PaintItem::SHAPE_BACK_ARROW;
    page->items.append(arrow);
    addZone(page, TapTarget::TAP_VIEWER_BACK, toQRect(viewerBackRect()));
}

// The picture fills the area (fitted by the painter); thirds of the area turn
// pages or toggle the chrome. Buttons come after the zones, so they win.
void ScreenLayout::drawViewerPage(ScreenPage *page) const
{
    const PictureTile *tile = viewerShownTile();
    if (!tile) {
        return;
    }
    const ScreenComponent &viewer = components[viewerIndex];
    QRect area = toQRect(viewerPictureArea(isChromeShown, height));
    PaintItem picture = rectItem(area.x(), area.y(), area.width(), area.height(), NO_FILL, 0);
    picture.shape = PaintItem::SHAPE_IMAGE;
    picture.imagePath = tile->image;
    page->items.append(picture);

    int third = area.width() / 3;
    addZone(page, TapTarget::TAP_VIEWER_PREVIOUS, QRect(area.x(), area.y(), third, area.height()));
    addZone(page, TapTarget::TAP_VIEWER_NEXT, QRect(area.right() + 1 - third, area.y(), third, area.height()));
    addZone(page, TapTarget::TAP_VIEWER_TOGGLE_CHROME, QRect(area.x() + third, area.y(), area.width() - 2 * third, area.height()));
    if (!isChromeShown) {
        drawViewerBackButton(page);
        return;
    }
    QRect primaryBox = toQRect(viewerPrimaryRect(height));
    if (viewer.secondary.isDefined()) {
        QRect secondaryBox = toQRect(viewerSecondaryRect(height));
        QStringList label(elided(viewer.secondary.label, secondaryBox.width() - 2 * RHYTHM, BODY_SIZE, FAMILY_SANS, false, true));
        paintButton(secondaryBox, label, false, false, true, viewerIndex, QStringLiteral("secondary"), page);
    } else {
        primaryBox.setLeft(GUTTER);
    }
    if (viewer.primary.isDefined()) {
        bool isCurrent = tile->isCurrent;
        const QString &text = isCurrent && !viewer.primary.labelCurrent.isEmpty() ? viewer.primary.labelCurrent : viewer.primary.label;
        QStringList label(elided(text, primaryBox.width() - 2 * RHYTHM, BODY_SIZE, FAMILY_SANS, false, true));
        paintButton(primaryBox, label, true, isCurrent, !isCurrent, viewerIndex, QStringLiteral("primary"), page);
    }
}
