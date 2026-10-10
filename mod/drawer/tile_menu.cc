#include "tile_menu.h"

#include "debug_log.h"

#include "dialogs.h"
#include "fallback_strings.h"

// "Remove…" opens a question; the button that answers it is "Remove".
static QString withoutTrailingEllipsis(QString label)
{
    if (label.endsWith(QChar(0x2026)) || label.endsWith(QLatin1String("..."))) {
        label.chop(label.endsWith(QChar(0x2026)) ? 1 : 3);
    }
    return label.trimmed();
}

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
    // `"confirm": true` declares no texts: ask about the chosen entry, answer with its name.
    QString actionName = withoutTrailingEllipsis(items[index].label);
    ConfirmQuestion *question = new ConfirmQuestion(confirm.title.isEmpty() ? title : confirm.title,
                                                    confirm.text.isEmpty() ? actionName + QLatin1Char('?') : confirm.text,
                                                    confirm.ok.isEmpty() ? actionName : confirm.ok, cancel,
                                                    ConfirmQuestion::DESTRUCTIVE);
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
