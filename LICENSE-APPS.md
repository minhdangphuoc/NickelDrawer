# App developer exception

NickelDrawer is licensed under the [PolyForm Noncommercial License 1.0.0](LICENSE). This file is an additional permission from the NickelDrawer authors (the licensor). It widens what you may do; it never narrows the main license. It does not apply to the apps in this repository's `apps/` folder, which stay under the main license.

## 1. Your app is yours

An app that runs inside NickelDrawer only by way of its public interfaces is not a change or new work based on NickelDrawer. The public interfaces are:

- the app folder format (`app.json`, `ui.json`, `locales/`, icons), as described in `docs/spec/`;
- the drawer protocol (the JSON messages between an app and the host);
- the command-line interface of `appsctl` and the other host tools;
- the release and update formats (`update.json`, signatures, `apps.json`).

You choose your app's license, including a proprietary or commercial one. The NickelDrawer license does not apply to your app, and you need not include NickelDrawer's license or notices with it, as long as you do not copy NickelDrawer's own code into it (see section 3).

## 2. Developer tools may be used for any app

You may use NickelDrawer's developer tools (`appsctl new`, `validate`, `app keygen`, `app pack`, `app sign`, `app verify`, `ndpreview`, and the documented make targets) to create, test, package, sign and publish apps under any license, for any purpose, including commercial ones. This permission covers using the tools, not redistributing or changing them; those stay under the main license.

## 3. The app template and examples

The files that `appsctl new` writes into a new app (from `templates/app/`) and the example apps in `examples/` are also licensed to you under the [Zero-Clause BSD license (0BSD)](https://opensource.org/license/0bsd):

> Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby granted.
>
> THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

So code you start from the template or copy from an example can go into an app under any license. Other code in this repository (for example `internal/`, `mod/`, `lib/copperui/` and the apps in `apps/`) stays under the main license: copying it into your app makes that part subject to the PolyForm Noncommercial terms.

## 4. What this does not cover

- NickelDrawer itself, its host tools and the apps in `apps/` stay under the main license. Selling or commercially distributing NickelDrawer is not permitted by this exception.
- Names: "NickelDrawer" and its icon are not licensed for your app's name or branding. Saying your app "works with NickelDrawer" is fine.
- Third-party code keeps its own license ([third_party/README.md](third_party/README.md)).
- Kobo and Nickel are trademarks of their owners; nothing here grants rights in them.

This text explains the licensor's permission in plain words; it is not legal advice.
