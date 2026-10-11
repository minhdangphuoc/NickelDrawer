// Compiled with -frtti (see page_controller.h).

#include "page_controller.h"

#include "debug_log.h"

// Values taken from libnickel's own implementations (23828).
static const int NO_PREFERRED_SIZE = -1;
static const int NICKEL_DEFAULT_ORIENTATIONS = 5;
static const ptrdiff_t NICKEL_CONTROLLER_OFFSET = 8;

static_assert(sizeof(QObject) == 8, "QObject size differs from Nickel's Qt 5.2.1");
static_assert(sizeof(QPointer<QWidget>) == 8, "QPointer layout differs from Nickel's");
static_assert(sizeof(AbstractController) == 0xc, "AbstractController must match libnickel's size 0xc");

AbstractController::AbstractController() {}

AbstractController::~AbstractController() {}

QSize AbstractController::size()
{
    return QSize(NO_PREFERRED_SIZE, NO_PREFERRED_SIZE);
}

void AbstractController::viewWillAppear() {}

void AbstractController::viewWillDisappear() {}

void AbstractController::viewWillBeDestroyed() {}

int AbstractController::allowedOrientations() const
{
    return NICKEL_DEFAULT_ORIENTATIONS;
}

int AbstractController::navSection() const
{
    return PageController::NAV_SECTION_NONE;
}

PageController::PageController(QWidget *view, int navSection)
    : QObject(NULL), preparedView(view), section(navSection)
{
    setObjectName(QStringLiteral("nickelPageController"));
}

PageController::~PageController()
{
    DebugLog::write("page controller destroyed");
}

int PageController::navSection() const
{
    return section;
}

void PageController::loadView()
{
    if (!preparedView) {
        DebugLog::write("page controller: loadView without a view (already deleted)");
        return;
    }
    DebugLog::write("page controller: loadView");
    loadedView = preparedView;
}

void PageController::viewWillAppear()
{
    DebugLog::write("page controller: viewWillAppear");
}

void PageController::viewWillBeDestroyed()
{
    DebugLog::write("page controller: viewWillBeDestroyed");
}

AbstractController *PageController::asNickelController()
{
    return static_cast<AbstractController *>(this);
}

bool PageController::hasNickelLayout()
{
    char *whole = reinterpret_cast<char *>(this);
    char *nickelPart = reinterpret_cast<char *>(asNickelController());
    return nickelPart - whole == NICKEL_CONTROLLER_OFFSET;
}
