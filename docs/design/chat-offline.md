Status: Proposed

<!-- cspell:words secretstream NSURLIs -->

# Chat that keeps working offline

Design for the epic [#3552](https://github.com/autobutler-org/quark/issues/3552), part of the chat epic
[#2414](https://github.com/autobutler-org/quark/issues/2414). It covers what chat keeps on the device and how that is
encrypted, a durable outbox for messages written while the Quark is out of reach, reading already-fetched history
offline, how a client catches up when it comes back, and what each state says to the user. No product code is written
until the owner approves this page or sends it back. Each section lays out the options and recommends one. The
tickets at the end start from the recommendations, and the owner's answers replace them where they differ.

## What is fixed

These come from the epic and from chat as it ships today. The design works around them.

1. **The Quark never reads a message** (#2414). Nothing here sends a key or plaintext anywhere.
2. **Plaintext is held in memory only** (`docs/chat-security.md`, Delivery). This page keeps that rule as it is
   written: what reaches the disk is ciphertext the Quark could have stored itself, inside an envelope that also hides
   the metadata.
3. **A nonce is taken once per channel and key version** (#2487), and **a send is named by its `clientMessageId`**
   (#3531, migration `039_chat_message_client_id`). A repeat of `(channel_id, author_id, client_message_id)` is
   answered with the stored row and 200, whatever the ciphertext, and nothing new is stored or announced
   (`internal/server/api/v0/chat/post_message.go`).
4. **Writes are rate limited per account** (#2485): a burst of 100, then 10 a second, per route, in memory. Over that
   is 429 with no `Retry-After`.
5. **The events socket drops and has no replay.** `EventsService` adds its own `resync` on every reconnect, and the
   Quark sends one when a subscriber falls behind. Clients catch up over REST.
6. **Non-goals from the epic:** sending to a Quark that is off (a message is only queued), federation, editing a queued
   message's recipients, any cloud relay.

## What exists today

- **Pending sends** are an in-memory list on `ChatController` (`_pending`, `lib/controllers/chat_controller.dart`).
  `send` adds a `ChatPendingSend` with a random 128-bit `clientMessageId` (`_newClientMessageId`), `_deliver` calls
  `ChatMessagesController.send`, which encrypts afresh and posts. A failure keeps it in `failedSends` with Retry and
  Discard; `retry` sends again under the same `clientMessageId`. A send only goes to the channel on screen: one left
  behind by switching fails with `StateError('left')` (the `ponytail:` note in `_deliver`). On `dispose`, a failed
  send goes back into its channel's draft (`_keepAsDraft`). Drafts are a static map that lives for the app session.
- **The timeline** is `ChatMessagesController` (`lib/controllers/chat_messages_controller.dart`): a `SplayTreeMap` of
  decrypted messages by id. `open` loads the newest page (50) and then `?after=` the newest id. `catchUp`, on every
  reconnect and refresh, re-reads the whole loaded window from the oldest id held, in pages of 200, so missed
  deletions and reactions land too. Its `ponytail:` note flags that whole-window re-read. `dispose` drops everything.
- **Channel keys** are opened from the account's grants on every start (`ChatChannelKeysController`, a map of
  `SecureKey` in memory). A restart fetches the grants again.
- **The identity** (the two 32-byte seeds) sits in the platform keystore through `flutter_secure_storage` on native
  (`lib/utils/chat_identity_cache_io.dart`, key `chat_identity:<host>`), and in the tab's `sessionStorage` on the web
  (#3257). Signing out deletes it (`ChatKeysController.lock(forget: true)`).
- **Read markers** come back with the channel list as `lastReadMessageId` (#3543), and `ChatUnreadController.refresh`
  restores them.
- **Reachability** is `ConnectionController.mode` (home, remote, or neither), shown by the app bar's indicator, and
  `isQuarkUnreachableError` (`lib/utils/connection_error.dart`) maps a failed request to
  `Errors.quarkDisconnectedInline`.
- **Storage the app already depends on:** `flutter_secure_storage`, `shared_preferences`, `path_provider`, and
  `sodium` (libsodium, XChaCha20-Poly1305 and `crypto_secretstream`). There is no on-device database package (no
  SQLite, no IndexedDB wrapper) and no background-task package.
- **The server pages** with `GET /chat/channels/:id/messages?before=|after=&limit=` (at most 200), oldest first,
  tombstones included and each message carrying its reactions. Ids come from the database: SQLite serializes writes,
  but on PostgreSQL two concurrent posts can commit in the other order from their ids.
- **The server accepts any key version the channel has**, not only the current one
  (`chatutil.PostMessage`, the `slices.ContainsFunc(versions, ...)` check).

## Summary of the proposal

| Question | Recommendation |
| --- | --- |
| What is on disk | Server rows exactly as served (ciphertext, tombstones, sealed reactions), the account's sealed grants, the channel list, and the outbox, all inside one envelope per file. Native only. |
| Envelope key | A random 32-byte key per host and account in `flutter_secure_storage`, never derived from the password and never sent. Deleting it shreds every file. |
| Outbox entries | The post body itself: `keyVersion`, channel-key ciphertext and `clientMessageId`, plus when it was written. Encrypted once, re-encrypted only if the key rotates before it goes. |
| Outbox order | One FIFO per channel, sent one at a time. A transient failure holds the channel's queue; a permanent one marks that message failed and the next goes. |
| Offline reading | Up to the newest 500 messages of each channel opened in the last 30 days, within a 50 MiB ceiling per account, evicting the least recently opened channel first. |
| Catch-up | Keep `?after=` plus re-reading the window, but bound the window to what is cached. Further back than that, drop the cached copy and load the newest page. No server change. |
| Web | No history on disk. The outbox lives in `sessionStorage` beside the identity, so a reload keeps it and closing the tab drops it, and the page says so. |
| Server changes | None needed. Two optional hardening tickets: refuse a post under a superseded key version, and a change sequence for bounded catch-up later. |
| New dependencies | None. Background sending while the app is closed would need one, and is left to the owner. |

## Names this page uses

- **Store**: the files under the app support directory for one host and account, and their envelope key.
- **Envelope**: XChaCha20-Poly1305 over a whole file under the envelope key.
- **Cached window**: the contiguous run of a channel's messages, by id, that the store holds.
- **Outbox entry**: one message written on this device that the Quark has not confirmed.

## Decision 1: what is stored on the device, and how it is encrypted

### What a device needs to read offline

To show a channel without the Quark the app needs the channel list and members (names, avatars come from the
existing image cache), the messages, and a way to open them. Opening needs the channel keys, which come from the
account's grants, which open with the identity, which native already keeps in the keystore.

### Options

| | What is on disk | Plaintext at rest | Cost |
| --- | --- | --- | --- |
| A. Rows as served, in an envelope | The server's ciphertext rows and sealed grants, with the metadata, all encrypted again under a device key | None, not even under the device key | Decrypt each message on open, as today. Two layers of crypto. |
| B. Decrypted text in an envelope | The plaintext of each message, under a device key | Yes, under a key in secure storage | Simplest reads. Changes the rule in `docs/chat-security.md`, and the same model as the persistent search index #2429 proposed. |
| C. Rows as served, no envelope | The server's rows in the clear | None | Leaves who said what to whom, and when, readable on the disk to anyone with the file. |

Against an attacker who has the keystore, A and B are equal: the identity in the keystore opens every grant, and so
every channel key. They differ against everything short of that, and in what the rule says. A keeps "plaintext in
memory only" true as written, and a cached row is byte for byte the Quark's row, so a cache and the server page
compare directly in tests and a tombstone is the same replacement on both. C is out: the metadata is the part an
envelope exists to hide.

**Recommendation: A.** B stays the answer if the owner wants on-device search over everything (#2429), and can be
added beside A later without changing A's files.

### The envelope key

- 32 random bytes from libsodium, made the first time chat writes to the store, kept in `flutter_secure_storage` under
  `chat_store_key:<host>:<userId>`. On iOS with `first_unlock_this_device` accessibility, so it never syncs to iCloud
  Keychain or restores onto another device.
- It is **not** derived from the password. The identity on native already opens without one, and a password-derived
  key would make offline reading need the Quark to fetch the wraps first.
- One key per host and account. A second account on the same Quark, or another Quark, has its own key and its own
  directory, so switching accounts never opens the other's files.
- If secure storage is unavailable (a Linux desktop with no Secret Service, `Errors.keyringUnavailable`), there is no
  store: chat works as today, in memory, and the outbox banner says messages will not survive closing the app.

### File layout

```text
<applicationSupportDirectory>/chat/<sha256(host) as hex, first 16 chars>/<userId>/
  channels.bin          the channel list, members, sealed grants, read markers
  channel_<id>.bin      one channel's cached window
  outbox.bin            every outbox entry for this account, all channels
```

Each file is `nonce(24) || XChaCha20-Poly1305(envelopeKey, nonce, body, ad)`, with

```text
ad = "quark-chat-store-v1" 0x00 || sha256(host) || be64(userId) || kind 0x00 || be64(channelId or 0)
```

so a file copied to another account's directory, renamed to another channel, or swapped between kinds fails to open
and is deleted. The body is JSON of the same shapes the API returns. A file is written whole to a temporary name and
renamed over the old one, so a crash leaves the old file or the new one, never half of either. A channel file is at
most 500 rows of at most 16 KiB of ciphertext, about 8 MiB in the worst case and tens of KiB in practice, so writing
it whole is fine. The host is hashed in the path so the directory names do not list which Quarks the device knows.

`getApplicationSupportDirectory` is backed up by default on both platforms (iCloud device backup, Android Auto Backup,
since `allowBackup` is not set). A backup carries ciphertext whose key stays in the keystore, so a restore onto
another device is unreadable and the app deletes it on the first failed open. Excluding the directory from backups
needs platform code on iOS (`NSURLIsExcludedFromBackupKey`) and a backup rules file on Android, and is listed as an
open question rather than assumed.

### Wiping

| When | What goes |
| --- | --- |
| Sign-out | The envelope key first, then the directory. The key going is what makes the files unreadable; deleting the files is tidying. Works with the Quark unreachable. |
| An unsent outbox at sign-out | The sign-out dialog says how many messages have not been sent and that signing out deletes them. |
| Leaving a channel, or being removed | Its channel file, and its outbox entries go failed (see Decision 2). Removal is noticed when the channel drops out of the channel list, or a fetch of it answers 404. |
| A new identity (recovery made a new one) | The whole store: the cached grants were sealed to the old identity and nothing opens them. |
| A file that fails to open | That file. The next fetch rebuilds it. |
| Host removed from the app | That host's directories and keys. |

`docs/chat-security.md` gets a section on the store and two lines in the threat model: a removed member's device keeps
up to the cached window rather than only what was on screen, until it next reaches the Quark; and anyone who can use
the signed-in app can read the cached history offline, as they can already read it online.

## Decision 2: the outbox

### What an entry holds

```text
clientMessageId   the 128-bit id #3531 already makes
channelId
keyVersion        the version it is encrypted under
ciphertext        nonce || XChaCha20-Poly1305 under the channel key, the bytes that will be posted
writtenAt         this device's clock, for display and the 7-day expiry only
state             queued | sending | failed
error             for a failed entry, the kind of failure (never the exception text)
attempts          for the backoff
```

The text is encrypted under the channel key before the entry is written, so the outbox holds what the post will
carry and nothing more, inside the envelope.

### Options for the entry's encryption

| | Encrypted when | On a key rotation before it goes | Queue while waiting for a key |
| --- | --- | --- | --- |
| A. Channel-key ciphertext | At send, once | Open under the old version, encrypt under the current one, same `clientMessageId` | No, as today |
| B. Text under the envelope key | At flush, every attempt | Nothing to do | Yes |

B is less code: flushing calls today's `ChatMessagesController.send(text, clientMessageId:)` unchanged. But it puts
plaintext at rest under the device key, which is Decision 1's option B for the outbox alone. **Recommendation: A**, so
the whole store keeps one rule. The re-encryption is a few lines: the device encrypted under the old version, so it
holds that key.

A re-encrypted entry carries a new nonce and the same `clientMessageId`. If the first attempt had in fact been stored,
the repeat is answered with the stored row, under the old version. That message was written before the rotation and
reached the Quark before it, so it is what any other member's client would have sent in that race today.

### States

```text
            write             flush picks it         201/200
 (typed) ─────────▶ queued ───────────────▶ sending ─────────▶ (stored row replaces it)
                      ▲                       │
                      │   transient failure   │ permanent failure
                      └───────────────────────┤
                                              ▼
                     Retry ◀──────────────── failed ────────▶ Discard (or back to draft)
```

- **queued**: written to `outbox.bin` and renamed into place before the composer clears. That is the durability
  promise: a message the user saw go to "queued" survives the app being killed.
- **sending**: one request in flight. On a restart every `sending` entry is `queued` again. That is safe because the
  retry carries the same `clientMessageId`.
- **stored**: the Quark answered 201 or 200. The returned row goes into the timeline and the channel file, and the
  entry is deleted in the same write. The live `chat_message_created` for it, if it arrives first, is matched by id
  and does nothing new.
- **failed**: waits for the user. Retry puts it back to queued at the tail of its channel's queue. Discard deletes it.
  Copy to draft puts its text back in the composer, as `_keepAsDraft` does today.

### Which failures hold and which fail

| Answer | Kind | What happens |
| --- | --- | --- |
| No response, timeout, socket error, `isQuarkUnreachableError` | transient | Stay queued. The channel's queue holds. |
| 5xx | transient | Same, with backoff. |
| 429 | transient | Same, with backoff. Also pauses every channel's queue, since the limit is per account. |
| 401 | transient | Stays queued while the session is refreshed or the user signs in again. |
| 400 (unknown key version), 403 (lost `send_messages`), 404 (left or removed), 413 | permanent | That entry fails, and the next in its channel goes. |
| 409 (nonce taken) | permanent | Cannot happen for a fresh random nonce. Treated as a failure rather than looped on. |
| Waiting for a key, chat locked | not sent | The entry stays queued, the queue holds, and the page shows that state's message, not "offline". |

**Hold on transient, skip on permanent.** Holding on every failure would let one message the Quark will never take
block a channel forever. Skipping on every failure would send message 3 while message 2 is only waiting for Wi-Fi.
#3481's third scenario asserts exactly this.

### Ordering

- **Within a channel**, entries go one at a time in the order written, and the next starts only after the previous
  is stored or failed. The server assigns ids in arrival order, so the timeline shows them in the order written.
- **Across channels**, queues flush independently, one request per channel at a time.
- **Against other people's messages**, a queued message lands after whatever was posted while this device was away.
  The timeline is ordered by the server's id and `createdAt`, and that stays the one order every reader agrees on.
  The page shows the entry's "written at" time beside its stored time when they differ by more than a minute
  ("Written 2:14 PM, sent 3:02 PM"), from `writtenAt`, which only this device has.

The alternative is to carry the written-at time to the other readers. The Quark cannot be trusted with it and cannot
check it, so it would go inside the ciphertext, and the plaintext today is bare UTF-8. A versioned plaintext format is
a larger change (it touches every reader, search and mentions), and per-message signatures (#3553) would want to
define it too. Left as an open question.

### Retry pacing and the rate limiter

- Exponential backoff with full jitter, from 1 second to 60 seconds, per channel, reset on any success.
- A flush starts on `EventsService.reconnects`, on `ConnectionController` moving from neither to home or remote, on
  the app coming to the foreground, on a successful unlock, and when a waited-for key lands. Never on a timer while
  unreachable beyond the backoff.
- The flusher sends at most 5 posts a second across all channels, half the limiter's steady rate, so a device that
  queued 100 messages offline does not spend the burst that the person's other device may need.
- 429 pauses every queue for the backoff interval.

### Expiry

An entry older than 7 days on this device goes failed with "This message waited too long to send" rather than landing
in a conversation that has moved on. Retry still sends it. The number is an open question.

### Where the outbox lives in the code

The outbox moves out of `ChatController` into its own `ChatOutboxController`, one per host and account, which owns
the queue, the flusher and the file. It does not need the channel on screen. That removes the `StateError('left')`
path: a send to a channel the user navigated away from keeps going. `ChatController.pending`, `outboundSends` and
`failedSends` become views of it for the open channel, so the page's widgets do not change shape.

## Decision 3: offline reading

### The cache window

- **What is cached**: the rows the app has loaded for a channel, as one contiguous run by id ending at the newest
  message the device knows of, capped at the newest 500. `loadOlder` past the cap still works online; those older
  pages are shown but not written.
- **Which channels**: a channel opened in the last 30 days. The channel list is always cached.
- **Ceiling**: 50 MiB per account. Over it, the least recently opened channel's file goes first.
- **Opt-in or not**: on by default on native, with a switch in Settings → Account, "Keep recent messages on this
  device", and a "Remove now" button. Off removes the store at once. Whether it should be off by default is an open
  question.

These numbers are starting values, kept on a `ChatStoreConfig` const class under `lib/utils/`, the house pattern for
tuning values.

### Opening a channel

1. Read the channel file and the grants from `channels.bin`, open the grants with the identity, decrypt the rows into
   the timeline. The page shows them at once, marked as saved on this device until the first fetch lands.
2. If the Quark is reachable, catch up (Decision 4) and rewrite the file.

On native the identity is in the keystore, so reading offline needs no password. If the identity is missing (a new
sign-in that has not unlocked yet), nothing in the store can be opened and the page shows the locked state.

### Key rotation and grants

- The member keeps every version it ever held, so cached rows under an old version still open from the cached grants.
- A row under a version this device holds no grant for is `waiting`, as today, and opens once the grant is fetched.
- A grant the app deletes because it will not verify (the first-write-wins case in `docs/chat-security.md`) is
  dropped from the cache in the same step.
- A new identity wipes the store (see Wiping).

### Tombstones

A deletion is applied to the cache as soon as it is seen, by the event or by catch-up, and that write is not
debounced: deleting a message is a privacy act, and the device should not keep the ciphertext for a few seconds
longer than it needs to. Other cache writes are debounced to one per channel per 2 seconds.

## Decision 4: catch-up

### What stays

`?after=` the newest id for new messages, and a re-read of the window for what changed in it (deletions, reactions).
`TestChatDelivery_ResyncOverlap` and `TestChatConverge_Offline` pin that merging the two never duplicates or drops an
id. The re-read also covers the PostgreSQL case where two posts commit out of id order: a cursor alone would step past
the lower id forever, while the re-read overlaps it.

### What changes

The window was "everything loaded this session", which grows without bound as the user scrolls back. It becomes the
cached window, at most 500 rows, which is three requests of 200.

| Situation on reconnect | What happens |
| --- | --- |
| Up to 500 new messages since the newest cached | Page forward with `?after=`, then re-read the window, as today. |
| More than 500 new | Stop paging forward after 500, drop the cached file, load the newest page, and start a new window there. Older messages load on scroll, online. |
| The channel is gone from the channel list | Wipe it (see Wiping). |
| A cached channel that is not open | Nothing until it is opened. The channel list's `unreadCount` and `lastReadMessageId` (#3543) are enough for the list. |

So reconnecting costs one request for the channel list and, for the open channel, at most five or six pages,
whatever the history length. The `ponytail:` in `catchUp` is resolved by that bound.

### Options for later

| | What it needs | When it pays off |
| --- | --- | --- |
| A. Bounded re-read (recommended) | Nothing on the server | Now. The window is bounded by the cache. |
| B. Change sequence | A migration pair adding a per-channel change counter bumped on post, delete and reaction change, and `?changedSince=` on the list route | When a window much larger than 500 is wanted, or offline caching of every channel at once. |
| C. Deletions route | `GET /channels/:id/deletions?after=` | Only fixes tombstones, not reactions. Not recommended. |

B is a database change, which the house rules make a last resort, and A makes it unnecessary for the windows this page
proposes. It is listed so the owner can see the next step.

### Read markers and reactions offline

- **Read markers** are not queued. `ChatUnreadController` already sends only a forward move; offline it keeps the
  newest id the user reached and sends that one marker on reconnect. The channel list then carries the standing
  marker (#3543), and whichever is further wins.
- **Reactions and deletes** are not queued in this epic. Offline, the reaction picker and Delete are disabled with
  the offline message. Queuing them would mean ordering them against a message that may itself still be queued, for
  little gain. An open question.

### More than one Quark instance

Events reaching every instance is #2965's. This design depends on nothing more than REST catch-up after every
reconnect and every `resync`, which is correct whichever instance served the post.

## Web and native

| | Native (iOS, Android, desktop) | Web |
| --- | --- | --- |
| History on disk | The store, as above | None. In memory for the open channel, as today. |
| Outbox | `outbox.bin` in the store | `sessionStorage`, under an envelope key that is also in `sessionStorage` |
| Survives | App kill, restart, update | A reload. Not closing the tab or opening a new one. |
| Unlock offline | Not needed: the identity is in the keystore | Possible after a reload, since the identity is in `sessionStorage`. A new tab needs the Quark to fetch the wraps. |
| Stated in the UI | "Saved on this device" on cached history | "Messages you send while offline stay in this tab until they're sent." |

On the web the envelope key in `sessionStorage` protects nothing that the identity beside it does not already expose;
it is there so the outbox has one format on every platform. A browser tab's code comes from the Quark it is talking
to (`docs/chat-security.md`), so caching history in IndexedDB would add a durable copy without adding a trust anchor.
`sessionStorage` is about 5 MiB, so the web outbox is capped at 50 entries, and the composer says so when full.

Closing a tab with an unsent outbox shows the browser's leave-page prompt.

## Multi-device convergence

- **Each device has its own outbox.** Another of the account's devices sees a queued message only once it is stored,
  through the same event and catch-up as anyone else. `clientMessageId` is random per entry, so two devices never
  collide, and its index is per author, so another account cannot claim one.
- **Each device has its own cache.** A deletion made on one device reaches the others by event or by the next
  re-read. A read marker moved on one device reaches the others by `chat_read_marker_changed`, or by the channel list.
- **Signing out of one device** wipes that device's store only. Signing out everywhere is not a feature today.
- **A message one device queued and another device discarded** cannot happen: an entry lives on one device.

The converge harness's sessions (`internal/server/api/v0/chat/converge_clients_test.go`) already model two sessions
of one account; the new Go cases below add one that queues while disconnected.

## UX states and copy

Three states the epic asks to keep apart, and the two the outbox adds. All copy is new `Errors` constants or
page-level labels in `lib/utils/error_text.dart`; nothing is composed at a call site, and no exception text reaches the
UI.

| State | How it is known | Banner or row | Composer |
| --- | --- | --- | --- |
| Quark out of reach | `ConnectionController.mode` is neither, or a request fails `isQuarkUnreachableError` | "Can't reach your Quark. You're reading messages saved on this device. New messages will send when it's back." | Enabled. Sends go to queued. |
| Chat locked | `ChatController.isLocked` | The existing unlock prompt. Offline in a new web tab: "Chat is locked. You can unlock it once your Quark is reachable." | Disabled. |
| Waiting for a key | `isWaitingForKey` | `Errors.chatWaitingForKey`, unchanged | Disabled, as today. |
| A queued message | Outbox entry `queued` | Row label "Waiting to send" with a clock icon | — |
| A sending message | `sending` | "Sending…" | — |
| A failed message | `failed` | One sentence per kind, with Retry and Discard: "You're no longer in this channel, so this didn't send." (403/404), `Errors.chatMessageTooLong` (413), "This message waited too long to send." (expiry), else `Errors.message(error, 'send the message')` | — |
| Store unavailable on this device | No secure storage | "Messages you send while offline will be lost if you close the app." | Enabled. |

A queued message is never shown as sent, never counts toward anyone's unread count, never moves this account's read
marker, and is not a search result until it is stored (#3481's fifth scenario). It is searchable as a draft is, not
at all.

The offline banner replaces, not stacks on, the locked and waiting states: if chat is locked and offline, the locked
message wins, since unlocking is what the user can act on first.

## Interplay with key verification (#3553)

#3553 pins members' identity keys on the device (trust on first use), warns hard and blocks sends when a pinned key
changes without re-verification, and may add a regenerate-keys path for abandoned channels and per-message signatures.

- **The pins need a durable home on the device**, and this store is it: a `pins.bin` beside `channels.bin`, under the
  same envelope. But the store is wiped on sign-out, and a pin set that resets at every sign-in is weak trust on first
  use. Whether pins survive sign-out, kept under a key that sign-out leaves, is a #3553 decision this page should not
  make; the file layout leaves room for either.
- **The send block applies at flush, not at queue time.** An entry queued before a key change is held, with the
  verification warning as its state, until the member is re-verified or the user discards it. That is a fourth
  not-sent reason in the table above.
- **A regenerated key version** is a rotation to the outbox: entries are re-encrypted under it before they go.
- **Per-message signatures**, if #3553 adds them, are computed when the entry is encrypted and recomputed on
  re-encryption. The entry gains a signature field, and the store's `v1` becomes `v2` with a migration that fails old
  entries visibly rather than sending them unsigned.
- **A sealed marker for a lost version** shows the same way for cached rows as for fetched ones.

## Test plan

Every scenario in #3481 maps to a test below; that issue stays the place they are listed, and this plan does not
duplicate its online lost-response case, which `TestChatDelivery_LostResponseRetry` and
`TestChatDelivery_RetryEncryptedAgainStoresOnce` already cover.

### Flutter

The harness is `test/support/fake_chat.dart`. It gains a controllable transport: `FakeChatMessages` and a fake
outbox post function that can be online, offline, accept-then-drop (store and throw), or answer a chosen status, and a
fake store over an in-memory file map and an in-memory secure storage. No sleeps: backoff takes an injectable clock.

- `test/services/chat_store_test.dart`: an envelope round trip; a file moved to another account, channel or kind
  fails to open and is deleted; a truncated file is deleted; deleting the key makes every file unreadable; the write
  is temp-and-rename (a failure between leaves the old file).
- `test/controllers/chat_outbox_controller_test.dart`:
  - Queue and flush: offline, send three; all queued; online; one post each, in order, each `clientMessageId` once.
  - Restart: queue two, dispose, build a new controller over the same store; both are queued again, none sending.
  - Partial failure: of three, the second answers 413; first and third are stored once, the second is failed, and a
    retry of it does not resend the others. A transient failure on the second holds the third.
  - Lost ack: accept-then-drop, then reconnect; one stored row, the entry is gone.
  - Rotation: queue under version 1, rotate to 2, flush; the post carries version 2 and the same `clientMessageId`.
  - Removal: a 404 fails the channel's entries with the removal copy and leaves other channels' entries alone.
  - 429 pauses every channel's queue; the flusher never exceeds its rate.
  - Expiry at 7 days, with the injectable clock.
- `test/controllers/chat_messages_controller_test.dart`: opening offline shows the cached rows; catch-up over more than
  the window drops it and loads the newest page; a tombstone is written at once; a row under a version with no grant is
  waiting and opens when the grant lands.
- `test/controllers/chat_controller_test.dart`: the three states and the outbox states pick the copy above; a queued
  message is not in `messageItems`, not in search results, and does not move the read marker; sign-out deletes the
  store and warns with the unsent count.
- `test/utils/error_text_test.dart` keeps guarding that no exception text reaches the UI.
- A narrow (360x640) and wide (1280x800) widget test for the queued, sending and failed rows, through the package's
  shared pump helper, in `packages/quark_widgets`.

### Go

The converge and delivery harnesses already model sessions, lossy sockets, disconnects and lost responses.

- `send_delivery_test.go`, new `TestChatDelivery_QueuedBurstInOrder`: a session posts five with client ids after a
  disconnect, losing two responses, then replays all five; five rows in post order, each heard once.
- `converge_clients_test.go`, new `TestChatConverge_OfflineQueue`: carol's second session queues three while
  disconnected and bob posts two; after reconnect the second session flushes; every session converges on the server
  page, carol's three after bob's two.
- If the stale-version hardening lands: a post under a superseded version is 409, and a repeated `clientMessageId`
  under it is still 200 with the stored row.
- Run both files with `DB_BACKEND=postgres` (`make test/db/postgres`) as well, since that is where commit order and id
  order can differ.

## Tickets, in landing order

Each is one reviewable PR. Paths are the ones each ticket owns; no two tickets share a file except where noted as
blocked by the earlier one.

1. **docs: describe the on-device chat store in chat-security.md.** The store section, the envelope construction, the
   wipe table, the two threat-model lines, and the web outbox. Lands first so review of the code has a written target.
   **Difficulty: mechanical (the design is this page).** Paths: `docs/chat-security.md`.
2. **feat: an encrypted on-device chat store.** `ChatStore` with the envelope, the key in secure storage, the file
   layout, temp-and-rename writes, wipe on sign-out and on host removal, and its tests. Not wired into chat yet.
   **Difficulty: hard (crypto construction and platform storage).** Paths: `lib/services/chat_store.dart`,
   `lib/services/chat_store_io.dart`, `lib/services/chat_store_web.dart`, `lib/utils/chat_store_config.dart`,
   `test/services/chat_store_test.dart`. Blocked by 1.
3. **feat: a durable chat outbox.** `ChatOutboxController`: entries, states, per-channel FIFO, hold-or-skip,
   backoff and pacing, rotation re-encryption, expiry; `ChatController` reads it for `pending` and `failedSends`, and
   the `StateError('left')` path goes. The fake transport in the test harness.
   **Difficulty: hard (state machine and ordering).** Paths: `lib/controllers/chat_outbox_controller.dart`,
   `lib/controllers/chat_controller.dart`, `test/controllers/chat_outbox_controller_test.dart`,
   `test/controllers/chat_controller_test.dart`, `test/support/fake_chat.dart`. Blocked by 2.
4. **test: chat offline sending, Go half.** `TestChatDelivery_QueuedBurstInOrder` and
   `TestChatConverge_OfflineQueue`, run on both backends. Closes the Go half of #3481. Can land any time; it needs no
   client code. **Difficulty: standard.** Paths: `internal/server/api/v0/chat/send_delivery_test.go`,
   `internal/server/api/v0/chat/converge_clients_test.go`.
5. **feat: queued, sending and failed message rows.** The package widget states, gallery entries, narrow and wide
   tests. **Difficulty: standard.** Paths: `packages/quark_widgets/lib/src/chat/quark_message_list.dart` and a new
   row-state part beside it, their test, `packages/quark_widgets/examples/widget_gallery/lib/registry.dart`, and the regenerated `docs.g.dart`.
   Independent of 2 and 3.
6. **feat: read cached chat history offline.** `ChatMessagesController` and `ChatChannelKeysController` read from and
   write to the store: the cached window, cached grants, immediate tombstone writes, eviction, the bounded catch-up
   replacing the whole-window re-read. **Difficulty: hard (catch-up correctness).** Paths:
   `lib/controllers/chat_messages_controller.dart`, `lib/controllers/chat_channel_keys_controller.dart`,
   `test/controllers/chat_messages_controller_test.dart`. Blocked by 2.
7. **feat: say which chat state the page is in.** The offline, locked, waiting and store-unavailable banners, the
   failed-row sentences as `Errors` constants, offline-disabled reactions and delete, the sign-out warning, and the
   Settings switch. **Difficulty: standard.** Paths: `lib/utils/error_text.dart`, `lib/pages/chat_page.dart`,
   `lib/widgets/chat/` (`chat_failed_send_bar.dart` and a new banner), `lib/pages/settings_page.dart` (the account tab's chat switch), their tests. Blocked by 3, 5
   and 6.
8. **feat: keep the web chat outbox across a reload.** The `sessionStorage` outbox, its 50-entry cap, the leave-page
   prompt. **Difficulty: standard.** Paths: `lib/services/chat_store_web.dart`, `lib/utils/leave_page_guard.dart`
   (with its `_io`/`_web` pair). Blocked by 3.
9. **fix (optional): refuse a chat post under a superseded key version.** `chatutil.PostMessage` answers 409 for a
   version below the current one unless the `clientMessageId` repeats a stored row; swagger; the outbox treats that
   409 as "re-encrypt and retry once". A contract change, so the backend PR is stacked with the client change.
   **Difficulty: standard (API contract change).** Paths: `pkg/util/chatutil/chatutil.go`,
   `internal/server/api/v0/chat/post_message.go`, `docs/swagger/`, a new case in
   `internal/server/api/v0/chat/messages_integration_test.go`; then the outbox's status table. Blocked by 3.
10. **docs: user journeys for offline chat.** New `JN-` entries for reading offline, sending offline and coming back.
    **Difficulty: mechanical.** Paths: `docs/user-journeys/chat.md`. Blocked by 7.

Option B of catch-up (a change sequence) is not a ticket here; it is filed only if the owner wants a larger window.

## Open questions for the owner

1. **Is offline reading on by default on native, and are 500 messages, 30 days and 50 MiB the right ceilings?** The
   page recommends on by default with a switch, since a family member who never finds the switch otherwise gets no
   offline reading at all. Off by default is the more conservative privacy stance.
2. **Should the server refuse a post under a superseded key version (ticket 9)?** Today it takes any version the
   channel has. The outbox re-encrypts on its own either way; the server check also covers a client that does not, at
   the cost of an API contract change.
3. **Background sending.** As proposed, the outbox flushes only while the app runs. Sending after the app is closed
   needs a new dependency (a background-task plugin such as `workmanager`, plus `BGTaskScheduler` on iOS) and,
   on iOS, the envelope key readable after first unlock. Is it wanted, and is a new dependency acceptable for it?
4. **Excluding the store from device backups.** Backups carry only ciphertext whose key does not travel, so they are
   harmless but useless. Excluding them needs a small iOS platform call and an Android backup rules file. Worth it?
5. **Does the outbox survive an app update while messages are queued?** Proposed: yes, the store is versioned and a
   format it cannot read fails its entries visibly. A key rotation re-encrypts. The epic asked; this is the answer
   unless the owner prefers failing queued entries on any update.
6. **How long may a message wait?** Proposed: 7 days, then failed with Retry.
7. **Carry the written-at time to other readers?** It would need a versioned plaintext format inside the ciphertext,
   which #3553's per-message signatures may define anyway. Proposed: not now; show it on the sender's device only.
8. **Queue reactions and deletes offline?** Proposed: no, disabled offline.
9. **Do #3553's pinned keys survive sign-out?** If they do, they cannot live in this store, which sign-out wipes.
