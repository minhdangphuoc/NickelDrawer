#ifndef NICKELDRAWER_FIELD_EDITOR_H
#define NICKELDRAWER_FIELD_EDITOR_H

#include <QObject>
#include <QPointer>
#include <QString>

class QLineEdit;
class QWidget;

// Edits one `field` value (docs/spec/ui-kit.md §3.10) with Nickel's keyboard:
// an N3Dialog titled with the field label holds a line edit; the keyboard's Go
// key commits, the dialog's close button cancels. Without the keyboard API
// (symbols missing, no-keyboard flag) it is not created: the caller shows the
// "edit on your phone" notice instead.
//
// Not a child of any Nickel widget (re/notes/protoA-crash.md): it deletes
// itself when the edit ends, and only its content widget lives in the dialog.
class FieldEditor : public QObject {
    Q_OBJECT
public:
    static bool isAvailable();
    // NULL when the dialog could not be opened (logged). The dialog closes
    // (cancelled) when owner goes away, e.g. Home closes the drawer under it.
    static FieldEditor *open(const QString &label, const QString &value, bool isSecret, const QString &keyboard,
                             QObject *owner);

signals:
    void committed(const QString &value);
    // Committed or cancelled; emitted once, then the editor deletes itself.
    void finished();

private slots:
    void onCommit();
    void onClose();
    void onDialogDestroyed();

private:
    FieldEditor();
    bool show(const QString &label, const QString &value, bool isSecret, const QString &keyboard);
    void finish(bool isCommitted);

    QPointer<QWidget> dialog;
    QPointer<QLineEdit> lineEdit;
    bool hasFinished;
};

#endif
