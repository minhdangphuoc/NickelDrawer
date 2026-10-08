# Security

## What NickelDrawer is

NickelDrawer adds an app menu to a Kobo e-reader. Apps are folders with scripts. The scripts run when you tap an item.

## Threat model

- **Apps run as root.** Nickel starts them with full rights on the device. An app can read and change anything on the Kobo, including your library and Wi-Fi settings.
- **Only install apps you trust.** Read the scripts first. An app from a stranger is the same as running a stranger's program as root.
- **Releases are signed.** Release files are signed with ed25519. Check the signature before you install. The private key stays on the maintainer's machine and is never in the repository or in CI.
- **Tokens stay on the device.** Secrets such as GitHub tokens live in the app's own `data/` folder, mode 600. They are never printed or logged. Do not paste them in issues.
- **No telemetry.** NickelDrawer sends nothing about you or your device. Apps only contact the services they are built for (for example GitHub for ghsync). Each app's README says which.
- **Anyone with USB access can change files.** The Kobo has no login. If someone holds your device, they can edit apps. Keep the device with you.

## Out of scope

- Bugs in Kobo's own firmware.
- Problems that need physical access and a modified app folder.
- Apps you wrote or installed yourself that misbehave.

## Report a vulnerability

Please report privately. Do not open a public issue.

1. Go to the repository's **Security** tab.
2. Choose **Report a vulnerability** (GitHub private vulnerability reporting).
3. Describe the problem, the steps to repeat it, and the firmware and NickelDrawer version.

Remove tokens and personal data from anything you attach.

We aim to reply within 7 days and to fix confirmed problems as soon as we can. We will credit you in the release notes if you want.

## Supported versions

Until v1.0, only the latest release gets fixes.
