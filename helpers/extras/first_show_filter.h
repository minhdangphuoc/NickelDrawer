#ifndef EXTRAS_FIRST_SHOW_FILTER_H
#define EXTRAS_FIRST_SHOW_FILTER_H

class QWidget;

// Calls a handler once per widget of one class, on that widget's first QEvent::Show, from an event
// filter on the application. Hook-free: it reaches Nickel's widgets without patching libnickel, so it
// cannot collide with another mod's hook on the same constructor. Plain Qt; the class name comes
// from the caller (the bridge, which knows Nickel's classes).
//
// Cost: every event of the GUI thread passes the filter; anything that is not a Show event returns
// after one integer compare, and a Show is matched by one string compare of the class name.
namespace FirstShowFilter {

typedef void (*Handler)(QWidget *widget);

// Watches widgets whose metaObject()->className() is exactly className (kept, not copied: pass a
// string literal). Installs the filter on the application now if it exists, otherwise from
// QCoreApplication's constructor (qAddPreRoutine). Call once; later calls are ignored (logged).
void watch(const char *className, Handler handler);

} // namespace FirstShowFilter

#endif
