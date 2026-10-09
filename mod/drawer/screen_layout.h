#ifndef NICKELDRAWER_SCREEN_LAYOUT_H
#define NICKELDRAWER_SCREEN_LAYOUT_H

// Lays out an app screen into drawing primitives in device pixels, as a
// one-to-one port of cmd/ndpreview (layout.go, components.go): the same
// blocks, sizes, wrapping, eliding and paging, so the Mac preview equals the
// device. The pure rules live in screen_logic; this file adds the texts.

#include <QByteArray>
#include <QList>
#include <QRect>
#include <QStringList>

#include "screen_logic.h"
#include "screen_model.h"

struct PaintItem {
    enum Shape { SHAPE_RECT, SHAPE_ELLIPSE, SHAPE_TEXT, SHAPE_QR };
    enum Align { ALIGN_START, ALIGN_CENTER, ALIGN_END };

    Shape shape = SHAPE_RECT;
    QRect rect;
    int fill = ScreenLogic::NO_FILL;
    int border = 0;
    QStringList lines;
    int size = 0;
    int lineHeight = 0;
    ScreenLogic::Family family = ScreenLogic::FAMILY_SANS;
    bool isBold = false;
    bool isUnderlined = false;
    Align align = ALIGN_START;
    int ink = ScreenLogic::INK_BLACK;
    QByteArray qrModules; // SHAPE_QR: side * side bytes, 1 = dark, quiet zone not included
    int qrSide = 0;

    bool operator==(const PaintItem &other) const;
    bool operator!=(const PaintItem &other) const { return !(*this == other); }
};

struct TapTarget {
    enum Kind { TAP_BACK, TAP_HEADER_ACTION, TAP_PREVIOUS_PAGE, TAP_NEXT_PAGE, TAP_COMPONENT };

    Kind kind = TAP_COMPONENT;
    QRect rect;
    int component = -1; // index into ScreenLayout::component()
    QString value;      // radio: the option id; list: the row's primary text
};

struct ScreenPage {
    QList<PaintItem> items;
    QList<TapTarget> targets;
};

// One vertical slice of the screen (layout.go "block"): what to draw and its size.
struct LayoutBlock {
    enum Kind {
        KIND_SPACER,
        KIND_PARAGRAPH,
        KIND_SECTION,
        KIND_VALUE_ROW,
        KIND_TAP_ROW,
        KIND_TOGGLE,
        KIND_RADIO_OPTION,
        KIND_FIELD_BOX,
        KIND_BIG_CODE,
        KIND_QR,
        KIND_LINE,
        KIND_DETAIL_LINE,
        KIND_CENTERED_LINE,
        KIND_NOTICE,
        KIND_BUTTON
    };

    Kind kind = KIND_SPACER;
    ScreenLogic::Block size = {0, 0};
    int component = -1;
    QStringList lines; // wrapped text of paragraphs, notices and buttons
    int textSize = 0;
    int lineHeight = 0;
    ScreenLogic::Family family = ScreenLogic::FAMILY_SANS;
    bool isBold = false;
    PaintItem::Align align = PaintItem::ALIGN_START;
    int ink = ScreenLogic::INK_BLACK;
    QString primary;
    QString secondary;
    QString title;
    bool isOn = false; // toggle on, radio option selected, button primary
    bool isTappable = false;
    bool isMiddleElided = false;
    QString tapValue;
    QByteArray qrModules;
    int qrSide = 0;
};

// Texts the host draws itself (ndpreview hostStrings).
struct HostTexts {
    QString on;
    QString off;
};

class ScreenLayout {
public:
    ScreenLayout();
    ScreenLayout(const ScreenModel &model, int viewHeight, const HostTexts &texts);

    int pageCount() const { return frame.pageCount; }
    // Header, the body blocks of page index, the page bar and the footer.
    ScreenPage page(int index) const;
    // Body components first, then footer components.
    int componentCount() const { return components.size(); }
    const ScreenComponent &component(int index) const { return components[index]; }

private:
    void addComponentBlocks(int index, QList<LayoutBlock> *blocks) const;
    void drawBlock(const LayoutBlock &block, int y, ScreenPage *page) const;
    void drawHeader(ScreenPage *page) const;
    void drawPageBar(int index, ScreenPage *page) const;

    ScreenModel model;
    HostTexts hostTexts;
    int height;
    QList<ScreenComponent> components;
    QList<LayoutBlock> body;
    QList<LayoutBlock> footer;
    QList<int> pageOfBlock;
    ScreenLogic::BodyFrame frame;
};

#endif
