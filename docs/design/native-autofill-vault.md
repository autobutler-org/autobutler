Status: Proposed

<!-- cspell:words lazysodium Tink badexample generichash outlen jailbroken -->

# How native autofill opens the vault

Design for [#3594](https://github.com/autobutler-org/quark/issues/3594), part of the native autofill epic
[#3592](https://github.com/autobutler-org/quark/issues/3592). It covers how the iOS AutoFill Credential Provider
extension and the Android `AutofillService` get the vault key, where they get the ciphertext, where the crypto runs,
and what happens when something is revoked. No platform code is written until the owner approves this page or sends it
back. Every decision marked **Owner call** is a security tradeoff that this page lays out and leaves open. The sub-issues
start from the recommendations below, and the owner's answers replace them where they differ.

#3594 suggested `docs/architecture/native-autofill.md`. This page is a proposal, so it sits beside the other proposals
in `docs/design/`. Once it is approved and built, #3603 moves the decisions that stand into `docs/architecture/`.

## What is fixed

These come from the epic and the vault as it ships today. The design works around them.

1. **End-to-end encryption.** The Quark stores ciphertext, nonce and `formatVersion` per item, and the vault key only
   wrapped (`docs/chat-security.md`, "The vault"). Autofill never sends a key or plaintext anywhere.
2. **No server contract change.** The browser extension's contract (`internal/server/extension_contract_test.go`)
   stays as it is. Nothing here needs a new route, field or status, and the
   [extension](https://github.com/autobutler-org/quark-browser-extensions) needs no change.
3. **The construction is permanent.** The vault key is a random 32-byte XChaCha20-Poly1305 key. An item is
   `XChaCha20-Poly1305(vaultKey, nonce, plaintext, ad)` with
   `ad = utf8("quark-vault-item-v1") || 0x00 || formatVersion` (`lib/services/account_crypto.dart:229-289`). The key's
   password wrap is `nonce || XChaCha20-Poly1305(vaultWrapKey, vaultKey)`, and `vaultWrapKey` comes from
   Argon2id13 (3 passes, 64 MiB, `KdfParams.standard`, `lib/models/account_keys.dart:28`) and then HKDF-SHA256 `vault-wrap`
   (`account_crypto.dart:44-55`).
4. **Passkeys and Android Credential Manager are out of scope.** Nothing here should make them harder later.

## What exists today

- **Unlock** is the login password only (`lib/controllers/vault_controller.dart:340-365`). The open key and the
  decrypted items live in the controller's memory and are forgotten on `lock()` (`:368`), after 15 idle minutes
  (`lib/utils/vault_config.dart:6`, `touch()` at `:384-387`), and on a session change (`_onSessionChanged`, `:616`).
- **Keystore use.** The chat identity seeds (`lib/utils/chat_identity_cache_io.dart`) and the session tokens
  (`lib/services/app_settings.dart:316`) are in the platform keystore through `flutter_secure_storage`. They have no
  biometric gate and no Keychain access group, so an extension cannot read them. `docs/chat-security.md` already says
  that anyone who can use the signed-in app can read chat. The vault does not work that way today: its key is never
  written to disk.
- **iOS.** The bundle id is `org.autobutler.quark`, the team is `4NK7MWUA57`, and the deployment target is 15.0
  (`ios/Runner.xcodeproj/project.pbxproj:470`, `:486`, `:493`). There is no entitlements file, no app group, no
  extension target, no `NSFaceIDUsageDescription`, and no URL scheme.
- **Android.** `applicationId = "org.autobutler.quark"`, Kotlin namespace `org.autobutler`, `minSdk = 24`
  (`android/app/build.gradle.kts:35`, `:52-53`). `allowBackup` is not set, so it defaults to on.
- **Multiple Quarks.** The app keeps several hosts with a session token for each (`app_settings.dart`), and each account
  on each host has its own vault.

## Summary of the proposal

| Question | Recommendation |
| --- | --- |
| Where the crypto runs | Native libsodium: swift-sodium on iOS, lazysodium-android on Android. No Flutter engine in the extension or the service. |
| Does the extension run Argon2id | No. It never sees the password. A locked vault with no usable cached key sends the user to the app. |
| Where the key lives | The unwrapped vault key in a biometric-gated keystore entry that never leaves the device. On iOS that is a Keychain item in a shared access group. On Android it is wrapped by a Keystore AES key that needs user authentication for every use. |
| How it unlocks | A biometric check for every fill, or for every opening of the extension's list. The 15-minute idle lock applies to the open key in memory, not to the cached entry. |
| Where the ciphertext comes from | A local copy of `GET /api/v0/vault/items` that the app writes and keeps current. The extension and the service never call the Quark. |
| Stale key detection | A key fingerprint stored beside both the cached key and the ciphertext cache, plus "no item opens" when the app reloads. |
| iOS floor | iOS 15.0, using the password APIs that 15 already has. Later APIs only behind `#available`. |
| Android floor | API 26 for the service. The app keeps API 24. |

Each row is argued below.

## Names the sub-issues use

These are proposals. #3595 and #3598 create them, and once Apple or Google has seen an App ID or a component name it is
hard to change, so the owner should confirm them (see [Q9](#open-questions-for-the-owner)).

| What | Value |
| --- | --- |
| iOS app bundle id | `org.autobutler.quark` (unchanged) |
| iOS extension bundle id | `org.autobutler.quark.CredentialProvider` |
| iOS extension target and folder | `CredentialProvider`, in `ios/CredentialProvider/` |
| App group (both targets) | `group.org.autobutler.quark` |
| Keychain access group (both targets) | `$(AppIdentifierPrefix)org.autobutler.quark.shared`, which resolves to `4NK7MWUA57.org.autobutler.quark.shared` |
| Extension entitlement | `com.apple.developer.authentication-services.autofill-credential-provider` |
| iOS URL scheme, if the extension can use it (see [Failure UX](#failure-ux)) | `quark://vault` |
| Android service | `org.autobutler.autofill.QuarkAutofillService`, `android:permission="android.permission.BIND_AUTOFILL_SERVICE"` |
| Android unlock activity | `org.autobutler.autofill.AutofillUnlockActivity`, not exported, launched through the dataset's `IntentSender` |
| Android service config | `res/xml/autofill_service.xml`, with `android:settingsActivity` pointing at the app's vault settings |
| Android API guard | `res/values/bools.xml` `quark_autofill_enabled=false`, `res/values-v26/bools.xml` `true`, read by `android:enabled` |
| Android Keystore alias | `quark_vault_autofill_v1` |
| Keystore and Keychain entry account key | `acct_<hex(sha256(normalizedHost || 0x00 || username))[0:32]>`, one entry per account |
| Ciphertext cache file | `vault-autofill/<account key>.json` in the app group container (iOS) or `noBackupFilesDir` (Android) |

## Decision 1: where the crypto runs

The extension and the service need XChaCha20-Poly1305 decryption, plus one keyed hash for the fingerprint. If
Decision 2 holds, they never derive keys. Encryption is needed only for Android save (#3602).

### Options

| Option | For | Against |
| --- | --- | --- |
| **A. Native libsodium**: swift-sodium (SPM) on iOS, lazysodium-android (JNA plus the bundled `libsodium.so`) on Android | The same C library the Dart side uses (`package:sodium`), so the same bytes come out. A small footprint, with no engine startup in a short-lived extension. | Two new native dependencies, and two more implementations to keep in step with Dart. #3597's vectors handle that. |
| B. A headless Flutter engine running `AccountCrypto` as is | One implementation. | An engine started inside a memory-capped extension for every fill. Cold-start latency on the fill path. Plugins in an app extension are poorly supported. The Android service would host an engine inside a system-bound service. |
| C. Platform crypto (CryptoKit, Tink) | No third-party binary on iOS. | CryptoKit has ChaChaPoly with 12-byte nonces only. XChaCha20 needs HChaCha20, which means writing crypto by hand. Tink has XChaCha20-Poly1305 on Android, but iOS would still need its own answer. |
| D. A shared Rust or C core over FFI to Swift, Kotlin and Dart | One implementation for all three. | A new toolchain in the build and in CI. Too much for one AEAD call. |

**Recommendation: A.** The native code is small: open a wrap, open an item, check `formatVersion == 1`, parse the item
JSON, match a host, and compute the fingerprint. It is pinned to Dart by test vectors, as described next. Neither the
extension nor the service links Argon2id code, so a regression cannot quietly start running it.

### Test vectors that pin the native code to Dart

#3597 adds one vector file, `test/fixtures/vault_vectors.json`. The Dart, Swift and Kotlin suites all read it, and a
comment in it names the source of each vector. It holds:

1. **Wrap vectors.** The pinned keys from `test/services/account_crypto_test.dart:23-48`: a password, a salt and
   `KdfParams` give an expected `vaultWrapKey`, and a wrap opens to an expected vault key. Native code only checks the
   unwrap half, because it never runs Argon2id. Dart checks both halves.
2. **Item vectors.** A fixed key, nonce, `formatVersion` and ciphertext, with the expected plaintext JSON. Reuse the
   vector quark-browser-extensions already pins (`test/vaultCrypto.test.ts:8-33`), so that all four clients agree on
   the same bytes. Negative cases: the wrong key, `formatVersion` 2, one byte flipped in the ciphertext, a 23-byte
   nonce, and a ciphertext shorter than the tag. Every one of them must fail closed.
3. **Fingerprint vectors.** A key gives an expected fingerprint (defined in Decision 2).
4. **Host matching table.** Rows of URL and site: what `VaultItemData.hostOf` returns (`lib/models/vault_item.dart:117`),
   and whether `entriesForUrl` matches (`vault_controller.dart:244-254`). Include a scheme-less URL, uppercase, a port,
   a parent domain (`accounts.example.com` against `example.com`), a lookalike suffix (`badexample.com` against
   `example.com`, which must not match), an IP literal, and a percent-encoded host (empty).
5. **Item JSON.** One item of each kind, with and without optional fields, and one with an unknown key. Unknown keys are
   ignored, as `VaultItemData.fromJson` ignores them.

If the file is copied anywhere rather than read in place (the extension's repository, for example), the copy says
where it came from and the Dart test checks that it has not drifted.

## Decision 2: where the key lives and how it unlocks without the password

### What is cached

| Option | For | Against |
| --- | --- | --- |
| **A. The vault key, behind a biometric-gated keystore entry** | No Argon2id outside the app. One key opens every item. A password change does not invalidate it (JN-VT-020 re-wraps the same key). | The vault key is on disk for the first time, protected by the device and biometrics rather than by the password. |
| B. `vaultWrapKey` (the password-derived key) | Also skips Argon2id. | Strictly worse than A. It opens the password wrap, and that changes with every password change, so the cache goes stale for no gain. |
| C. Nothing: the extension asks for the login password | Nothing new on disk. | Argon2id at 64 MiB inside the extension (see [Memory](#memory-and-argon2id)), and the user types the master password into a keyboard extension context on every fill. This is the experience the epic exists to replace. |

**Recommendation: A**, as the epic assumes. **Owner call ([Q1](#open-questions-for-the-owner)):** this changes what
the vault promises. Today the key is in memory only, and a stolen phone with no password gives nothing. Under A,
whoever can pass the biometric (or the passcode, if [Q2](#open-questions-for-the-owner) allows it) on an unlocked
phone can fill from the vault. It is opt-in per device, under "Unlock with Face ID / fingerprint" (#3595), and off by
default.

### iOS: a Keychain item with biometric access control

- A generic-password item. `kSecAttrService` = `org.autobutler.quark.vault`, `kSecAttrAccount` = the account key,
  `kSecAttrAccessGroup` = the shared access group, and `kSecValueData` = the 32-byte vault key.
- `kSecAttrAccessControl`: `SecAccessControlCreateWithFlags(..., kSecAttrAccessibleWhenPasscodeSetThisDeviceOnly,
  .biometryCurrentSet, ...)`. The effects:
  - `ThisDeviceOnly`: the item is never in an iCloud Keychain sync or a device backup, and is not restored onto
    another phone.
  - `WhenPasscodeSet`: iOS deletes the item if the passcode is turned off.
  - `.biometryCurrentSet`: enrolling or removing a face or fingerprint makes the item unreadable. #3595 then deletes
    it and falls back to the password. Passcode fallback is excluded unless [Q2](#open-questions-for-the-owner) says
    otherwise, in which case the flag becomes `.userPresence`. That flag survives enrollment changes, which is exactly
    the tradeoff Q2 is about.
- **Reboot.** The item survives a reboot. After a reboot iOS requires the passcode before Face ID works at all, so the
  first fill after a reboot follows a passcode unlock of the phone, which is iOS's rule and not ours.
- **Uninstall.** Keychain items outlive an app's deletion. On first launch after an install (a flag in
  `UserDefaults`, which uninstall does clear), the app deletes every item in the `org.autobutler.quark.vault` service.
  Otherwise a reinstall would find a key from a previous life.
- Reading the item shows the Face ID prompt. `NSFaceIDUsageDescription` goes in both `ios/Runner/Info.plist` and the
  extension's `Info.plist`.

### Android: a Keystore key that needs authentication for every use

- An AES-256-GCM key generated in `AndroidKeyStore` under `quark_vault_autofill_v1` with:
  - `setUserAuthenticationRequired(true)` and authentication for every use: `setUserAuthenticationParameters(0,
    AUTH_BIOMETRIC_STRONG)` on API 30+, and `setUserAuthenticationValidityDurationSeconds(-1)` on API 26-29. Each
    decryption needs a `BiometricPrompt` bound to that `Cipher` through a `CryptoObject`. A hooked prompt that only
    reports success cannot use the key.
  - `setInvalidatedByBiometricEnrollment(true)`. A new enrollment makes the key unusable
    (`KeyPermanentlyInvalidatedException`), and the app deletes it and falls back to the password.
  - `setUnlockedDeviceRequired(true)` on API 28+.
  - `setIsStrongBoxBacked(true)` where `PackageManager.FEATURE_STRONGBOX_KEYSTORE` is present, and the TEE otherwise.
    #3595 records which one was used. A software-only Keystore is treated as unavailable: no biometric unlock offered.
- The vault key, encrypted under that key (a fresh 12-byte GCM IV, with the account key as additional data), goes in a
  small file per account in `noBackupFilesDir`. Keystore keys are never backed up, so that file alone is useless off
  the device.
- `flutter_secure_storage` is not used for this. It does not expose `CryptoObject`-bound keys or per-use
  authentication. #3595 adds a small platform channel (`org.autobutler/vault_keystore`), shared by the app's vault page
  and the native service, so both use one Kotlin implementation.
- **Reboot.** The key survives. With `setUnlockedDeviceRequired`, it is usable only once the user has unlocked the
  phone after boot.
- **Device credential fallback** (passcode or pattern instead of a fingerprint) is
  [Q2](#open-questions-for-the-owner). Allowing it means `AUTH_BIOMETRIC_STRONG | AUTH_DEVICE_CREDENTIAL` (API 30+
  only), and it gives up invalidation on enrollment.

### The idle lock and how often to ask

The app's 15-minute idle lock exists so that a vault left open on screen closes itself. A cached key behind a per-use
biometric check is a different thing: nothing is open while the user is not filling.

| Option | What it means |
| --- | --- |
| **A. A biometric check for every fill, and no expiry on the cached entry** | iOS: every Keychain read prompts. Android: every `Cipher` use prompts. The 15-minute idle lock still governs the open key in the app's memory and in the extension's memory while its list is on screen. |
| B. Store an expiry beside the cached entry, and delete it 15 minutes after the last password unlock | Autofill works for 15 minutes after the user typed the password into the app and never otherwise, so the feature does almost nothing. |
| C. A reuse window: one biometric check covers fills for N minutes (iOS `LAContext.touchIDAuthenticationAllowableReuseDuration`, Android an auth validity of N seconds) | Fewer prompts. Anyone holding the unlocked phone in that window fills without a check. |

**Recommendation: A**, because it is the only option where "the cached key is protected by biometrics" (the epic's
acceptance criterion) holds at the moment of every fill. **Owner call ([Q3](#open-questions-for-the-owner)):** whether
to add a maximum age, so that the login password must be typed into the app again after N days (password managers
commonly use two weeks), and whether a short reuse window (C) is acceptable. Both are cheap to add later, and neither
changes the storage format.

**What the app's Lock button does.** It forgets the in-memory key, as it does today. It does not delete the cached
entry, or every manual lock would turn autofill off until the next password unlock. Turning off "Unlock with Face ID /
fingerprint" deletes it. #3603 documents this in the journeys.

### Detecting a stale cached key

The vault key changes when the vault is replaced: "Start over" (JN-VT-021, `vault_controller.dart:375`), or a deletion
followed by a new setup from another session. A password change re-wraps the same key and does not change it.

- **A fingerprint, for local consistency.** `fp = crypto_generichash(outlen = 16, message =
  utf8("quark-vault-key-fingerprint-v1"), key = vaultKey)`, which is keyed BLAKE2b. It says nothing about the key
  without the key. The app writes it beside the cached key (in the Keychain item's `kSecAttrGeneric`, or in the
  Android file's header) and into the ciphertext cache. If the extension or the service finds that the two disagree,
  it treats the state as locked and fills nothing. That catches a crash between writing the new key and writing the
  new cache. It is in the vectors file.
- **Failed decryption, for replacement elsewhere.** The fingerprint cannot be checked against the Quark without a
  server change, so the app relies on the items instead. On every `load()` while a cached key exists, if the server's
  list has at least one item at `formatVersion` 1 and none of them opens under the cached key, the vault was replaced.
  The app deletes the cached key and the cache, and asks for the password next time. If `GET /vault/key` answers 404,
  the vault is gone, and the app deletes both. An empty vault after a reset leaves a cached key that opens nothing,
  which is harmless until the first new item fails to open.
- **Until the app next runs**, the extension fills from the last cache it was given, which is what the vault looked
  like then. That is the cost of not giving the extension a session. See [Revocation](#revocation).

### More than one account

Each signed-in account that turned on biometric unlock has its own cached key and its own ciphertext cache, under its
account key. The extension lists matches from all of them, labeled by account when there is more than one. On iOS one
`LAContext` evaluated once is passed to every Keychain read, so one Face ID covers all accounts. On Android one
Keystore key wraps every account's vault key, so one prompt does too. Whether a second Quark's vault should be
fillable at all is part of [Q1](#open-questions-for-the-owner). If not, only the active account is cached.

## Decision 3: where the ciphertext comes from

| Option | For | Against |
| --- | --- | --- |
| **A. A local cache of `GET /api/v0/vault/items` that the app keeps current** | Works offline and on a slow network. The extension holds no session token and makes no network call, so a malicious or unreachable Quark cannot affect a fill. No stale-token failures. | It is only as fresh as the app's last sync. |
| B. The extension calls `GET /api/v0/vault/items` with a session token shared through the access group | Always current. | A bearer token for the whole account sits in a shared Keychain group, and it opens every route, not just the vault. A password change elsewhere signs every other session out (`docs/auth.md`, "Changing a password"), so the token dies without warning and fills fail until the app is opened. Network latency lands on the fill path, inside the extension's time budget. |
| C. A, plus a background refresh with B | Freshest. | All the costs of B. |

**Recommendation: A.** The cache holds what the server returned and nothing else:

```json
{
  "schema": 1,
  "host": "https://quark.local",
  "username": "you",
  "keyFingerprint": "<base64, 16 bytes>",
  "syncedAt": "2026-10-20T12:00:00Z",
  "items": [{ "id": 7, "ciphertext": "<base64>", "nonce": "<base64>", "formatVersion": 1, "updatedAt": "..." }]
}
```

- **Writing it.** On every successful `_fetchItems` (`vault_controller.dart:537`, which runs on unlock, on reload and on
  `vault_changed` and `resync`, `:610-613`), the app writes it atomically: to a temporary file in the same directory,
  then a rename. The bytes come from the server's response, not from re-encrypting decrypted items, so the cache can
  never contain anything the server did not already hold. #3596's test compares them.
- **A locked app can refresh it.** The list is ciphertext and needs only the session, so `load()` can refresh the
  cache on `vault_changed` even while the vault page is locked. #3596 should do that, because it keeps the extension
  current without asking the user to unlock.
- **iOS.** It goes in the app group container (`FileManager.containerURL(forSecurityApplicationGroupIdentifier:)`),
  under `Library/Application Support/vault-autofill/`. The data protection class is
  `completeUntilFirstUserAuthentication`, so the app can write it from a background event after the first unlock.
  The file is ciphertext, so a stricter class buys nothing. The directory is excluded from backup
  (`isExcludedFromBackup`), because a restored cache has no key on the new phone.
- **Android.** It goes in `noBackupFilesDir/vault-autofill/`. The service runs in the app's own process and user
  ID, so it reads the files directly.
- **Connection info.** The host and username in the cache are there for labels and for matching an account key. The
  extension never uses them to connect.

**What the extension decrypts.** After the biometric check it opens every item in the matching accounts' caches to read
the hosts (they are inside the ciphertext), keeps only the matches' names and usernames, and holds a match's password
only when that match is chosen. Plaintext is never written anywhere, and dataset presentations never contain a
password (#3600).

## Decision 4: platform floor

**iOS.** The password APIs from iOS 12 cover everything this epic needs:
`prepareCredentialList(for: [ASCredentialServiceIdentifier])`, `provideCredentialWithoutUserInteraction(for:
ASPasswordCredentialIdentity)`, `prepareInterfaceToProvideCredential(for:)` and `ASCredentialIdentityStore`. The
iOS 17 request-based API (`ASCredentialRequest`, `prepareCredentialList(for:requestParameters:)`) exists mainly for
passkeys, which are out of scope. iOS 18's one-time code list is a possible later feature, because items carry
`totpSecret`. **Recommendation:** keep the extension at the app's 15.0 floor, adopt the newer methods behind
`if #available` only where they replace a deprecated one, and leave the passkey follow-up to raise the floor if it
needs to. #3598's in-app help names both Settings paths (iOS 18 and later, iOS 17 and earlier).

**Android.** The service is enabled for API 26 and later through the `bool` resource above. API 24 and 25 install and
run with no service. `BIND_AUTOFILL_SERVICE` means only the system can bind it.

## Memory and Argon2id

An iOS app extension runs under a memory limit far below the app's, and a process that exceeds it is killed with no
error the user can see. Apple does not publish the limit for credential provider extensions. Figures around 120 MB are
commonly cited, but they are not a guarantee and may differ by device and iOS version.

**What was measured for this page: nothing yet.** This design was written without a device. #3594's acceptance
criterion asks for the measurement to be recorded here, so it is an explicit follow-up: before #3598 merges, its PR
records the extension's peak footprint in Instruments (Allocations and the VM Tracker's "Dirty" size) on the oldest
supported device to hand, for:

1. the empty extension showing its list;
2. a fill from a cache of 1,000 items;
3. one Argon2id13 run at `KdfParams.standard` (64 MiB, 3 passes) in a throwaway build, to know how much headroom a
   password fallback would leave.

The numbers then go into this section, and [Q4](#open-questions-for-the-owner) is answered with them.

What follows from the design without the numbers:

- Argon2id at 64 MiB allocates 64 MiB of RAM by definition, about half of a 120 MB budget before the extension's own
  code, libsodium, the UI and the decrypted list are counted. If the extension never runs it, the question goes away.
- The cost cannot simply be lowered for the extension. `KdfParams` are stored with the wrap and set by the app, and the
  same password must derive the same key on every client. A lower cost would mean a second, weaker password wrap,
  which is a change to the server's key format and to the browser extension.
- So the extension has no password fallback in the first version. If [Q4](#open-questions-for-the-owner) adds one
  later, it must run Argon2id with no Flutter engine and with nothing else large in memory, and the measurement above
  must show it fits.
- The Android service has no comparable hard cap, but it runs Argon2id for the same reason as iOS: never. One design
  for both is easier to reason about.
- 1,000 items of a few hundred bytes each is well under a megabyte of ciphertext. Decrypting all of them to match
  hosts costs nothing next to the limit.

## Threat model

What each attacker reaches, with the recommendations above in place. "Today" is the vault as it ships, where the key
lives in the app's memory only.

| Attacker | Today | With native autofill |
| --- | --- | --- |
| **Holds the phone locked** (stolen while locked, or after a reboot) | Nothing in the vault. | Nothing. Keychain items need the device unlocked, Keystore keys need `setUnlockedDeviceRequired`, and both need the biometric. The ciphertext cache is ciphertext. |
| **Holds the phone unlocked but cannot pass the biometric** (grabbed from the user's hand) | The signed-in app, chat included (already documented), but not the vault without the password. | The same. Every fill and every opening of the extension's list prompts. |
| **Holds the phone unlocked and knows the passcode** (shoulder-surfed, the usual phone theft) | As above. The passcode does not open the vault. | Depends on [Q2](#open-questions-for-the-owner). With biometrics only: the same as above, and enrolling the thief's own face or finger invalidates the cached key. With passcode fallback: **the whole vault**, through the extension or the vault page. iOS's Stolen Device Protection helps only where the user has turned it on. |
| **Can pass the biometric** (the user, coerced or asleep) | Not the vault without the password. | The vault. That is the feature. |
| **Has a backup of the phone** (iTunes, iCloud, Android backup) | Ciphertext at most. | The same. The Keychain item is `ThisDeviceOnly`, Keystore keys are never exported, and the cache directory is excluded from backup. |
| **Another app on the same phone** | Nothing. | iOS: the app group and the access group are limited to apps signed by team `4NK7MWUA57` with those entitlements. Android: app-private storage, and only the system can bind the service. **The open risk is a phishing app on Android** (below). |
| **A rooted or jailbroken phone, the attacker with code execution** | Whatever the app holds in memory while the vault is open, and the password as it is typed. | The same, plus the vault key in memory for the moment of each fill, and every item it opens. A hardware-backed key still cannot be copied off the device or used without a real biometric, so the attacker has to wait for the user to fill. **Root means the vault is as exposed as on a compromised desktop.** Whether to detect and refuse is [Q6](#open-questions-for-the-owner). |
| **A malicious or compromised Quark** | Unchanged: `docs/chat-security.md`'s threat model. | Unchanged. The extension and the service never talk to the Quark, so this adds nothing for it. |
| **A forensic image of the storage** | Ciphertext. | Ciphertext, plus a wrapped key that is useless without the device's secure hardware and a biometric. |

### Android phishing through `webDomain`

On Android the app being filled builds the `AssistStructure`, including `ViewNode.webDomain`. A malicious app can put
`webDomain = "bank.example"` on its own fake login field and receive the bank's password. This is a known attack class
on Android password managers. The countermeasures #3600 must include:

- Trust `webDomain` only from a requesting package on a known-browser allowlist (Chrome, Firefox, Samsung Internet,
  Edge, Brave and so on, checked by package name and signing certificate). For any other package, ignore
  `webDomain` and match the package through #3601, which means no match until #3601 lands.
- The unlock activity shows the requesting app's label and the domain being filled before the biometric prompt, so a
  mismatch is visible.
- Never fill a field that is not visible or not focusable.

On iOS the service identifiers come from the OS. An app's domains are verified through associated domains, so this
attack does not apply in the same form.

### iOS QuickType identities

#3599 stores site addresses and usernames, unencrypted, in the OS's identity store. It is already opt-in in that issue.
This design adds only that the identity store is cleared at every revocation event below.

## Revocation

| Event | What happens to the cached key, the cache and the identity store | Who does it |
| --- | --- | --- |
| Sign-out on this device | All three are deleted for that account before the session is dropped. | App (`_onSessionChanged`, `vault_controller.dart:616`, through #3595 and #3596) |
| Account or Quark switch | If [Q1](#open-questions-for-the-owner) allows every signed-in account: nothing is deleted, because the account stays signed in and the extension keeps listing it. If only the active account may be cached: the previous account's three are deleted, which is how #3595 and #3596 are scoped today. | App (`_onSessionChanged`) |
| Biometric unlock turned off | The cached key is deleted. The cache can stay, because it is ciphertext, but is deleted for tidiness. The identity store is cleared. | App |
| A biometric enrolled or removed | The OS makes the key unusable. The next app or extension read fails, and the app deletes the key and falls back to the password. | OS, then app |
| Passcode turned off | iOS deletes the Keychain item, and Android invalidates the key. | OS |
| Password change on this device | Nothing: the vault key is unchanged (JN-VT-020). | None |
| Password change, or recovery, on another device | This device's session is signed out on the server. The next time the app talks to the Quark it gets a 401, signs out locally, and deletes all three. **Until the app runs, autofill keeps working from the cache.** | App, on next run |
| Start over on this device | All three are deleted before the new vault is made, and the new key is cached after setup if biometric unlock stays on. | App (`startOver`, `:375`) |
| Vault deleted or replaced from another session | The next `load()` sees 404, or sees that no item opens, and deletes all three. | App, on next run |
| App uninstalled | Android removes everything. On iOS the Keychain item survives and is deleted on the next install's first launch. | OS, and app on reinstall |

**Owner call ([Q5](#open-questions-for-the-owner)): remote revocation.** "Change my password because my phone was
stolen" signs the phone's session out but does not stop the extension from filling on it, because the extension has no
session to lose and does not contact the Quark. The cached key still needs a biometric, and a key already on a device
can never truly be taken back. But users may expect "sign out everywhere" to mean autofill too. The options:

- Accept this and say so in the journeys and the settings copy.
- Have the extension check the session before filling. That needs a shared token (Decision 3, option B) and the
  network, and fails closed offline.
- Have the app check on every launch and on every resume, which is what the table already does, and expire the cached
  entry after N days without a successful session check (the same mechanism as Q3's maximum age).

## Failure UX

These principles are for #3598 and #3600. The exact copy goes through `Errors` in the app. The native strings live in
each platform's string resources and follow the same rules.

1. **Fill nothing when unsure.** A locked vault, a stale key, a mismatched fingerprint, an unreadable cache, or no
   match: no credential leaves the extension or the service.
2. **Say what to do, not what broke.** "Open Quark and unlock your vault to use autofill" rather than "Key invalidated"
   or an exception name. No error code, no exception text, no host URL in copy a user reads (#1622).
3. **One next step.** Each failure state has exactly one action. On Android that is an "Open Quark" dataset, which can
   launch the app through its `IntentSender`. On iOS, whether a credential provider extension can open its containing
   app is not established: `NSExtensionContext.open(_:)` is documented for Today widgets, and other extensions are not
   guaranteed to honor it. #3598 tries `quark://vault` on a device. If that does not work, the extension shows text
   telling the user to open Quark, with no dead button.
4. **Name the reason when it changes what the user does.** After an enrollment change: "Your fingerprints or face
   changed, so unlock with your password once." No match: "No saved login for bank.example." Plain words, in the
   language of the people this is for (older, non-technical users), with no "keystore", "biometric enrollment" or
   "cache".
5. **Never blame the user for the system.** A failed or canceled biometric check fills nothing and shows nothing
   alarming. Canceling is not an error.
6. **Show how fresh the data is when it might matter.** If the cache is older than a threshold (#3596 picks it, for
   example a week), the list footer says "Last updated <date>. Open Quark to refresh."
7. **Phishing context on Android.** The unlock screen names the app and site being filled, in words: "Fill your
   bank.example login into Example App?"

## Open questions for the owner

1. **Cache the vault key on the device at all?** This changes the vault from "key in memory only" to "key on the device
   behind biometrics", opt-in per device. Also: may every signed-in account's vault be cached, or only the active
   account's?
2. **Biometrics only, or biometrics or the device passcode?** Biometrics only (iOS `.biometryCurrentSet`, Android
   `AUTH_BIOMETRIC_STRONG` with invalidation on enrollment) protects against the shoulder-surfed-passcode thief, but
   leaves out people with no usable biometric. Allowing the passcode includes them, and gives that thief the vault.
   On Android, Class 2 ("weak") biometrics are never accepted in either case.
3. **How often to re-prove.** A biometric check for every fill is recommended. Should the login password be required
   again after N days (and what N), and is a short reuse window, one prompt covering a few minutes of fills,
   acceptable?
4. **Password fallback in the extension.** None in the first version is recommended. Revisit once #3598's memory
   measurement is recorded here?
5. **Remote revocation.** Accept that signing the phone out from elsewhere stops autofill only when the app next runs,
   or require a periodic session check (and fail closed offline)?
6. **Rooted and jailbroken devices.** Do nothing beyond the threat model's honesty, warn, or refuse biometric unlock?
   Detection on Android means Play Integrity, a dependency on Google services, and it can be bypassed.
7. **Android app matching.** #3601 already says that Google's Digital Asset Links API must be opt-in or ruled out.
   Confirm which, since until then a native app with no browser `webDomain` matches nothing.
8. **iOS floor.** Stay at 15.0 (recommended), or move the extension to 17 or 18 for the newer APIs?
9. **Names.** Confirm the bundle id, app group, access group, Android components and Keystore alias in
   [Names the sub-issues use](#names-the-sub-issues-use) before #3595 and #3598 register them with Apple.

## Recommended order for #3595 to #3603

1. **#3597, native crypto parity and test vectors.** First, because it is pure code with no platform registration and
   no security decision beyond this page, and because #3595's fingerprint and #3598 and #3600's decryption all depend
   on it. It also puts Swift and Kotlin tests in CI before any native feature code exists.
2. **#3596, the shared ciphertext cache.** Next. It is ciphertext only, so it needs none of the owner calls, and it can
   land behind no visible UI. It also creates the app group and the iOS entitlements file, which #3595 and #3598 need.
3. **#3595, biometric unlock and the cached key.** After Q1 to Q3 are answered. It adds the platform channel that the
   vault page and both native pieces use, and is useful on its own: a biometric unlock for the vault page.
4. **#3600, the Android `AutofillService`.** Before iOS, because it needs no new signing profile or App ID, it is easier
   to iterate on, and it is where the phishing countermeasures above must be proven.
5. **#3598, the iOS extension.** Needs the second App Store profile and the CI signing change, and records the memory
   measurement this page is waiting on.
6. **#3601, Android app matching.** After Q7.
7. **#3599, iOS QuickType identities.** Opt-in, and it builds on #3598.
8. **#3602, Android save.** Optional and last. With no session in the service (Decision 3), the recommended shape is to
   encrypt the new item natively and put the ciphertext in an outbox file beside the cache. The app uploads it with
   `POST /api/v0/vault/items` the next time it runs, so no token reaches native code and no plaintext crosses an
   intent.
9. **#3603, docs and journeys.** Folded into each layer as it lands, closing with the move of this page's decisions
   into `docs/architecture/`.
