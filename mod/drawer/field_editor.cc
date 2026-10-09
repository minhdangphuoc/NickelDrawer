#include "field_editor.h"

#include <QFrame>
#include <QLineEdit>
#include <QVBoxLayout>

#include <NickelHook.h>

#include "fallback_strings.h"
#include "mod_switches.h"
#include "nickel_bridge.h"
#include "screen_logic.h"

static const char *const CONTENT_STYLE = "QLineEdit { font-size: 40px; border: 2px solid black; padding: 16px; }";
static const char *const NUMBER_KEYBOARD = "number";
static const char *const URL_KEYBOARD = "url";

static Qt::InputMethodHints hintsFor(const QString &keyboard)
{
    if (keyboard == QLatin1String(NUMBER_KEYBOARD)) {
        return Qt::ImhDigitsOnly;
    }
    return keyboard == QLatin1String(URL_KEYBOARD) ? Qt::ImhUrlCharactersOnly : Qt::ImhNone;
}

FieldEditor::FieldEditor() : QObject(NULL), hasFinished(false) {}

bool FieldEditor::isAvailable()
{
    return ModSwitches::usesNickelKeyboard() && NickelBridge::hasKeyboardDialog();
}

FieldEditor *FieldEditor::open(const QString &label, const QString &value, bool isSecret, const QString &keyboard,
                               QObject *owner)
{
    if (!isAvailable()) {
        return NULL;
    }
    FieldEditor *editor = new FieldEditor();
    if (!editor->show(label, value, isSecret, keyboard)) {
        delete editor;
        return NULL;
    }
    connect(owner, SIGNAL(destroyed()), editor, SLOT(onClose()));
    return editor;
}

bool FieldEditor::show(const QString &label, const QString &value, bool isSecret, const QString &keyboard)
{
    // The content becomes the dialog's child, so Nickel deletes it with the dialog.
    QFrame *content = new QFrame();
    content->setStyleSheet(QString::fromLatin1(CONTENT_STYLE));
    QVBoxLayout *layout = new QVBoxLayout(content);
    layout->setContentsMargins(ScreenLogic::GUTTER, ScreenLogic::RHYTHM, ScreenLogic::GUTTER, ScreenLogic::RHYTHM);
    lineEdit = new QLineEdit(content);
    // A secret arrives masked from appsctl: start empty instead of editing the dots.
    lineEdit->setText(isSecret ? QString() : value);
    lineEdit->setEchoMode(isSecret ? QLineEdit::Password : QLineEdit::Normal);
    lineEdit->setInputMethodHints(hintsFor(keyboard));
    layout->addWidget(lineEdit);
    layout->addStretch(1);

    dialog = NickelBridge::openKeyboardDialog(content, lineEdit, label,
                                              FallbackStrings::text(FallbackStrings::SCREEN_KEYBOARD_GO), this,
                                              SLOT(onCommit()), SLOT(onClose()));
    if (!dialog) {
        if (!content->parentWidget()) { // otherwise the popped dialog deletes it
            delete content;
        }
        return false;
    }
    connect(dialog, SIGNAL(destroyed()), this, SLOT(onDialogDestroyed()));
    return true;
}

void FieldEditor::onCommit()
{
    finish(true);
}

void FieldEditor::onClose()
{
    finish(false);
}

// Home, a tab switch or Nickel itself removed the dialog.
void FieldEditor::onDialogDestroyed()
{
    finish(false);
}

void FieldEditor::finish(bool isCommitted)
{
    if (hasFinished) {
        return;
    }
    hasFinished = true;
    QString value = lineEdit ? lineEdit->text() : QString();
    if (dialog) {
        disconnect(dialog, SIGNAL(destroyed()), this, SLOT(onDialogDestroyed()));
        NickelBridge::closeKeyboardDialog(dialog);
    }
    if (isCommitted) {
        emit committed(value);
    }
    emit finished();
    deleteLater();
}
