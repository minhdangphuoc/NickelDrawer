#include "tile_menu.h"

#include "debug_log.h"

#include "dialogs.h"
#include "fallback_strings.h"

TileMenu::TileMenu(const QString &menuTitle, const QList<TileMenuItem> &menuItems)
    : QObject(NULL), title(menuTitle), items(menuItems), confirmingIndex(-1)
{
}

void TileMenu::popup(const QPoint &anchor)
{
    QList<DialogChoice> choices;
    for (int i = 0; i < items.size(); i++) {
        DialogChoice choice = {QString::number(i), items[i].label};
        choices.append(choice);
    }
    RowMenu *menu = new RowMenu(title, choices);
    connect(menu, SIGNAL(chosen(QString)), this, SLOT(onMenuChoice(QString)));
    menu->popup(anchor);
}

void TileMenu::onMenuChoice(const QString &choiceId)
{
    bool isIndex = false;
    int index = choiceId.toInt(&isIndex);
    if (!isIndex || index < 0 || index >= items.size()) {
        DebugLog::write("tile menu: dismissed");
        finish();
        return;
    }
    DebugLog::write("tile menu: \"%s\" chosen%s", qPrintable(items[index].label), items[index].hasConfirm ? ", asking first" : "");
    if (!items[index].hasConfirm) {
        emit itemChosen(index);
        finish();
        return;
    }
    const DeclaredDialog &confirm = items[index].confirm;
    QString cancel = confirm.cancel.isEmpty() ? FallbackStrings::text(FallbackStrings::CANCEL) : confirm.cancel;
    confirmingIndex = index;
    ConfirmQuestion *question =
        new ConfirmQuestion(confirm.title.isEmpty() ? title : confirm.title, confirm.text, confirm.ok, cancel, ConfirmQuestion::DESTRUCTIVE);
    connect(question, SIGNAL(answered(bool)), this, SLOT(onConfirmAnswer(bool)));
    question->ask();
}

void TileMenu::onConfirmAnswer(bool isAccepted)
{
    int index = confirmingIndex;
    confirmingIndex = -1;
    DebugLog::write("tile menu: confirm %s", isAccepted ? "accepted" : "declined");
    if (isAccepted && index >= 0 && index < items.size()) {
        emit itemChosen(index);
    }
    finish();
}

void TileMenu::finish()
{
    emit closed();
    deleteLater();
}
