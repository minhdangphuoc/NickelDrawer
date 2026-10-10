#ifndef COPPERUI_BUTTON_H
#define COPPERUI_BUTTON_H

// What a button shows and does in each state (docs/design-system.md sections 3 and 4.10).
// Pure rules shared by the app-screen canvas and the preview.

namespace copperui {
namespace v1 {

struct ButtonPresentation {
    bool showsPendingLabel; // the label is replaced by the generic "Working..." text
    bool isDimmed;          // outline and label at ink.dim
    bool isTappable;        // has a touch target
};

// A disabled button is dimmed and has no target. A quick action (not async) that was just tapped
// shows the pending label until it returns: normal look, no target. An async button stays
// tappable while its job runs, so it never shows the pending label. Detached jobs are disabled
// by the app itself (its `enabled` binding) and show their progress as `steps`.
ButtonPresentation buttonPresentation(bool isEnabled, bool isPending, bool isAsync);

} // namespace v1
} // namespace copperui

#endif
