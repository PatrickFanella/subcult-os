# Mobile app links for identity email

Verification and recovery emails contain web links built from `PUBLIC_WEB_URL`:
`/verify-email?token=…` and `/recover-password?token=…`. When the native app is
installed and its domain association verifies, those two paths open the app.
Otherwise they open the existing web pages. No other path is claimed, so ticket,
consent and public event links stay in the browser.

The custom `subcultos://` scheme still works for development, but email never
uses it: a custom scheme cannot be verified and has no browser fallback.

## Configuration

| Where | Variable | Value |
|---|---|---|
| API | `MOBILE_APPLE_APP_IDS` | Comma-separated `TEAMID.tv.clpr.subcultos` values. Serves `/.well-known/apple-app-site-association`. |
| API | `MOBILE_ANDROID_PACKAGE` | `tv.clpr.subcultos`. Set together with the fingerprint list. |
| API | `MOBILE_ANDROID_CERT_SHA256` | Comma-separated SHA-256 signing-certificate fingerprints (`AA:BB:…`, 32 bytes). Serves `/.well-known/assetlinks.json`. |
| Mobile build | `SUBCULT_APP_LINK_HOST` | The hostname of `PUBLIC_WEB_URL`, for example `os.subcult.tv`. Adds iOS `associatedDomains` and an Android `autoVerify` intent filter. |

Each `/.well-known` document returns 404 until its variables are set. The web
nginx image proxies both paths to the API; any edge proxy in front of it must
pass them through over HTTPS without redirects.

- **Apple Team ID:** Apple Developer account, Membership details. Associated
  Domains needs a paid developer team; a free personal team cannot sign it.
- **Android fingerprints:** list every key that signs an installed build. For
  local builds, `keytool -list -v -keystore mobile/android/app/debug.keystore -alias androiddebugkey -storepass android -keypass android`.
  For EAS builds, `eas credentials`. With Play App Signing, add the app signing
  key from Play Console as well as the upload key.
- The host is compiled into the app. Changing `PUBLIC_WEB_URL` requires a new
  build with the matching `SUBCULT_APP_LINK_HOST`, or existing installs will
  send email links to the browser.

## Checking the association

```sh
curl -fsS https://HOST/.well-known/apple-app-site-association
curl -fsS https://HOST/.well-known/assetlinks.json
# Android, after installing the build:
adb shell pm verify-app-links --re-verify tv.clpr.subcultos
adb shell pm get-app-links tv.clpr.subcultos
# iOS reads through Apple's CDN, which can lag the origin:
curl -fsS https://app-site-association.cdn-apple.com/a/v1/HOST
```

Android reports `verified` for the host when the fingerprint and package match.
Before a real host serves the file, an Android tester can approve the host by
hand with `adb shell pm set-app-links-user-selection --user cur --package tv.clpr.subcultos true HOST`.
That is a development shortcut, not verification evidence. iOS has no
equivalent; it needs the real HTTPS host.

## IDENT-02 device qualification

This is the remaining evidence for the real-device line of IDENT-02 (#6). Use a
synthetic `@example.test` account and a disposable stack; mail delivery stays
disabled and links are read from the held outbox.

1. Start an isolated stack with a `subcult_qa_*` database, the API published on
   the LAN, and `PUBLIC_WEB_URL=https://HOST`, following the pattern in
   [the operations-panel rehearsal](../qa/operations-panels-2026-09-23.md).
2. Build a development client: `cd mobile && SUBCULT_APP_LINK_HOST=HOST EXPO_PUBLIC_API_URL=http://KVANT_LAN_IP:API_PORT npx expo run:android`
   (or `run:ios`). Expo Go cannot test app links.
3. Print or open a link with
   `API_URL=http://127.0.0.1:API_PORT QA_DATABASE_URL=… QA_DISPOSABLE_DATABASE=1 scripts/device-link.sh verify|recover RECIPIENT@example.test [--adb]`.
   On iOS, send the printed link to the device and tap it from Notes or Mail.

| Check | Expected result |
|---|---|
| Association | Both documents served; Android `get-app-links` shows `verified` (or record that the manual approval shortcut was used). |
| Verify link | Tapping the https link opens the app, verifies, and lands signed in on Staff. Opening it again is rejected. |
| Secure storage | Settings shows access and refresh tokens present; they are absent from plain app storage on a debug device. |
| Restart | Force-quit and relaunch: still signed in without a prompt. |
| Refresh | After more than 15 minutes idle, a workspace screen loads without a sign-in prompt. |
| Recovery link | Request recovery from sign in; the https link opens the app's new-password screen; saving returns to sign in with the recovered notice. |
| Recovery revokes | A browser session for the same account is signed out after recovery; the old password fails. |
| Revoke-all | Logout-all from the browser; the phone's next request fails refresh and returns to sign in. |
| Logout | Log out on the phone, relaunch: still signed out; the old refresh token is rejected. |
| Fallback | With the app uninstalled, the same kind of link opens the web page and completes there. |

Record on #6: device model and OS version, app build and source commit, API
commit, the host and how the association was verified, each row's result, and
any limit such as an untested platform.
