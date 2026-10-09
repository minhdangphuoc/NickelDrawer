// Compiled with -frtti (see drawer_controller.h).

#include "drawer_controller.h"

#include "debug_log.h"

// Values taken from libnickel's own implementations (23828).
static const int NO_PREFERRED_SIZE = -1;
static const int NICKEL_DEFAULT_ORIENTATIONS = 5;
static const int NAV_SECTION_MORE = 6; // MoreController/WishlistController::navSection()
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
    return 0;
}

DrawerController::DrawerController(QWidget *view)
    : QObject(NULL), preparedView(view)
{
    setObjectName(QStringLiteral("nickelDrawerController"));
}

DrawerController::~DrawerController()
{
    DebugLog::write("drawer controller destroyed");
}

int DrawerController::navSection() const
{
    return NAV_SECTION_MORE;
}

void DrawerController::loadView()
{
    if (!preparedView) {
        DebugLog::write("drawer controller: loadView without a view (already deleted)");
        return;
    }
    DebugLog::write("drawer controller: loadView");
    loadedView = preparedView;
}

void DrawerController::viewWillAppear()
{
    DebugLog::write("drawer controller: viewWillAppear");
}

void DrawerController::viewWillBeDestroyed()
{
    DebugLog::write("drawer controller: viewWillBeDestroyed");
}

AbstractController *DrawerController::asNickelController()
{
    return static_cast<AbstractController *>(this);
}

bool DrawerController::hasNickelLayout()
{
    char *whole = reinterpret_cast<char *>(this);
    char *nickelPart = reinterpret_cast<char *>(asNickelController());
    return nickelPart - whole == NICKEL_CONTROLLER_OFFSET;
}
