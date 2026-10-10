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
#include <QVector>

#include "screen_logic.h"
#include "screen_model.h"

struct PaintItem {
    // SHAPE_IMAGE: the picture at imagePath fitted inside rect; the picture glyph
    // when the path is empty or unreadable (unless showsMissingGlyph is off).
    // SHAPE_CHECK: a check mark in ink inside rect, border = stroke width (0: not drawn).
    // SHAPE_PICTURE_GLYPH: an outline picture icon in ink, border = stroke width.
    // SHAPE_BACK_ARROW: the header's drawn "←" in ink inside rect, border = stroke width.
    // SHAPE_SWITCH: Nickel's on or off switch image (isOn) centred in rect, never drawn by us.
    enum Shape {
        SHAPE_RECT,
        SHAPE_ELLIPSE,
        SHAPE_TEXT,
        SHAPE_QR,
        SHAPE_IMAGE,
        SHAPE_CHECK,
        SHAPE_PICTURE_GLYPH,
        SHAPE_BACK_ARROW,
        SHAPE_SWITCH
    };
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
    int ink = ScreenLogic::INK_BLACK; // text, and the border of rects and ellipses
    QByteArray qrModules; // SHAPE_QR: side * side bytes, 1 = dark, quiet zone not included
    int qrSide = 0;
    QString imagePath;            // compared, so a thumbnail that appears repaints its tile only
    bool isDashed = false;        // SHAPE_RECT border drawn dashed
    bool showsMissingGlyph = true;
    bool isOn = false;            // SHAPE_SWITCH: the on image

    bool operator==(const PaintItem &other) const;
    bool operator!=(const PaintItem &other) const { return !(*this == other); }
};

struct TapTarget {
    // TAP_LIST_ROW: a row of a list with a tap action and/or a menu (value = its
    // primary text, row = its index in the list); tap and long press.
    // TAP_TILE: a gallery picture (value = its id); tap and long press.
    // TAP_VIEWER_PREVIOUS/NEXT/TOGGLE_CHROME: the viewer's picture zones (no pressed feedback).
    // TAP_VIEWER_BACK: the full view's back button (circle feedback): shows the chrome again.
    enum Kind {
        TAP_PREVIOUS_PAGE,
        TAP_NEXT_PAGE,
        TAP_COMPONENT,
        TAP_TILE,
        TAP_LIST_ROW,
        TAP_VIEWER_PREVIOUS,
        TAP_VIEWER_NEXT,
        TAP_VIEWER_TOGGLE_CHROME,
        TAP_VIEWER_BACK
    };

    Kind kind = TAP_COMPONENT;
    QRect rect;
    int component = -1; // index into ScreenLayout::component()
    int row = -1;       // TAP_LIST_ROW: index of the row in the list's items
    QString value;      // radio: the option id; list: the row's primary text; tile: its id; viewer button: "primary" / "secondary"
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
        KIND_BUTTON,
        KIND_PICTURE_ICON
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
    int listRow = -1; // list rows: index in the list's items, for the row menu
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
    // The body blocks of page index, the page bar and the footer, in view
    // coordinates; the top HEADER_HEIGHT pixels are the ScreenView's header.
    ScreenPage page(int index) const;
    // Gallery: the grid page holding the tile, or -1 (also when there is no grid).
    int pageOfGalleryTile(const QString &id) const;
    int galleryComponent() const { return hasGrid ? galleryIndex : -1; }
    // Viewer: the pictures, the shown one and the chrome. Changing them only
    // changes what page(0) draws.
    int viewerComponent() const { return viewerIndex; }
    int viewerCount() const { return viewerIndex < 0 ? 0 : components[viewerIndex].tiles.size(); }
    int viewerShownIndex() const { return shownIndex; }
    const PictureTile *viewerShownTile() const;
    void showViewerPicture(const QString &id);
    // Index of the picture with this id, or -1.
    int viewerIndexOfTile(const QString &id) const;
    void showViewerPictureAt(int index);
    void setChromeShown(bool isShown) { isChromeShown = isShown; }
    bool chromeShown() const { return isChromeShown; }
    // Body components first, then footer components.
    int componentCount() const { return components.size(); }
    const ScreenComponent &component(int index) const { return components[index]; }

private:
    void addComponentBlocks(int index, QList<LayoutBlock> *blocks) const;
    void drawBlock(const LayoutBlock &block, int y, ScreenPage *page) const;
    void drawPageBar(int index, ScreenPage *page) const;
    void drawGalleryPage(int index, ScreenPage *page) const;
    void drawGallerySlot(int slotIndex, int slotOnPage, ScreenPage *page) const;
    void planGrid(int footerHeight, const QVector<ScreenLogic::Block> &bodySizes);
    void drawViewerPage(ScreenPage *page) const;
    void drawViewerBackButton(ScreenPage *page) const;
    void addGalleryEmptyBlocks(int component, QList<LayoutBlock> *blocks) const;

    ScreenModel model;
    HostTexts hostTexts;
    int height;
    QList<ScreenComponent> components;
    QList<LayoutBlock> body;
    QList<LayoutBlock> footer;
    QList<int> pageOfBlock;
    ScreenLogic::BodyFrame frame;
    int galleryIndex;
    bool hasGrid;
    ScreenLogic::GalleryFrame gridFrame;
    int gridTop;
    int viewerIndex;
    int shownIndex;
    bool isChromeShown;
};

#endif
