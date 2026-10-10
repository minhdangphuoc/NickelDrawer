# Adding a language

English is the source language. Translations are flat JSON files, one per language, so no code changes are needed.

Language codes follow the Kobo setting (`CurrentLocale` in `Kobo eReader.conf`): `vi`, `fr`, `pt_BR`. A device set to `pt_BR` uses `pt_BR.json`, then `pt.json`, then English; a missing key falls back to English on its own.

1. Pick the language code, for example `fr`.
2. Host strings (refresh and error items): copy `host/locales/en.json` to `host/locales/fr.json`.
3. Host screens: the settings and About screens have their own folder. Copy `host/screens/settings/locales/en.json` to `host/screens/settings/locales/fr.json`.
4. For each app in `apps/<id>/`: copy `locales/en.json` (if it exists) to `locales/fr.json`, and add the manifest strings you want translated: `"name"`, `"description"`, `"actions.<action-id>.label"` and, for actions with a `confirm` question, `"actions.<action-id>.confirm"` (the English text is in `app.json`).
5. Translate the values only. Never change a key.
6. Keep every `{placeholder}` exactly as written (you may move it within the sentence). Keep `\n` for line breaks.
7. Keep it short: the screen is small. Labels are at most 32 characters (names 24) and must not contain `:`.
8. Check your work: `appsctl validate --apps-dir apps` (run `make build` first and use `build/appsctl-host`). It reports unknown keys, mismatched placeholders and labels that break the rules.
9. Try it: `build/appsctl-host drawer tweaks --locale fr --koboapps-dir host --conf /tmp/x.conf` and read the labels.

Not covered by these files: the few fixed strings of the native mod itself (More-tab row, "Cancel", "Remove", the Wi-Fi and error messages, Info dialog labels) are compiled in, English and Vietnamese only (`mod/drawer/fallback_strings.cc`). A new language needs a small code change there plus a rebuilt mod; everything else, including all app, screen and Tweaks text, comes from the JSON files above.

After installing, change the Kobo's language; the drawer picks it up the next time it opens.
