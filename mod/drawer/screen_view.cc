#include "screen_view.h"

#include <QEvent>
#include <QFontMetrics>
#include <QHBoxLayout>
#include <QLabel>
#include <QMouseEvent>
#include <QTouchEvent>

#include "debug_log.h"

#include "nickel_bridge.h"
#include "nickel_fonts.h"
#include "page_header.h"
#include "painted_marks.h"
#include "screen_canvas.h"
#include "screen_logic.h"
#include "status_bar_inset.h"

using namespace ScreenLogic;

// The layout leaves exactly the header's height free at the top (ndpreview: headerHeight).
static_assert(HEADER_HEIGHT == PageHeader::ROW_HEIGHT + PageHeader::DIVIDER_HEIGHT,
              "screen_logic.h HEADER_HEIGHT must match PageHeader");

// The header row's texts at the right, in Nickel's UI sans (never generic
// families: no Vietnamese). Sizes and greys are cmd/ndpreview's (layout.go).
static const char *const HEADER_SIDE_STYLE =
    "#screenHeaderStatus { font-family: @BODY_FAMILY@; font-size: 32px; color: #404040; background: white; }"
    "#screenHeaderAction QLabel { font-family: @BODY_FAMILY@; font-size: 32px; color: black;"
    " text-decoration: underline; background: white; }"
    "#screenHeaderAction { background: white; }";

// ---- Header action ----

HeaderActionButton::HeaderActionButton(QWidget *parent) : PressArea(parent), label(new QLabel(this))
{
    setObjectName(QStringLiteral("screenHeaderAction"));
    setFixedHeight(PageHeader::ROW_HEIGHT); // the whole row height: a 120 px tap target
    setMaximumWidth(HEADER_SIDE_WIDTH + RHYTHM);
    setPressHighlight(HIGHLIGHT_INVERT);
    label->setAlignment(Qt::AlignRight | Qt::AlignVCenter);
    label->setAttribute(Qt::WA_TransparentForMouseEvents, true);
    QHBoxLayout *layout = new QHBoxLayout(this);
    layout->setContentsMargins(RHYTHM, 0, 0, 0);
    layout->addWidget(label);
}

void HeaderActionButton::setText(const QString &text)
{
    label->ensurePolished(); // the style sheet's font, not the default one
    QFontMetrics metrics(label->font());
    label->setText(metrics.elidedText(text, Qt::ElideRight, HEADER_SIDE_WIDTH));
}

// ---- Screen view ----

ScreenView::ScreenView(const QString &title, QWidget *drawerView)
    : QWidget(drawerView), host(drawerView), canvas(NULL), header(NULL), statusLabel(NULL), actionButton(NULL),
      isInputBlocked(false)
{
    setObjectName(QStringLiteral("nickelDrawerScreen"));
    setStyleSheet(NickelFonts::styleSheet(HEADER_SIDE_STYLE));
    // Only to log touches that miss the canvas (event()); they are not accepted here.
    setAttribute(Qt::WA_AcceptTouchEvents, true);
    // The canvas first, the header after it: a later sibling lies on top.
    canvas = new ScreenCanvas(host, this);
    connect(canvas, SIGNAL(componentTapped(int, QString)), this, SIGNAL(componentTapped(int, QString)));
    connect(canvas, SIGNAL(tileLongPressed(int, QString, QPoint)), this, SIGNAL(tileLongPressed(int, QString, QPoint)));
    connect(canvas, SIGNAL(viewerPictureChanged(QString)), this, SIGNAL(viewerPictureChanged(QString)));
    connect(canvas, SIGNAL(viewerStateChanged()), this, SLOT(showHeader()));
    header = createHeader(title);
    canvas->logTapTarget("constructed");
    placeBelowStatusBar();
    host->installEventFilter(this);
    raise();
    show();
}

// The drawer's header (page_header.h) with the status text and the header
// action at the right of the same row.
PageHeader *ScreenView::createHeader(const QString &title)
{
    PageHeader *pageHeader = new PageHeader(title, NickelBridge::textHeaderApi(), this);
    connect(pageHeader, SIGNAL(backTapped()), this, SLOT(onBackTapped()));
    statusLabel = new QLabel(pageHeader);
    statusLabel->setObjectName(QStringLiteral("screenHeaderStatus"));
    statusLabel->setAlignment(Qt::AlignRight | Qt::AlignVCenter);
    statusLabel->setMaximumWidth(HEADER_SIDE_WIDTH);
    statusLabel->hide();
    actionButton = new HeaderActionButton(pageHeader);
    connect(actionButton, SIGNAL(tapped()), this, SLOT(onHeaderActionTapped()));
    actionButton->hide();
    pageHeader->addTrailingWidget(statusLabel);
    pageHeader->addTrailingWidget(actionButton);
    DebugLog::write("screen view: header created (%s back, title, status, action)",
                    pageHeader->usesNickelHeader() ? "Nickel's" : "own");
    return pageHeader;
}

// The screen covers the drawer page except the strip under Nickel's status bar.
void ScreenView::placeBelowStatusBar()
{
    int inset = StatusBarInset::topInsetFor(host);
    setGeometry(host->rect().adjusted(0, inset, 0, 0));
    StatusBarInset::logGeometry("screen placed", this, header);
}

// Header where the drawer has it (inside the side gutters, at the top); the
// canvas under the whole view, so layout coordinates are the view's.
void ScreenView::resizeEvent(QResizeEvent *event)
{
    QWidget::resizeEvent(event);
    header->setGeometry(GUTTER, 0, CONTENT_WIDTH, header->height());
    canvas->setGeometry(rect());
    header->raise();
}

// Geometry is final only once shown (the pending resize lays out the canvas).
void ScreenView::showEvent(QShowEvent *event)
{
    QWidget::showEvent(event);
    canvas->logTapTarget("shown");
    actionButton->logTapTarget("shown");
}

// Where a touch or mouse press landed (view coordinates); false for other events.
static bool inputPosition(QEvent *event, QPoint *position)
{
    if (event->type() == QEvent::TouchBegin) {
        const QList<QTouchEvent::TouchPoint> &points = static_cast<QTouchEvent *>(event)->touchPoints();
        *position = points.isEmpty() ? QPoint() : points.first().pos().toPoint();
        return true;
    }
    if (event->type() == QEvent::MouseButtonPress) {
        *position = static_cast<QMouseEvent *>(event)->pos();
        return true;
    }
    return false;
}

// The canvas covers this view and takes every touch, so a press arriving here
// means a body tap missed the canvas: the log names what lies under it.
bool ScreenView::event(QEvent *event)
{
    QPoint position;
    if (inputPosition(event, &position)) {
        QWidget *under = childAt(position);
        DebugLog::write("screen view: %s at %d,%d reached the view, not the canvas (canvas %dx%d visible %d; "
                        "under it: %s \"%s\")",
                        event->type() == QEvent::TouchBegin ? "touch begin" : "mouse press", position.x(),
                        position.y(), canvas->width(), canvas->height(), canvas->isVisible() ? 1 : 0,
                        under ? under->metaObject()->className() : "nothing",
                        under ? qPrintable(under->objectName()) : "");
    }
    return QWidget::event(event);
}

bool ScreenView::eventFilter(QObject *watched, QEvent *event)
{
    if (watched == host && event->type() == QEvent::Resize) {
        placeBelowStatusBar();
    }
    return false;
}

void ScreenView::showScreen(const ScreenModel &next)
{
    model = next;
    canvas->showScreen(model); // may call showHeader() through viewerStateChanged()
    showHeader();
}

// Title, status and action text; QLabel repaints only what changed (partial).
// The viewer's title is its position ("3 / 12") and its status marks the
// current wallpaper; with the chrome hidden the whole header is hidden.
void ScreenView::showHeader()
{
    bool isViewer = canvas->hasViewer();
    QString title = model.title;
    QString status = model.status;
    if (isViewer && canvas->viewerCount() > 0) {
        title = QStringLiteral("%1 / %2").arg(canvas->viewerPosition()).arg(canvas->viewerCount());
        status = canvas->isViewerPictureCurrent() ? canvas->viewerCurrentStatus() : QString();
    }
    header->setTitle(title);
    statusLabel->ensurePolished(); // the style sheet's font, not the default one
    QFontMetrics metrics(statusLabel->font());
    statusLabel->setText(metrics.elidedText(status, Qt::ElideRight, HEADER_SIDE_WIDTH));
    statusLabel->setVisible(!status.isEmpty());
    bool hasIcons = !model.headerActions.isEmpty();
    actionButton->setText(hasIcons ? QString() : model.headerLabel);
    actionButton->setVisible(!hasIcons && !model.headerLabel.isEmpty());
    showHeaderIcons();
    bool isHeaderWanted = !isViewer || canvas->isChromeShown();
    if (isHeaderWanted != header->isVisibleTo(this)) {
        DebugLog::write("screen view: header %s (viewer chrome %s)", isHeaderWanted ? "shown" : "hidden",
                        canvas->isChromeShown() ? "shown" : "hidden");
    }
    header->setVisible(isHeaderWanted);
}

// Up to two painted icon buttons at the right of the row. A button is rebuilt
// only when the icons differ; the old ones are deleted from the event loop.
void ScreenView::showHeaderIcons()
{
    QStringList names;
    for (int i = 0; i < model.headerActions.size(); i++) {
        names << model.headerActions[i].icon;
    }
    if (names == shownIconNames) {
        return;
    }
    shownIconNames = names;
    for (int i = 0; i < iconButtons.size(); i++) {
        iconButtons[i]->hide();
        iconButtons[i]->deleteLater();
    }
    iconButtons.clear();
    for (int i = 0; i < names.size(); i++) {
        HeaderIconButton::Icon icon;
        if (!HeaderIconButton::iconNamed(names[i], &icon)) {
            DebugLog::write("screen view: unknown header icon \"%s\" skipped", qPrintable(names[i]));
            continue;
        }
        HeaderIconButton *button = new HeaderIconButton(icon, model.headerActions[i].label, header);
        button->setObjectName(QStringLiteral("screenHeaderIcon%1").arg(i));
        button->setToolTip(model.headerActions[i].label);
        button->setProperty("actionIndex", i);
        button->setAcceptsPresses(!isInputBlocked);
        connect(button, SIGNAL(tapped()), this, SLOT(onHeaderIconTapped()));
        header->addTrailingWidget(button);
        button->show();
        iconButtons.append(button);
    }
    DebugLog::write("screen view: %d header icon button(s)", iconButtons.size());
}

void ScreenView::onHeaderIconTapped()
{
    QObject *button = sender();
    int index = button ? button->property("actionIndex").toInt() : -1;
    DebugLog::write("screen: header icon %d tapped%s", index, isInputBlocked ? ", ignored while an action is pending" : "");
    if (!isInputBlocked && index >= 0) {
        emit headerIconTapped(index);
    }
}

QString ScreenView::viewerPictureId() const
{
    return canvas->viewerPictureId();
}

void ScreenView::dropPictures()
{
    canvas->dropPictures();
}

void ScreenView::restoreTileOnNextScreen(const QString &id)
{
    canvas->restoreTileOnNextScreen(id);
}

int ScreenView::currentPage() const
{
    return canvas->currentPage();
}

void ScreenView::restorePageOnNextScreen(int page)
{
    canvas->restorePageOnNextScreen(page);
}

bool ScreenView::componentAt(int index, ScreenComponent *copy) const
{
    return canvas->componentAt(index, copy);
}

static ScreenComponent *componentIn(ScreenModel *model, int index)
{
    if (index >= 0 && index < model->body.size()) {
        return &model->body[index];
    }
    index -= model->body.size();
    return index >= 0 && index < model->footer.size() ? &model->footer[index] : NULL;
}

void ScreenView::showToggleState(int index, bool isOn)
{
    ScreenModel next = model;
    ScreenComponent *toggle = componentIn(&next, index);
    if (toggle && toggle->type == QLatin1String("toggle")) {
        toggle->isOn = isOn;
        showScreen(next);
    }
}

void ScreenView::showRadioValue(int index, const QString &value)
{
    ScreenModel next = model;
    ScreenComponent *radio = componentIn(&next, index);
    if (radio && radio->type == QLatin1String("radio")) {
        radio->value = value;
        showScreen(next);
    }
}

void ScreenView::setInputBlocked(bool isBlocked)
{
    isInputBlocked = isBlocked;
    header->setBackEnabled(!isBlocked);
    actionButton->setAcceptsPresses(!isBlocked);
    for (int i = 0; i < iconButtons.size(); i++) {
        iconButtons[i]->setAcceptsPresses(!isBlocked);
    }
    canvas->setInputBlocked(isBlocked);
}

// PageHeader and PressArea emit from the event loop: the slot may delete this view.
void ScreenView::onBackTapped()
{
    DebugLog::write("screen: back tapped on \"%s\"%s", qPrintable(model.screen),
                    isInputBlocked ? ", ignored while an action is pending" : "");
    if (!isInputBlocked) {
        emit backRequested();
    }
}

void ScreenView::onHeaderActionTapped()
{
    DebugLog::write("screen: header action \"%s\" tapped%s", qPrintable(model.headerLabel),
                    isInputBlocked ? ", ignored while an action is pending" : "");
    if (!isInputBlocked) {
        emit headerActionTapped();
    }
}
