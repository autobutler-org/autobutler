# Android Play Store Release

How to build and publish the Quark Android app to Google Play from a local machine.

For *development* setup (running on a device or emulator), see
[Dev Onboarding](./dev-onboarding.md) and `make emulate/android`. This document is only
about shipping to Play.

This pipeline produces two workflow artifacts: an AAB for the Play Console, and an APK for
local testing. Neither is published to a GitHub Release. Direct distribution — Obtainium,
F-Droid, published checksums — is tracked separately, and the APK it needs is not the one
built here; see [The APK built here is not distributable](#the-apk-built-here-is-not-distributable).

## Identifiers

| Thing | Value |
|---|---|
| `applicationId` | `org.autobutler.quark` |
| Gradle `namespace` | `org.autobutler` |
| Play listing name | Quark by AutoButler |
| Home screen name | Quark |
| Default track | `internal` |

`applicationId` matches the iOS bundle ID. `namespace` is the Kotlin/R-class package and
stays `org.autobutler`; the two are independent, and the sources do not move.

**`applicationId` is permanent.** Play identifies an app by it forever. Changing it after
the first upload means a new listing with no installs, no reviews and no upgrade path for
existing users.

## Signing keys

Android will not install an update signed by a different key than the installed app, so
signing outlives every other decision here. Two different keys are involved, and confusing
them is the most expensive mistake available in this document.

| Key | Who holds it | What it signs |
|---|---|---|
| **Upload key** | us — `quark-release.jks` | the AAB we hand to Play, and nothing users ever see |
| **App signing key** | Google | what Play actually installs on devices |

Play verifies our upload key, strips that signature, and re-signs with the app signing key
it holds. Every AAB must be signed with the upload key or the upload is rejected outright.

### Create the upload keystore

```bash
keytool -genkeypair \
  -keystore quark-release.jks \
  -alias quark-upload \
  -keyalg RSA -keysize 4096 -validity 10000 \
  -dname "CN=Quark, O=Autobutler LLC, C=US"
```

`keytool` defaults to PKCS12, which holds **one** password — the key password and the
store password must be identical. Set them the same; the CI script fails explicitly if
they differ, because Gradle will not tolerate the mismatch even though `keytool` shrugs at
it.

Keep the keystore out of the repo. `android/.gitignore` covers `key.properties`,
`**/*.keystore` and `**/*.jks`, so a stray copy will not be committed by accident.

### Google holds the app signing key

New apps are enrolled in Play App Signing automatically, with **quantum-ready hybrid
signing using Google-generated keys**. That is what Quark uses. Providing a copy of our own
app signing key instead was possible at app creation and was deliberately not taken.

Three certificates are downloadable from Play Console > Test and release > Setup > App
signing:

| Certificate | Algorithm | Role |
|---|---|---|
| `deployment_cert.der` | RSA | what pre-Android 17 devices verify |
| `hybrid_classical_cert.der` | RSA | classical half of the hybrid signature |
| `hybrid_pqc_cert.der` | ML-DSA-65 | post-quantum half of the hybrid signature |

These are **public certificates**. They contain no private key and nothing to leak — they
exist so anyone can verify what Play signed. Google holds the private keys, and the hybrid
scheme is not something an upload key could reproduce.

The practical consequences:

- **Losing the upload key is recoverable.** Request an upload key reset in the Play
  Console, generate a new keystore, and updates keep flowing to installed users, because
  the app signing key is untouched. Still back it up — a reset is support friction, not a
  disaster, and it is avoidable.
- **The fingerprint to publish** for anyone verifying a sideloaded download is the
  *deployment certificate's*, not the upload key's. Users verify what Play signed.
- **CI's printed fingerprint** should be compared against Play's **upload** certificate,
  never the app signing one. They are supposed to differ.

### The APK built here is not distributable

`make build/frontend/android/apk` produces an APK signed with the **upload key**. Play
installs a build signed with the **app signing key**. Those signatures do not match, and
Android treats a different signature as a different app.

**Anyone who installs this APK cannot then take a Play update.** They would have to
uninstall first, losing local app data. Do not put it on a release page, a website, or in
front of a user. It is for installing on your own device to check a build.

The APK for direct distribution has to come *from* Play, after it processes the bundle:

- Play Console > Test and release > Latest releases and bundles > select the bundle >
  **Downloads** tab > signed universal APK, or
- the Play Developer API, `generatedapks.list` and `generatedapks.download`.

That APK carries the app signing key's signature, so a Play install and a sideloaded
install remain the same app and users can move between them. Automating that pull is
tracked with the rest of the non-Google distribution work and is not wired up yet.

### `android/key.properties`

Gitignored, read by `android/app/build.gradle.kts`. Create it once locally:

```properties
storeFile=/absolute/path/to/quark-release.jks
storePassword=...
keyAlias=quark-upload
keyPassword=...
```

A relative `storeFile` resolves against `android/`, not the repo root.

**Without this file the release build falls back to the Android debug key**, so
`flutter run --release` still works for a developer with no keystore. Gradle prints a
line saying so. Play rejects debug-signed uploads, so that build can never be published by
accident.

## The first release is uploaded by hand

Play requires an **AAB** for a new app — a standalone APK is not accepted as a first
release — and the very first one has to go through the Play Console by hand. Automated
publishing through the Play Developer API is a follow-up change; today this pipeline's job
is to hand you a correctly signed AAB.

1. Run the build, or download the `android-aab` artifact from a
   [release-android workflow run](../../../actions/workflows/release-android.yml).
2. Play Console > Test and release > Internal testing > Create new release.
3. Upload `quark-v<version>.aab`.

The app signing key is Google's and is set up automatically — see
[Google holds the app signing key](#google-holds-the-app-signing-key). Upload the **AAB**,
never the APK from `build/android-release/`.

## Building

```bash
make check/frontend/android/release   # prerequisites, before a long Gradle run
make build/frontend/android/aab       # signed AAB, for the Play Console
make build/frontend/android/apk       # signed universal APK, for direct install
```

Both land in `build/android-release/` under their published names:

| File | Goes to |
|---|---|
| `quark-v<version>.aab` | Play Console, by hand |
| `quark-v<version>-universal.apk` | your own device, for testing only |

The APK is **not** publishable — see
[The APK built here is not distributable](#the-apk-built-here-is-not-distributable). The
name is nonetheless kept stable and predictable, because direct distribution will need one
Obtainium can match with a regex, and changing the pattern later breaks update detection
for everyone already tracking it.

Builds are near-silent by default: the Makefile sets `.SILENT:`, and Flutter prints a
single spinner line while Gradle runs. To see what is happening:

```bash
make build/frontend/android/aab FLUTTER_VERBOSE=1
```

Each build target verifies its own artifact is a genuine release build before handing it
over — see [Debug build shipped to a store](#debug-build-shipped-to-a-store). To check one
by hand:

```bash
make check/frontend/android/aab                    # newest AAB in the output dir
make check/frontend/android/apk ANDROID_APK=path   # a specific one
```

A universal APK, not per-ABI splits: one file is simpler to install by hand, and direct
distribution will want a single asset to match rather than three to choose between. The
size cost never reaches Play users — Play serves per-device APKs generated from the AAB
regardless.

## Continuous integration

`.github/workflows/release-android.yml` builds on an `ubuntu-latest` runner — Android
builds are Linux-native, so unlike iOS this costs no macOS runner minutes. It runs
automatically as a job in `release.yml` on any `v*.*.*` tag push, after the GoReleaser job
succeeds, and can also be triggered by hand from the Actions tab.

Every run — tag or manual — produces both artifacts as workflow artifacts and publishes
nothing. Download `android-aab` from the run to get the file for the Play Console. No run
touches a GitHub Release, so the pipeline can be exercised at any time without side
effects.

CI runs the same `make` targets as a local build, so the two cannot drift. The only
difference is where the keystore comes from: locally `android/key.properties` is a file you
created, while CI writes it from secrets and deletes it again.

### Secrets

All four are required. Add them under Settings > Secrets and variables > Actions.

| Secret | What it is |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | base64 of `quark-release.jks` |
| `ANDROID_KEYSTORE_PASSWORD` | the keystore password |
| `ANDROID_KEY_ALIAS` | the key alias, `quark-upload` |
| `ANDROID_KEY_PASSWORD` | the key password (same as the store password, for PKCS12) |

```bash
base64 -i quark-release.jks | gh secret set ANDROID_KEYSTORE_BASE64 --repo autobutler-org/quark
```

`scripts/android-ci-signing.bash` proves all four signing secrets actually open the key
before any build starts, and names the one that is wrong. It also prints the key's SHA-256
fingerprint. Compare it against the **upload** certificate shown under Test and release >
Setup > App signing — not the app signing certificate, which is Google's and is supposed to
differ. A mismatch there means CI holds the wrong keystore and Play will reject the
upload.

### Expiry

The upload keystore above is valid for 10000 days. Play rejects a signing certificate that
expires before 2033, which a shorter `-validity` would produce. The app signing key's
lifetime is Google's to manage. Nothing else here expires — unlike iOS, there are no
annually-renewed certificates or profiles.

## Versioning rules

Neither number lives in `pubspec.yaml` — it has no `version:` field at all.

The **versionName** comes from the most recent git tag with the leading `v` stripped:
`v0.33.1` builds `0.33.1`. A build cannot claim a version that was never released, and
there is no second place to remember to edit.

The **versionCode** is computed from that version as `major*10000 + minor*100 + patch`:

| Tag | versionCode |
|---|---|
| `v0.33.1` | 3301 |
| `v0.34.0` | 3400 |
| `v1.0.0` | 10000 |

```bash
make build/frontend/android/aab                          # both derived from the tag
BUILD_NAME=0.34.0 make build/frontend/android/aab        # override the version
ANDROID_BUILD_NUMBER=3402 make build/frontend/android/aab  # override the versionCode
```

Both build targets take the same overrides. Play requires versionCode to be **strictly
increasing, permanently**, and never frees a
consumed one. The obvious alternative is a trap: Flutter derives versionCode by stripping
non-digits from the build number, so passing the version alone gives `0.33.1` -> `331` and
then `1.0.0` -> `100`, which goes *backwards* at the 1.0 boundary and locks the app out of
Play. The positional formula is monotonic across that boundary and stays far below the
int32 cap of 2100000000 (good to version 209999.99.99).

`github.run_number` is not usable either: it is a per-workflow-file counter, so a tag
release and a standalone dispatch draw from different sequences.

If Play rejects an upload as a duplicate — typically after a failed publish that got far
enough to consume the code — pass `ANDROID_BUILD_NUMBER` explicitly, above the last one
Play accepted.

Both numbers need git tags present. `actions/checkout` is shallow by default and fetches
none, so the Android jobs set `fetch-tags: true`. Without them the build fails up front
rather than silently shipping `1.0.0`, which is what Flutter defaults to.

## What is already configured

In `android/app/build.gradle.kts`:

- `compileSdk`, `minSdk` and `targetSdk` are pinned to `36`, `24` and `36` rather than read
  from `flutter.*`, so upgrading the Flutter SDK cannot silently move the levels a
  published app is compiled and tested against. Play enforces a rolling minimum
  `targetSdk`; raising these stays a deliberate commit.
- A `release` signing config, present only when `key.properties` exists.

In `android/app/src/main/AndroidManifest.xml`:

- `usesCleartextTraffic="true"` with a `networkSecurityConfig`, because the app talks to a
  self-hosted server over plain HTTP on the LAN.

## Troubleshooting

### `Version code N has already been used`

Play has seen that versionCode and will never free it. Pass a higher one:

```bash
make build/frontend/android/aab ANDROID_BUILD_NUMBER=<higher>
```

### `Release app bundle failed to strip debug symbols from native libraries`

Misleading message: the NDK is fine and Gradle reported `BUILD SUCCESSFUL`. The real cause
is on the preceding line of `--verbose` output:

```
Failed to find cmdline-tools when checking final appbundle for debug symbols.
```

`flutter build appbundle` verifies the finished bundle with `apkanalyzer`, which ships in
the Android SDK's **cmdline-tools** package. Without it the build fails at the very end,
after the whole Gradle run, pointing at the NDK instead of the missing package.
`flutter doctor` reports it as "cmdline-tools component is missing".

```bash
sdkmanager --install 'cmdline-tools;latest'
```

or Android Studio > Settings > Languages & Frameworks > Android SDK > SDK Tools > tick
**Android SDK Command-line Tools**.

`make check/frontend/android/release` checks for this up front, so the failure costs six
seconds rather than a full build.

### `Failed to read key ... from store`

The four signing values disagree. Run `make check/frontend/android/release` locally, or
look at the "Prepare code signing" step in CI — `scripts/android-ci-signing.bash` names
the specific secret at fault.

The most common cause is a PKCS12 keystore with a key password different from the store
password. PKCS12 supports only one; `keytool` ignores the mismatch with a warning, Gradle
does not.

### Debug build shipped to a store

The iOS pipeline shipped a debug build to TestFlight once. It installed and then refused
to launch. The Android failure mode is the same, and so is the cause.

**`FLUTTER_BUILD_MODE=debug` leaking into the build environment.** The bare `export` near
the top of the Makefile is not scoped to `.env` — GNU make applies it to *every* variable,
so `FLUTTER_BUILD_MODE ?= debug` reaches the environment of every recipe, and Flutter's
build backend reads it in preference to the real configuration. The build reports success.

`build/frontend/android/aab` pins the variable for its own recipe, exactly as the iOS
target does:

```make
build/frontend/android/aab: FLUTTER_BUILD_MODE := release
```

The other frontend targets pass `--$(FLUTTER_BUILD_MODE)` and stay self-consistent; only
the two release targets hardcode `--release`, which is why they alone can diverge. CI never
hits this — there is no `.env` on a runner, so the `export` block never activates.

To confirm the variable is clean:

```bash
make --eval='p:; @env | grep FLUTTER_BUILD_MODE' p    # should print nothing
```

How to tell an AAB apart:

| | debug | release |
|---|---|---|
| `base/assets/flutter_assets/kernel_blob.bin` | present | absent |
| `base/lib/<abi>/libapp.so` | absent | present, tens of MB |

An APK nests the same entries one level higher (`lib/<abi>/libapp.so`).
`make check/frontend/android/aab` and `make check/frontend/android/apk` check both
conditions, and each build target runs its own check automatically.

## Before submitting for review

Getting a build onto the internal track is not the same as passing review. Play review is
tracked separately; the known outstanding items are:

- [ ] **Data Safety form** — must tell the same "we collect nothing" story as the iOS
      privacy labels, exactly.
- [ ] **Content rating questionnaire** (IARC).
- [ ] **Privacy policy URL** — required for every app. Quark does not have one yet.
- [ ] **A reachable demo server** — the highest rejection risk, same as iOS. A reviewer has
      no Quark device. Put the URL and credentials in the reviewer notes.
- [ ] **Closed testing requirement** — new developer accounts must run a closed test before
      production access is granted. This does not gate the internal track, but it can add
      weeks before public release.
- [ ] Store listing: screenshots, feature graphic, short and full description, category.
