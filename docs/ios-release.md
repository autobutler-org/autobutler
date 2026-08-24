# iOS App Store Release

How to build and publish the Quark iOS app to the App Store from a local machine.

For *development* setup (running on your own device), see [iOS Mobile Dev Setup](./ios-dev/README.md).
This document is only about shipping to the App Store.

## Identifiers

| Thing | Value |
|---|---|
| Apple Team | `4NK7MWUA57` (Autobutler LLC) |
| Bundle ID | `org.autobutler.quark` |
| App Store listing name | Quark by AutoButler |
| Home screen name (`CFBundleDisplayName`) | Quark |

The listing name and the home screen name are deliberately different. The App Store
requires a globally unique listing name; the home screen truncates at ~12 characters.

## Prerequisites

### 1. Apple Distribution certificate

You need an **Apple Distribution** identity in your keychain — not Apple Development.
Xcode > Settings > Accounts > sign in to the team above > Manage Certificates > `+` >
Apple Distribution.

Verify:

```bash
security find-identity -v -p codesigning | grep "Apple Distribution"
```

### 2. App Store Connect API key

Create a **Team Key** with the **App Manager** role at
<https://appstoreconnect.apple.com/access/integrations/api>.

Apple serves the `.p8` file exactly once. Save it where `altool` looks for it:

```bash
mkdir -p ~/.appstoreconnect/private_keys
mv ~/Downloads/AuthKey_XXXXXXXXXX.p8 ~/.appstoreconnect/private_keys/
```

### 3. Secrets in `.env`

`.env` is gitignored and auto-loaded by the Makefile. See `.env.example`.

```bash
APP_STORE_CONNECT_KEY_ID=XXXXXXXXXX                          # the KEY_ID in the .p8 filename
APP_STORE_CONNECT_ISSUER_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx # top of the API keys page
```

The `.p8` file itself is **not** referenced by path — `altool` finds it by key ID in
`~/.appstoreconnect/private_keys/`. Never commit it.

### Check everything at once

```bash
make check/frontend/ios/release
```

## Building and uploading

```bash
make build/frontend/ios/ipa     # clean, archive, export, verify
make publish/frontend/ios       # upload to App Store Connect
```

`build/frontend/ios/ipa` wipes DerivedData and runs `flutter clean` first, then verifies
the exported IPA is a genuine release build before handing it to you. Both steps exist
because of a real incident — see [Debug build shipped to
TestFlight](#debug-build-shipped-to-testflight). Set `IOS_SKIP_CLEAN=1` to skip the clean
when iterating on signing or export settings; the verification always runs.

Check an IPA by hand:

```bash
make check/frontend/ios/ipa                 # newest IPA in build/ios/ipa/
make check/frontend/ios/ipa IOS_IPA=path    # a specific one
```

The build appears in TestFlight after processing, usually 5–30 minutes.

Override the build number for a one-off:

```bash
IOS_BUILD_NUMBER=7 make build/frontend/ios/ipa
```

## Versioning rules

Both numbers come from `version:` in `pubspec.yaml` (`<version>+<build>`).

- **Build numbers are consumed permanently.** Once App Store Connect has seen a
  `version (build)` pair, it can never be uploaded again — expiring or deleting the
  build does not free it. Always bump.
- **Released version numbers must increase.** You can upload any version to TestFlight,
  but once a version is *released* on the App Store, every later release must be higher.
- **Uploads cannot be deleted.** TestFlight builds can be *expired*; unreleased App Store
  versions can be removed from the sidebar. Neither frees the build number.

## What is already configured

Set in `ios/Runner/Info.plist`:

- `ITSAppUsesNonExemptEncryption = false` — skips the export compliance prompt on every
  upload. Correct because Quark only uses TLS and Keychain, both exempt.
- `NSAllowsLocalNetworking` (not `NSAllowsArbitraryLoads`) — the app talks to a
  self-hosted server over plain HTTP on the LAN, and nothing else.
- `NSLocalNetworkUsageDescription` — required on iOS 14+ to reach `*.local` hostnames
  and LAN IPs. Without it, connections silently fail on device.

Set in `ios/Runner.xcodeproj/project.pbxproj`:

- No hardcoded `CODE_SIGN_IDENTITY`. Automatic signing picks Apple Development for local
  builds and Apple Distribution for archives. Do not pin it back.
- `IPHONEOS_DEPLOYMENT_TARGET = 15.0`. Apple rejects uploads below 15.0 from Spring 2027.
  This costs no devices — iOS 15 runs on the same hardware as iOS 13 (iPhone 6s onward).

`ios/ExportOptions.plist` drives the export: `app-store-connect` method, automatic
signing, symbols uploaded.

## Troubleshooting

### `accessing build database ... disk I/O error`

Usually accompanied by `mkdtemp ... No such file or directory` and asset catalog errors
like `Each TDDistiller instance can be distilled only one time!`.

Two build services are fighting over one DerivedData tree, or the cache is corrupt.
**Quit Xcode**, then:

```bash
rm -rf ~/Library/Developer/Xcode/DerivedData/Runner-* \
       ~/Library/Developer/Xcode/DerivedData/Pods-*
flutter clean && flutter pub get && (cd ios && pod install)
```

Keep Xcode closed during CLI archives.

### `MinimumOSVersion too low (90068)`

A warning, not a failure. Fixed already by the 15.0 deployment target — if it reappears,
check `IPHONEOS_DEPLOYMENT_TARGET` in the pbxproj and `platform :ios` in `ios/Podfile`
are both 15.0.

### Debug build shipped to TestFlight

Symptom: the app installs from TestFlight, then shows a black screen reading *"In iOS 14+,
debug mode Flutter apps can only be launched from Flutter tooling, IDEs with Flutter
plugins or from Xcode."*

Cause: `flutter build ipa --release` can silently produce a debug artifact. Flutter's
incremental build system skips `release_unpack_ios` and `aot_assembly_release` and reuses
a debug-mode `App.framework` and `Flutter.framework` left behind in DerivedData's
`Release-iphoneos` products directory by an earlier `flutter run`. The build reports
success, `xcodebuild` really does use `-configuration Release`, and the IPA uploads and
installs normally. It just cannot launch.

How to tell them apart:

| | debug | release |
|---|---|---|
| `flutter_assets/kernel_blob.bin` | present | absent |
| `App.framework/App` | ~35 KB stub | tens of MB (AOT) |
| `Flutter.framework/Flutter` | ~39 MB (JIT) | ~9 MB |

`make check/frontend/ios/ipa` checks this, and `make build/frontend/ios/ipa` runs it
automatically. Never upload an IPA that has not been through it.

Recovery is a clean rebuild — which the build target now does unconditionally. Note the
bad build number is gone forever; bump and re-upload.

### `The bundle version must be higher than the previously uploaded version`

The build number was already used. Bump `version:` in `pubspec.yaml` or pass
`IOS_BUILD_NUMBER`.

## Before submitting for review

Getting a build into TestFlight is not the same as passing review. Still outstanding:

- [ ] **`PrivacyInfo.xcprivacy`** — a privacy manifest is required. Declare tracking
      status and any required-reason API use.
- [ ] **In-app account deletion** — Guideline 5.1.1(v) requires it for any app that
      supports account creation. Quark has a login flow and no deletion path.
- [ ] **A reachable demo server** — the highest rejection risk. Quark is useless without
      a backend, and a reviewer has none. Guideline 2.1 rejections follow. Stand up a
      demo instance and put the URL and credentials in App Review notes.
- [ ] App Privacy questionnaire, age rating, category, screenshots (6.9" iPhone, and 13"
      iPad while `TARGETED_DEVICE_FAMILY` is `1,2`).
- [ ] **Privacy policy URL** — required for every app. Quark does not have one yet.
