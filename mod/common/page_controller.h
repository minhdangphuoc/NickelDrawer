#ifndef MODS_PAGE_CONTROLLER_H
#define MODS_PAGE_CONTROLLER_H

// A page of a mod as a Nickel controller: route A of the drawer (a "More
// sub-page", navSection 6) and modal pages (navSection 0: Nickel hides the
// tab bar and sets the status bar from the view's properties, as for its own
// Settings dialogs). Shared by every mod (each compiles its own copy).
//
// AbstractController below mirrors libnickel's class (4.38.x, decompiled on
// 23828, re/notes/G2-findings.md Q2): vptr, QPointer<QWidget> (+4 d, +8 widget),
// size 0xc; virtual slots in this exact order. Nickel calls these slots through
// the vtable and casts with __dynamic_cast, so:
//   - do not add, remove or reorder virtuals or data members;
//   - the mod must be compiled with -frtti (the Makefile does this for every
//     file), or the vtable has no typeinfo and Nickel's dynamic_cast crashes;
//   - PageController does not use Q_OBJECT, so moc cannot add virtuals.

#include <QObject>
#include <QPointer>
#include <QSize>
#include <QWidget>

class AbstractController {
public:
    AbstractController();
    virtual ~AbstractController();             // +0x00 D1, +0x04 D0
    virtual QSize size();                      // +0x08
    virtual void viewWillAppear();             // +0x0c
    virtual void viewWillDisappear();          // +0x10
    virtual void viewWillBeDestroyed();        // +0x14
    virtual int allowedOrientations() const;   // +0x18
    virtual int navSection() const;            // +0x1c (MainNavSection)
    virtual void loadView() = 0;               // +0x20

protected:
    QPointer<QWidget> loadedView;
};

// Wraps an already built view. Nickel calls loadView() during push(), then
// makes this controller a child of the view (QObject::setParent), so the view's
// deletion on pop/tab switch/Home also deletes the controller.
class PageController : public QObject, public AbstractController {
public:
    // MainNavSection values Nickel's own controllers return.
    static const int NAV_SECTION_NONE = 0; // no tab bar (AbstractController's default)
    static const int NAV_SECTION_MORE = 6; // MoreController/WishlistController::navSection()

    explicit PageController(QWidget *view, int navSection = NAV_SECTION_MORE);
    ~PageController();

    int navSection() const;
    void loadView();
    void viewWillAppear();
    void viewWillBeDestroyed();

    AbstractController *asNickelController();

    // False when the compiler laid the class out differently from Nickel's
    // expectation (AbstractController at offset 8); never push it then.
    bool hasNickelLayout();

private:
    QPointer<QWidget> preparedView;
    int section;
};

#endif
