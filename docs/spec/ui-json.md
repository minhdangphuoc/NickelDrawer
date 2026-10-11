# ui.json v1 (normative schema)

This file is the exact contract for an app's `ui.json`. The picture of each component, the sizes and the screen patterns are in [ui-kit.md](ui-kit.md); the app-to-host protocol is [drawer-protocol.md](drawer-protocol.md) (action scripts, `state`, JSON signals; the JSON-lines backend of plan §7.7 was dropped). Where this file and `ui-kit.md` differ on a key name, a limit or a rule, this file wins.

Three things are kept equal by tests, so they cannot drift:

- the Go registry in `internal/uikit/spec.go` (what `appsctl validate` enforces),
- the JSON Schema `spec/ui.schema.json` (generated from that registry; `make schema` rewrites it),
- this document (a test checks that every component below is named here).

## 1. Files

| File | Purpose |
|---|---|
| `ui.json` | Required for a native app. `kit`, `start`, `vars`, `screens`. |
| `screens/<id>.json` | Optional. One screen object per file, the file name is the screen id. A screen defined in both places is an error. |
| `locales/en.json`, `locales/<lang>.json` | Texts referenced as `"@ui.key"` (section 4). |

Keys starting with `_` at the top level are comments and are ignored (for example `_preview`). Every other unknown key is an error.

## 2. Top level

```json
{ "kit": 1, "start": "main", "vars": { "status": "", "items": [] }, "screens": { "main": { "title": "..." } } }
```

| Key | Type | Rule |
|---|---|---|
| `kit` | number | Must be `1`. |
| `start` | string | Must be `"main"`: every app opens on its `main` screen (section 3.1). |
| `vars` | object | Default value of every variable a screen binds to. A variable that is not declared here cannot be used. Names: letters, digits, `_`, not starting with a digit. |
| `screens` | object | Screen id to screen. Use `{}` when every screen is a file in `screens/`. At least one screen must exist in total. Ids match `^[a-z0-9][a-z0-9_-]{0,31}$`. |

## 3. Screen

| Key | Required | Type | Notes |
|---|---|---|---|
| `title` | yes | text, 40 chars | Title in Nickel's page header (left, next to the back arrow). |
| `status` | no | text, 40 chars | Plain text at the right of the header. `badge` is accepted as the old name. |
| `action` | no | object | Header action (text link): `label` (text, 24 chars, no `:`) and exactly one of `push` (screen id) or `action` (app.json action id). |
| `actions` | no | array, 1 to 2 | Header icon actions instead of `action` (a screen has one or the other, else `ui.needs_exactly_one`): each `{ icon, label, push \| action }`. `icon` is `import`, `settings`, `refresh` or `add`; `label` (text, 24 chars, no `:`) is the accessible name, not drawn. Exactly one of `push` / `action`. |
| `body` | no | components | Scrolls page-wise. |
| `footer` | no | components | Stays at the bottom: put the primary button here. |
| `dialogs` | no | map id to dialog | `title` (40), `text` (200), `ok` (24, no `:`), optional `cancel` (24) and `action` (runs when `ok` is tapped). The backend opens one with `{"dialog":"id"}`. |
| `toasts` | no | map id to text (60) | Texts the backend may show with `{"toast":"id"}`. |

The header is part of the screen, not a component: back arrow, `title`, `status`, `action` or `actions`.

Rule: **at most one primary button per screen**, counting body, footer, the button of an `empty` component, the button of a gallery's `empty` object and the `primary` of a `viewer`. A button inside `samples` is a picture and does not count. A screen whose main action changes with its state (LocalSend: Start receiving / Stop; GitHub Sync: Pair with phone / Sync now) may declare several primaries when **each has its own `show` flag** and no two share one flag (`"$running"` and `"!$running"`): the host leaves the hidden ones out, so one primary is visible at a time. A primary with `enabled` bound to `$x` next to a button enabled by `!$x` is a warning (`ui.primary_disabled_while_other_enabled`, design-system A8): promote the possible action instead of disabling the primary.

### 3.1 Navigation

The view flow is not declared separately: it is the graph of the `push` links of all screens: the header `action`, the header `actions[]`, a `row` or `button` with `push`, a gallery's `on_tap.push` and its `on_long_press[].push` (`ui.json` and `screens/*.json`). `appsctl validate` checks it:

| Rule | Code |
|---|---|
| A screen with the id `main` exists; the app opens on it (`start` is `"main"`). | `ui.flow_no_main`, `ui.flow_start_not_main` |
| Every screen is reachable from `main` through pushes (no orphan screens). | `ui.flow_unreachable` |
| At most 2 screens below `main` (main → screen → detail). | `ui.flow_too_deep` |
| No cycles, a screen pushing itself included. | `ui.flow_cycle` |
| A push names an existing screen. | `ui.unknown_screen` |

On the Kobo, back ("←") returns one level, like a back stack: a pushed screen → the screen that pushed it (reloaded, at the page it was on) → `main` → the drawer (same list/grid mode and page) → More. A tab tap or Home clears the whole stack. The drawer also refuses at run time to push deeper than 2 below `main` and logs it.

## 4. Texts

Wherever a text is allowed, one of:

1. a plain string: `"Stop"`,
2. a reference `"@ui.stop_button"` into the app's locale files; the key must exist in `locales/en.json`; a missing key in another locale is a warning and shows English,
3. a map `{ "en": "Stop", "vi": "Dừng" }`; `en` is required.

A plain string that starts with `@` is always a reference (there is no escape; reword the text). Locale keys use the prefix `ui.` (lowercase letters, digits, `_` and `.`); `msg.*` keys belong to scripts. The text may contain bindings (section 5).

Limits, counted in characters of every language, apply to texts without bindings (the length of a filled-in template is unknown): they are listed per property in section 7. A translation more than 150 % of the English length (English of 10 characters or more) is a warning. Labels of buttons, menu items, header actions and dialog buttons must not contain `:`, because menus use it as a separator.

## 5. Bindings

| Form | Meaning |
|---|---|
| `$name` | The value of variable `name`. `$device.name` reads a field of an object. |
| `Received $count books` | Template: every `$ident` is replaced. |
| `$$` | A literal dollar sign. A `$` followed by anything else is an error (`costs $5` must be `costs $$5`). |
| `$list[0]` | Not supported in kit 1 (error). |
| `"enabled": "$running"`, `"!$running"` | Flags: `true`, `false`, `$name` or `!$name`. A missing `enabled` means enabled. |

The first name of every binding must be declared in `vars`. `bind` props (toggle, radio, field) take a plain variable name without `$`. `list` takes exactly one binding to a variable whose default is an array.

## 6. Components

A component is one object with one component key; its other keys are properties of that component. `text`, `button` and `title` are also property names of other components (`empty`, `notice`); the component key is the one that is not a property of another key present in the same object.

| Component | Value | Properties | Event (an action script run by `drawer event`) |
|---|---|---|---|
| `text` | text (400) | `style`: `body` / `caption` / `heading`; `align`: `start` / `center` | none |
| `section` | text (40) | none | none |
| `kv` | array of `[key, value]`, 1 to 12; key text (24), value text with bindings (60) | none | none |
| `row` | label text (60) | `detail` (60), `action` (action id; makes the row tappable) or `push` (screen id), `confirm` (the object of a button's `confirm`, see Details; asks before the `action`; needs `action`), `show` (flag; hidden while false) | `action` |
| `toggle` | label text (60) | `bind` (required), `enabled`, `detail` (hint line inside the row, 60) | `toggle` |
| `radio` | label text (60) | `bind` and `options` (both required), `enabled`. `options` is 1 to 8 entries `[value, label]` or `[value, label, sub-line]`; values are ids (`^[a-z0-9][a-z0-9_-]{0,31}$`), unique; label text (40), sub-line text (60, makes the row 128 px) | `input` |
| `field` | label text (40) | `bind` (required), `keyboard`: `text` / `number` / `url`, `placeholder` (40), `enabled` (flag; a disabled field is dimmed and has no target), `value` (text, 40: the shown text of a field inside `samples` only; a live field shows its bound variable), `secret` (bool; a warning when `bind` looks like a token or key: use `pair`) | `input` |
| `bigcode` | binding or literal (16) | `digits`: 3 to 12 | none |
| `qr` | binding or literal (200) | `caption` (60) | none |
| `pair` | object `{url, code, title, hint, expires}`; `url` (120) and `code` (16) required | none | none |
| `steps` | object `{label, done, total, detail}`; `label` (40), `done` and `total` (16) required. A steps block whose `label` resolves to empty is hidden (no job is running); with `done` and `total` both empty the headline is the label alone (a job without a count, such as "Checking for updates…") | none | none |
| `notice` | text (200); a notice whose text resolves to empty is hidden, so bind a variable to show it conditionally | `title` (40) | none |
| `button` | label text (32, no `:`) | exactly one of `action` (action id) or `push` (screen id; no `confirm`), `style`: `primary` / `secondary` (default `secondary`), `enabled`, `show` (flag; the button is left out, and the page re-packs, while it is false), `confirm`, `async` | `action` |
| `list` | one binding to an array variable | `title` (40), `empty_text` (60), `style` (`plain` default, `checklist`), `max` (1 to 200, default 50), `tap` (action id), `menu` (1 to 6 items; `id` is an action id) | `tap`, `menu` |
| `gallery` | one binding to an array variable of pictures | `columns`, `rows` (1 to 6, default 4 and 3), `on_tap` (`{push \| action}`, exactly one), `on_long_press` (1 to 6 items `{label (32, no `:`), action \| push \| info: true, confirm}`, exactly one of the three), `pending` (binding or literal, 8: dashed placeholder tiles), `empty` (`{title (40, required), text (120), icon: picture, button (32) with action}`; its button is the primary). At most one per screen | `on_tap`, `on_long_press` (the tile id is the value) |
| `viewer` | one binding to the same array | `selected` (required, a declared variable that receives the screen value), `current_status` (40), `primary` (required `{label (32), label_current (32), action}`, counts as the primary), `secondary` (`{label (32), action, confirm}`). Must be the only body component and the screen has no footer (`ui.viewer_not_alone`). At most one per screen | `primary`, `secondary` (the shown id is the value) |
| `empty` | heading text (40) | `text` (120), and together `button` (label, 32) with `action`; the button is a primary button | `action` |
| `samples` | array of 1 to 8 cells (see below) | `columns`: 1 to 3 (default 1; more than 1 only for button cells) | none |

Details:

- `button.confirm` is `{ "text": ..., "ok": ..., "cancel": ..., "title": ... }`. `text` (160) and `ok` (24, no `:`) are required. The host shows the confirm dialog first and sends the event only on OK.
- `list` items are strings or `[primary, secondary]` arrays; a secondary text of exactly `✓` is drawn as the check icon. The list keeps the last `max` items. `menu` items are `{ "id", "label", "confirm" }`: `id` names an app.json action (checked); a long-press on a row opens the menu and runs that action as `drawer event <app> <id> --list <bind> --index <n> --value <row text>` (`confirm: true` asks first). See drawer-protocol.md.
- `list` with `"style": "checklist"` draws a 64 px box at the left of every row's text: Tabler `square` for an open item, `square-check` for a done item. An item is done when it is `[primary, "✓"]`; a plain string or an empty secondary text is open. In a checklist the `✓` is not drawn at the right; any other secondary text stays a caption. Tap and the long-press menu work as in a plain list, and the pressed row is inverted with the box in white. Default items must be strings or `[primary, secondary]` arrays (`ui.checklist_items`). Needs NickelDrawer 1.0.0-beta.8; an app that uses it sets `min_host` (developer-guide.md).
- `list.tap` names an app.json action. A tap runs that action with the row's primary text in `NICKELDRAWER_VALUE`.
- `gallery` and `viewer` items come from the `state` action, not from `ui.json`: `{ id, thumb, image, label, info, badge }` (`id` matches `^[A-Za-z0-9_-]{1,32}$`, `badge` is `current`). `thumb` and `image` are absolute `.png` paths below the app's `data/`; the drawer drops any other path. Looks and behaviour: [ui-kit.md](ui-kit.md) sections 3.19 and 3.20, contract: [wallpaper-gallery.md](wallpaper-gallery.md).
- A `text` whose text resolves to empty is left out, like a `notice`, so `"$summary"` hides when the variable is empty. A `kv` row whose key and value both resolve to empty is left out, and a `kv` with no rows left is hidden, so `["$label", "$value"]` shows a row only while its variables are set.
- `row` is 128 px high (above the 120 px tap target); `kv` and `list` rows are 104 px; see ui-kit.md section 2.
- `qr` shows the QR code of its value (a URL). `pair` is the phone-pairing block of ui-kit.md section 3.11.
- `section` is the small-caps label of a group of rows.
- `samples` is the states matrix of the UI Kit app (uikit-plan.md section 5.1): a picture of controls, not controls. A cell is one sampleable component (`button`, `toggle`, `radio` with exactly one option, `row`, `field`, `kv` with one row, `notice`, `steps`, `text`) plus `look` (`normal` default, `pressed` = the cell inverted as a finger-down, `selected` = the on state, only for `toggle` and `radio`, `disabled` = the `ink.dim` look) and `caption` (text, 24 chars, caption style, centred under the cell). The inner component needs none of its usual required props (`action`, `bind`, `options` may be left out), takes no `$binding` and no `push` or `bind` (`ui.bad_binding`, `ui.sample_not_static`), has no target and sends no event. Cells that cannot be sampled (`list`, `gallery`, `viewer`, `pair`, `qr`, `bigcode`, `empty`, `section`, `samples`) give `ui.sample_not_static`. Cell width is (1120 - (columns - 1) x 24) / columns; a button cell is one centred line, 120 px, elided. One `samples` is one block and never splits across pages.
- `toast` and `dialog` are not components: they are declared per screen (section 3).

## 7. Error codes

`appsctl validate` prints every problem as `<path>: [<code>] <message>`. The code is stable; the message says what to change.

| Code | Meaning |
|---|---|
| `ui.not_json`, `ui.wrong_type` | Syntax or JSON type is wrong. |
| `ui.bad_kit` | `kit` is not 1. |
| `ui.unknown_prop` | A key that is not allowed here. |
| `ui.unknown_component` | No known component key in the object. |
| `ui.ambiguous_component` | Two component keys in one object. |
| `ui.missing_prop`, `ui.needs_exactly_one`, `ui.needs_together` | A required property, or a required combination, is missing. |
| `ui.bad_enum`, `ui.out_of_range`, `ui.too_many_options` | A value outside its allowed set or range. |
| `ui.bad_binding`, `ui.unknown_var`, `ui.list_not_array` | Binding syntax, an undeclared variable, a list over a non-array. |
| `ui.checklist_items` | A checklist whose default items are not strings or `[primary, secondary]` arrays, so they cannot carry a done mark. |
| `ui.unknown_action`, `ui.unknown_screen` | An id that does not exist in app.json or in the screens. |
| `ui.bad_id`, `ui.duplicate` | A malformed or repeated id, or a second `gallery` / `viewer` on a screen. |
| `ui.multiple_primary` | More than one primary button on a screen (unless every primary has its own `show` flag). |
| `ui.sample_not_static` | A `samples` cell holds a component that cannot be sampled, or uses `push` / `bind`. |
| `ui.viewer_not_alone` | A `viewer` shares its screen with another body component or a footer. |
| `ui.missing_english`, `ui.bad_locale_key`, `ui.missing_locale_key`, `ui.text_empty` | Text problems. |
| `ui.text_too_long`, `ui.text_has_colon` | E-ink length limit, or `:` in a label. |
| `ui.flow_no_main`, `ui.flow_start_not_main`, `ui.flow_unreachable`, `ui.flow_too_deep`, `ui.flow_cycle` | Navigation graph rules (section 3.1). |

Warnings: `ui.translation_longer`, `ui.locale_key_untranslated`, `ui.secret_field`, and the content-style warnings of design-system section 5, which never stop an app from loading:

| Code | Meaning |
|---|---|
| `ui.uppercase_text` | A text (or any `locales/*.json` value) is typed in capitals (words of 3+ letters, `SỰ CỐ`), except the acronyms NFC, QR, URL, USB, PIN, IP and a few more. Write sentence case: section labels are uppercased by the renderer. |
| `ui.status_too_wide` | A header `status` text, or a locale key named like a status (`ui.status*`, `msg.status_*`, `msg.state_status_*`, `ui.*_status`), is estimated wider than 340 px at 32 px in some language (`{count}` counts as `99`). |
| `ui.caption_period` | A one-sentence caption (`text` in caption style, `toggle.detail`, radio sub-line, `qr.caption`, `empty.text`, `list.empty_text`, sample captions) ends with a full stop. |
| `ui.section_over_paragraph`, `ui.section_empty` | A `section` heads a single `text` or `notice`, or nothing at all. |
| `ui.primary_disabled_while_other_enabled` | The A8 rule above. |

## 8. Decisions where ui-kit.md was open

- Header: screen-level keys, not a component. `badge` stays an alias of `status`.
- QR: `qr` shows a code for one value; `pair` is the full pairing block (the plan's `QR` and `pairing` in one place).
- `dialog`/`confirm` and `toast`: screen-level `dialogs` and `toasts` maps, plus the `confirm` property of a button.
- Locale keys for screens: prefix `ui.`, referenced as `"@ui.key"`; `appsctl validate` accepts them in `locales/*.json` (like `msg.*`).
- Text limits are per property (section 6); they are the e-ink budget, not a guess about Nickel's fonts.
- Variables must be declared in `vars`; there is no implicit variable.
