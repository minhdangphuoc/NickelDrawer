#include "copperui/button.h"

namespace copperui {
namespace v1 {

ButtonPresentation buttonPresentation(bool isEnabled, bool isPending, bool isAsync)
{
    ButtonPresentation presentation;
    presentation.showsPendingLabel = false;
    presentation.isDimmed = false;
    presentation.isTappable = true;
    if (!isEnabled) {
        presentation.isDimmed = true;
        presentation.isTappable = false;
    } else if (isPending && !isAsync) {
        presentation.showsPendingLabel = true;
        presentation.isTappable = false;
    }
    return presentation;
}

} // namespace v1
} // namespace copperui
