# Adding a language

English is the source language. Translations are flat JSON files, one per language, so no code changes are needed.

Language codes follow the Kobo setting (`CurrentLocale` in `Kobo eReader.conf`): `vi`, `fr`, `pt_BR`. A device set to `pt_BR` uses `pt_BR.json`, then `pt.json`, then English; a missing key falls back to English on its own.

1. Pick the language code, for example `fr`.
2. Host strings (refresh and error items): copy `host/locales/en.json` to `host/locales/fr.json`.
3. For each app in `apps/<id>/`: copy `locales/en.json` (if it exists) to `locales/fr.json`, and add the manifest strings you want translated: `"name"`, `"description"`, `"actions.<action-id>.label"` (the English text is in `app.json`).
4. Translate the values only. Never change a key.
5. Keep every `{placeholder}` exactly as written (you may move it within the sentence). Keep `\n` for line breaks.
6. Keep it short: the screen is small. Labels are at most 32 characters (names 24) and must not contain `:`.
7. Check your work: `appsctl validate --apps-dir apps` (run `make build` first and use `build/appsctl-host`). It reports unknown keys, mismatched placeholders and labels that break the rules.
8. Try it: `build/appsctl-host gen-menu --locale fr --apps-dir apps --koboapps-dir host --out /tmp/menu --errors-file /tmp/errors.txt` and read `/tmp/menu`.

After installing, change the Kobo's language, then tap the refresh item once so the menu is regenerated in the new language.
