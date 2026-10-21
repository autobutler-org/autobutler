Status: Proposed

<!-- cspell:words fntdata fsType sfnt deobfuscate deobfuscated deobfuscation obfuscatedFont fontdata libeot subsetted CanvasKit skwasm FreeType Montserrat Lato subsetting odttf Skia Noto -->

The owner has not accepted this page. Nothing described here is built yet.

# Carrying a PowerPoint file's embedded fonts into the deck

Design for [#3657](https://github.com/autobutler-org/quark/issues/3657), the follow-up to
[#3632](https://github.com/autobutler-org/quark/issues/3632). A `.pptx` can carry the fonts its text is set in. Today
the import notices them and says so: "Embedded fonts (Montserrat and Lato) were not carried over, so text may look
different." This page decides which of those fonts Quark may keep, where they live, how the `.qslide` names them,
and how the editor draws text with them.

No code is written until the owner approves this page or sends it back. Each decision below gives a recommendation
and the alternatives it beat. Decisions marked **Owner call** are tradeoffs this page lays out and leaves open; the
implementation tickets start from the recommendation and take the owner's answer where it differs.

## What is fixed

These come from the issue, `AGENTS.md` and the code as it is today. The design works around them.

1. **A font's license is the font's, not the user's.** Each OpenType font says in its `OS/2` table (`fsType`) how
   it may be embedded. Quark honors that, and when it is unsure it keeps the font out and keeps the warning.
2. **Streaming and memory.** A `.pptx` is user-supplied and unbounded. A font part is read as a stream with a cap
   checked against what is actually read, never against what the zip declares.
3. **The file system first.** Fonts live on disk beside the deck, the way imported pictures do. No database table.
4. **The audience is not technical.** A warning names the font and says what the person will see, not what
   `fsType` is.
5. **Three platforms.** The editor runs on web, iOS and Android, so loading a font must work on all three with one
   code path.
6. **`quark_slides` never loads anything.** The package stores a reference and the host app resolves it, exactly as
   it does for pictures (`SlideImageBuilder`).

## What exists today

- **The warning.** `readPptx` in `pkg/util/pptxutil/import.go` reads `p:embeddedFontLst` from
  `ppt/presentation.xml` into `xPresentation.EmbeddedFonts` (`import_xml.go`), keeping only each entry's
  `p:font typeface`. It ignores the `p:regular`, `p:bold`, `p:italic` and `p:boldItalic` relationships and adds one
  deck-level `ImportWarning` built by `embeddedFontsWarning`. `TestImportWarnsThatEmbeddedFontsWereNotCarriedOver`
  covers it.
- **Runs already name their typeface.** `slideReader.typeface` (`import_text.go`) writes each run's `a:latin`
  typeface into `fontFamily`, resolving `+mj-` and `+mn-` to the theme's major and minor fonts. A run set in
  Montserrat already says `"fontFamily": "Montserrat"`. What is missing is a font registered under that name.
- **How pictures are stored.** `ImportPptxParams.StoreMedia` is a `StoreMediaFunc(part, name string, r io.Reader)`.
  `slideReader.storePicture` (`import_shapes.go`) checks the declared size against `MaxImageBytes` and
  `MaxMediaBytes`, opens the part through `pptxArchive.open`, whose `budgetReader` charges every byte against the
  part's cap and the package's `MaxImportBytes`, sniffs a 512-byte head, and streams the rest to `StoreMedia`.
  `fileutil.ImportPptxToQslide` (`pkg/util/fileutil/pptx_import.go`) implements it with `uploadutil.WriteFile` into
  a `<stem>_media` folder beside the `.qslide`, with `KeepBoth`. It records each path so `discardImport` removes
  them if the import fails.
- **The header is written before any slide.** `readPptx` decodes `presentation.xml` first, then
  `qslideWriter.header` writes `schemaVersion`, `title` and `size`, then the slides stream out. Fonts are listed in
  `presentation.xml`, so they can be stored and written into the header before the first slide.
- **The import writes schema version 1.** `importSchemaVersion = 1` (`pkg/util/pptxutil/qslide.go`). Fields that
  later versions added (transitions, tables, shadows) are written in that version 1 file, and the codec's
  migrations carry unknown root and slide fields through. `TestImportTypesMirrorTheCodec` holds the Go types to
  `packages/quark_slides/test/fixtures/v1_sample.qslide`.
- **The codec's compatibility rules.** `QslideCodec` (`packages/quark_slides/lib/src/format/qslide_codec.dart`)
  writes version 6. An additive field keeps the version and rides through `extra` in an older reader. A field whose
  absence would draw the deck wrong bumps the version, as shadows did for version 6.
- **Picture references.** `ImageSource.parse` (`packages/quark_slides/lib/src/model/image_source.dart`) reads a
  stored string as a Quark file path, `asset:<id>`, or `part:<entry>`. The read-only viewer's
  `fileutil.PreviewSlides` uses the same importer with a `StoreMedia` that discards the bytes and returns
  `part:ppt/media/...`, which `GET /previews/part` serves.
- **The editor has no font loading.** `SlideEditorController.defaultFontFamilies` is
  `['sans-serif', 'serif', 'monospace']`, the toolbar's family menu lists those, and `SlideTextLayout`
  (`packages/quark_slides/lib/src/canvas/slide_text_layout.dart`) passes `run.fontFamily` straight to a `TextStyle`.
  A family the platform lacks falls back to the default face. The app's `pubspec.yaml` bundles no fonts for slides.

## Summary of the proposal

| Question | Recommendation |
| --- | --- |
| Which fonts are kept | Only those whose `fsType` says **installable** (no embedding bits set) or **editable** (bit 3 alone). Preview & print, restricted, bitmap-only, several bits at once, or no readable `OS/2` table keeps the warning. |
| Which encodings are read | A plain TrueType or OpenType file, and one obfuscated as ECMA-376 describes. An Embedded OpenType (EOT) part is unwrapped only when it is uncompressed. A compressed one keeps the warning. |
| Streaming | Two forward passes over the part: one reads the header, the table directory and `fsType` within a 64 KiB bound, and one streams the font to storage. Nothing is buffered beyond a 32-byte key and that header. |
| Limits | 16 MiB a face, 64 MiB of fonts a deck, 32 faces a deck, all charged against `MaxImportBytes`. |
| Where they live | In the same `<stem>_media` folder as the pictures, named `<Family>-<Style>.ttf` or `.otf`, through a new `StoreFont` hook with `StoreMedia`'s signature. |
| How the `.qslide` names them | A new root `fonts` list of `{family, bold, italic, source}`, with `source` in the picture reference grammar. Additive: no version bump. |
| How the editor uses them | The app fetches each face and registers it with Flutter's `FontLoader` under a family name scoped to its bytes. The package maps a run's `fontFamily` to that scoped name when it lays text out. |
| What still warns | Every font not carried, by name, with a reason a person can read. |

## Decision 1: which embedding permissions allow Quark to keep a font

`fsType` is a 16-bit field in a font's `OS/2` table. Bits 0 to 3 set the embedding level, and two more bits narrow it:

| Value | Name | What the font's maker allows |
| --- | --- | --- |
| `0x0000` | Installable | The font may be installed on the receiving system and used for anything. |
| bit 1 (`0x0002`) | Restricted | The font may not be embedded or extracted at all. |
| bit 2 (`0x0004`) | Preview & print | The font may be loaded to view or print the document, read-only. |
| bit 3 (`0x0008`) | Editable | The font may be loaded to view and edit the document, then unloaded. |
| bit 8 (`0x0100`) | No subsetting | The font must be embedded whole. |
| bit 9 (`0x0200`) | Bitmap only | Only the font's bitmaps may be embedded, not its outlines. |

Quark keeps a font file beside a deck the user can edit, and loads it only for that deck. That is editing, so the
level the font allows has to cover editing.

### Options

| Option | For | Against |
| --- | --- | --- |
| **A. Installable or editable only** (recommended) | Matches what Quark does with the font: load it to view and edit this deck. Anything Quark is unsure about stays out. | A preview & print font is not shown even in the read-only viewer, where its license would allow it. |
| B. A, plus preview & print in the read-only viewer only | The viewer, which never saves, draws the deck as its author saw it. | The viewer would extract and serve a font the editor must refuse. Two policies for one font, and a bug in the line between them is a license breach. |
| C. Everything but restricted | Most decks would look right. | Preview & print fonts end up stored in a folder the user can copy and edit with, which that level forbids. |
| D. Ignore `fsType` and keep every font | Simplest. | Quark would hand out fonts their makers said must not be extracted. Not acceptable. |

**Recommended: A.** The check, in order:

1. The font has an `OS/2` table that can be read within the bounds below. If not, keep the warning.
2. Bit 9 (bitmap only) is clear. Quark draws outlines, so a bitmap-only font is refused.
3. The low four bits are exactly `0x0` or exactly `0x8`. Restricted (`0x2`), preview & print (`0x4`), and any
   combination of bits are refused.
4. Bit 8 (no subsetting) needs no check. Quark stores the font as it came and never subsets it.
5. An EOT part carries its own `fsType` copy in the EOT header. Both that copy and the inner font's must pass.

Step 3 is stricter than the OpenType specification. For `OS/2` versions 0 to 2, the specification says the least
restrictive bit set wins. Quark refuses instead, because a font that sets two levels is unclear, and the issue says
unclear means warn.

A font that passes is loaded only for the deck it came with. It is never added to the app's font list for other
decks, and never offered as a download on its own. The editor does list it in that deck's family menu, since editing
with the font is what both levels allow.

## Decision 2: reading the font parts

### What a part can hold

`p:embeddedFont` names up to four faces, `p:regular`, `p:bold`, `p:italic` and `p:boldItalic`. Each `r:id` resolves
through `ppt/_rels/presentation.xml.rels` to a part such as `ppt/fonts/font1.fntdata`. The extension says nothing
about the encoding. The part's content type in `[Content_Types].xml` and its first bytes do:

| Encoding | How it is recognized | What Quark does |
| --- | --- | --- |
| Plain TrueType or OpenType | Starts with `0x00010000`, `true` or `OTTO` | Reads it as it is. |
| Obfuscated (`application/vnd.openxmlformats-officedocument.obfuscatedFont`) | The content type, and the GUID in the part's file name | XORs the first 32 bytes with the key from the GUID, then reads it as plain. |
| Uncompressed Embedded OpenType (`application/x-fontdata`) | An EOT header whose `Flags` have no compression bit | Skips the EOT header, undoes its XOR flag (`0x50` on every byte) if set, and reads the font inside. |
| Compressed Embedded OpenType (MicroType Express) | The compression bit in `Flags` | Keeps the warning. Decompressing MicroType Express is a codec of its own (LibreOffice links `libeot` for it), and this page does not take one on. |
| Anything else, or a `.ttc` collection | | Keeps the warning. |

The obfuscation key is the GUID in the part's name (`{…}.fntdata` or `{…}.odttf`), written as 16 bytes and
reversed. Bytes 0 to 15 and 16 to 31 of the part are each XORed with it. This is the scheme the issue cites from
ECMA-376. The first implementation ticket confirms it, and the table above, against real files from PowerPoint for
Windows and Mac, Keynote, Google Slides and LibreOffice before it writes the reader. Which of these each one writes
is not settled, and the table changes if the files disagree.

### Streaming

The license check needs the `OS/2` table, which the font's table directory places anywhere in the file. The font must
not be stored before the check passes, and must not be held in memory whole.

| Option | For | Against |
| --- | --- | --- |
| **A. Two forward passes over the part** (recommended) | Nothing refused ever reaches the disk. Memory is the 32-byte key and a header bounded at 64 KiB. Pass 1 stops as soon as it has `fsType`. | The part is decompressed twice, up to the `OS/2` table on the first pass and whole on the second. Both passes are charged to the import's budget. |
| B. Buffer the font in memory up to its cap, then check and store | One pass. `golang.org/x/image/font/sfnt` can parse the whole font from a `bytes.Reader`. | 16 MiB of heap per face on a device with little memory, which `AGENTS.md` asks us to avoid when streaming will do. |
| C. Store first, reopen through `vfs` as a `ReaderAt`, check, delete when refused | One decompression, full random access. | A font the license refuses is written to the user's files, announced or not, and then removed. A crash between leaves it there. |

**Recommended: A.**

- **Pass 1.** Open the part with `pptxArchive.open` and a 16 MiB cap, and wrap it in a reader that undoes the
  obfuscation or the EOT wrapper. Read the 12-byte sfnt header and its table directory, at most 64 tables
  (`12 + 64 × 16` bytes, well under the 64 KiB bound). Refuse a directory whose records point outside the font's
  declared length or overlap. Find `OS/2`, discard forward to its offset, read its first 10 bytes, and take `fsType`
  at offset 8. Close the part. An `OS/2` table that starts inside the bytes already read is read from them; one that
  overlaps the directory is refused.
- **Pass 2**, only if the check passed. Open the part again with the same cap and wrapper, and stream it to
  `StoreFont`. A face whose second pass reads differently from the first (a different length, a different directory)
  fails as `ErrNotPptx`, the file's fault.
- **Faces in a typeface.** Each face is checked on its own. A typeface whose regular face passes and whose bold
  face does not is carried in regular, and the warning names the bold face.

### Limits

New constants beside the import limits in `pkg/util/pptxutil/pptxutil.go`:

| Limit | Value | Past it |
| --- | --- | --- |
| `MaxFontBytes`, one face as read | 16 MiB | The face is left out, and the warning says it was too large. |
| `MaxFontsBytes`, every face in the deck together | 64 MiB | The faces after it are left out, and the warning names them. |
| `MaxEmbeddedFaces`, the faces a deck lists | 32 (8 typefaces in 4 styles) | The faces after the 32nd are not read, and the warning names them. |
| The sfnt header and table directory | 64 KiB, at most 64 tables | The face is refused as unreadable. |

Every byte read in either pass is charged to `MaxImportBytes` through `budgetReader`, as pictures are. 16 MiB covers
any Latin face whole and most CJK faces. A larger one is rare in a slide deck and would cost the editor that much to
load on a phone.

### Untrusted fonts reach the client's parser

A font that passes is handed to Skia and FreeType on every platform, since Flutter's web renderers (CanvasKit and
skwasm) do not go through the browser's own font sanitizer. A malformed font is an attack on that parser. The
server's checks above (the magic number, a bounded directory whose records stay inside the font) turn away a part
that is not a font at all, but they are not a sanitizer. What narrows the risk is who sees the font: only a person who can
open the deck, which someone in the household imported from a file they chose. A sanitizer on the server is left as
an open question.

## Decision 3: where the fonts live and how the `.qslide` names them

### Storage

| Option | For | Against |
| --- | --- | --- |
| **A. In the deck's `<stem>_media` folder, beside its pictures** (recommended) | One folder holds everything the deck needs, so copying, moving or deleting the deck together with its folder takes the fonts along. `discardImport` already cleans the folder up after a failure. | A user browsing the folder sees `.ttf` files next to the pictures. |
| B. A `<stem>_fonts` folder beside the deck | Pictures and fonts are kept apart. | A second folder to create, clean up after a failure, and keep in step when the deck moves. |
| C. Inline in the `.qslide`, as base64 | One file. | A 1 MiB font is 1.3 MB of JSON that the editor parses, holds and rewrites on every autosave. |
| D. A cache on the Quark keyed by the font's hash, outside the user's files | Two decks with the same font share one copy. | Not beside the deck. It outlives the deck, needs garbage collection, and copying the deck to another Quark loses its fonts. |

**Recommended: A.** Faces are named `<Family>-<Style>.<ext>` (`Montserrat-Regular.ttf`, `Lato-BoldItalic.ttf`), the
family stripped of anything a file name cannot hold, and the extension `ttf` or `otf` by the sfnt magic. Files are
written with `KeepBoth`, so a second import beside the first gets `Montserrat-Regular_(1).ttf`, as pictures do.
What is stored is always the plain font, never the obfuscated or EOT form, so any font tool can open it and the
license check never has to run again.

### The hook

`ImportPptxParams` gains `StoreFont StoreMediaFunc`. It takes `StoreMedia`'s signature, `(part, name, r)` and the
reference back, and `StoreMediaFunc`'s doc grows to say it stores fonts too. A separate field rather than reusing
`StoreMedia` keeps each caller's choice explicit: nil keeps every font out, with the warning as it reads today.

- `fileutil.ImportPptxToQslide` passes the same closure for both, so fonts land in the media folder and join
  `media` for `discardImport`. `ImportPptxResult` gains `Fonts int` in both packages, and
  `POST /api/v0/files/import/pptx`'s response gains `fonts` beside `pictures`. That is an additive change to the
  response, which the frontend ticket picks up.
- `fileutil.PreviewSlides` passes nil until the read-only viewer gets its own ticket (see the open questions).

### The schema

The `.qslide` root gains a `fonts` list:

```json
{"schemaVersion": 6, "title": "Talk",
 "size": {"width": 1920, "height": 1080},
 "fonts": [
   {"family": "Montserrat", "source": "Talk_media/Montserrat-Regular.ttf"},
   {"family": "Montserrat", "bold": true, "source": "Talk_media/Montserrat-Bold.ttf"}
 ],
 "slides": [...]}
```

- **A `DeckFont` model** in `packages/quark_slides/lib/src/model/deck_font.dart`: `family`, `bold` and `italic`
  (left out when false, as the codec leaves out defaults), `source`, and `extra`. `Presentation` gains
  `List<DeckFont> fonts`, empty by default and left out of the JSON when empty.
- **`source` uses the picture grammar.** A Quark file path, `asset:<id>` or `part:<entry>`, read with
  `ImageSource.parse`. The name `ImageSource` reads oddly for a font. Renaming it to something like `MediaSource`
  touches every picture call site, so it is left for its own change.
- **Additive, no version bump.** An older editor keeps `fonts` in `Presentation.extra`, writes it back on save, and
  draws the deck in fallback fonts, which is what every editor does with this deck today. Shadows bumped the version
  because an older reader drew an element wrong. Missing fonts draw it the way it already looks, and refusing to open
  a deck over its fonts would be worse than that. It also keeps `pptxutil`'s exporter, which reads up to version 6,
  able to export a deck that has fonts.
- **The importer writes it in the header.** Fonts are stored before `qslideWriter.header` runs, so the header gains a
  `fonts` key after `size`, in the order `QslideCodec.encode` writes it. `outPresentation` gains the matching field,
  and `v1_sample.qslide` gains a `fonts` entry, so `TestImportTypesMirrorTheCodec` holds the two writers together.
- **Order and duplicates.** One entry per family and style. A second entry for the same family and style wins
  nothing: the codec keeps the first and drops the rest on read.

**Owner call: the version.** Additive (recommended) or version 7? The case for 7 is the shadows precedent: a reader
that does not know a field the deck relies on asks to be updated. The case against is above.

## Decision 4: loading the fonts in the editor

Flutter registers a font at run time with `FontLoader(family)..addFont(bytes)..load()`. It works the same on web,
iOS and Android, so the app needs no conditional import. It has two properties that shape the design:

- **Registration is global and permanent.** A family registered once is visible to every `TextStyle` in the app
  until it quits. A deck that registers `Montserrat` would change how a second deck that only names `Montserrat`
  draws.
- **PowerPoint often embeds subsets.** Its "embed only the characters used" option, on by default, stores a face
  with only the glyphs the deck uses. Registered as plain `Montserrat`, a subset from one deck would drop letters from
  another.

### Family names

| Option | For | Against |
| --- | --- | --- |
| **A. Register under a name scoped to the face's bytes, and map to it when laying text out** (recommended) | Two decks never share a face unless the bytes are identical, and identical bytes are registered once. The run keeps `fontFamily: "Montserrat"`, so the toolbar, export and an older editor all see the real name. | `quark_slides` gains a family map on its text layout. |
| B. Register under the plain family name | No package change. | The subset problem above, and the second deck to register a family loses. |
| C. Rewrite each run's `fontFamily` to the scoped name at import | No mapping at draw time. | The file now names `Montserrat#a1b2c3`, which the toolbar shows, export writes into PowerPoint, and an older editor cannot draw at all. |

**Recommended: A.**

- **In the package.** `SlideCanvas` and its style take an optional `Map<String, String> fontFamilies`, a family as the
  file names it to the family to draw it in. `SlideTextLayout` maps `run.fontFamily` and the theme's heading and body
  fonts through it, and leaves a name not in the map as it is. The package never fetches a font, in keeping with
  `SlideImageBuilder`.
- **In the app.** A `SlideFontLoader` in `lib/services/` takes a deck's `fonts` and a fetch function (the controller's
  `mediaUrl`, so it reads through the same download route and token as pictures). For each family it fetches every
  face, hashes the bytes, and registers them together under `<family>#<first 12 hex digits of the SHA-256>`. A
  family already registered with that hash is not loaded again. It returns the map.
- **When the deck appears.** `SlideEditorController.load` starts loading fonts beside the deck and waits for them up
  to 2 seconds before it shows the canvas, so most decks first appear in their own fonts. Fonts that arrive later
  update the map, and text relays out when they do. A face that fails to fetch or register is skipped, and the deck
  shows an inline notice naming the family, built through `Errors`.
- **The family menu.** The deck's own families come after the defaults in the toolbar menu, for that deck only.
  A subsetted face lacks glyphs the deck did not use, so a typed character can fall back to the default face. The
  menu says nothing about that. It is how PowerPoint behaves with the same file.
- **Presenting and thumbnails** use the same map, so a deck looks the same in the editor, the slide strip and
  presentation mode.
- **Memory.** A registered font stays in memory until the app quits, up to the 64 MiB a deck may carry. On a phone
  that opens many font-heavy decks in one session, that adds up. Flutter offers no way to unregister a font, so this
  is a known cost, not a bug to fix here.

## Warnings

The deck-level warning keeps its place and names only the fonts that were not carried. Each reason gets one
sentence, grouped so a deck with three refused fonts reads as one line:

| Reason | Message |
| --- | --- |
| The license does not allow it | "The fonts Montserrat and Lato don't allow copying, so text set in them may look different." |
| Compressed or unknown encoding, unreadable `OS/2` | "The font Montserrat couldn't be read, so text set in it may look different." |
| Past `MaxFontBytes` or `MaxFontsBytes` | "The font Noto Sans CJK is too large to bring in, so text set in it may look different." |
| Fonts not carried here at all (`StoreFont` is nil) | The warning as it reads today. |

A typeface carried in some styles and not others is named with its missing style: "Lato Bold".

## Implementation tickets

Each is one reviewable PR. Tickets 1 and 2 can start at the same time, and 2 lands after 1. The backend tickets sit below the frontend ones
that consume them, per `AGENTS.md`'s stacking rule.

1. **The schema: `DeckFont` and `Presentation.fonts`.** The model, `fromJson` and `toJson`, the codec's doc on the
   new field, a `fonts` entry in `v1_sample.qslide` and the current golden fixture, a round-trip test, a test that an
   unknown key on a font survives, and a test that a duplicate family and style keeps the first. Additive at version
   6 unless the owner picks 7.
   **Difficulty: standard (a model and codec change on an established pattern).**
   Claims: `packages/quark_slides/lib/src/model/deck_font.dart`, `packages/quark_slides/lib/src/model/presentation.dart`,
   `packages/quark_slides/lib/src/format/qslide_codec.dart`, `packages/quark_slides/lib/quark_slides.dart`,
   `packages/quark_slides/test/fixtures/**`, `packages/quark_slides/test/format/**`,
   `packages/quark_slides/test/model/deck_font_test.dart`.
2. **Reading and checking font parts in `pptxutil`.** First, a short survey: import one deck from each of
   PowerPoint for Windows and Mac, Keynote, Google Slides and LibreOffice, record what each writes, and correct
   [Decision 2](#decision-2-reading-the-font-parts) in this page if it is wrong. Then the face relationships on
   `xPresentation`, the obfuscation and uncompressed EOT readers, the two-pass check, the limits, `StoreFont`, the
   `fonts` header key and `outPresentation` field, `ImportPptxResult.Fonts`, and the new warnings. Tests build
   their fonts in Go (a minimal sfnt with an `OS/2` table), so no font file with a license of its own is committed.
   They cover installable and editable kept, preview & print, restricted, bitmap-only and two-bit fonts refused,
   an obfuscated and an EOT face kept, a compressed EOT refused, a face past each limit, a directory pointing past
   the end, a missing `OS/2`, and a heap test showing a part that declares 1 GiB fails without allocating it.
   Includes a fuzz target for the sfnt directory reader. Depends on 1 for the fixture.
   **Difficulty: hard (a parser of untrusted input, a license boundary, and a format whose real-world encodings need
   confirming).**
   Claims: `pkg/util/pptxutil/import_font*.go`, `pkg/util/pptxutil/import.go`, `pkg/util/pptxutil/import_xml.go`,
   `pkg/util/pptxutil/import_qslide.go`, `pkg/util/pptxutil/pptxutil.go`, `pkg/util/pptxutil/import_effects_test.go`,
   `pkg/util/pptxutil/import_internal_test.go`.
3. **Storing fonts beside the deck.** `StoreFont` in `fileutil.ImportPptxToQslide`, fonts in `media` so a failed
   import removes them, `Fonts` on the result, `fonts` on the import response with its swagger, and an integration
   test that a permitted font lands in `<stem>_media` and a refused one does not. Depends on 2.
   **Difficulty: mechanical (one closure and one response field, specified here).**
   Claims: `pkg/util/fileutil/pptx_import.go`, `pkg/util/fileutil/pptx_import_test.go`,
   `internal/server/api/v0/files/import_pptx.go`, `internal/server/api/v0/files/types.go`, `docs/swagger/**`.
4. **The family map in `quark_slides`.** `fontFamilies` on `SlideCanvas` and its style, the mapping in
   `SlideTextLayout` for runs and theme fonts, and passing it through thumbnails and presentation. Widget tests at
   narrow and wide sizes that a mapped family reaches the `TextStyle` and an unmapped one is left alone. Depends on 1.
   **Difficulty: standard (a new input threaded through the canvas).**
   Claims: `packages/quark_slides/lib/src/canvas/slide_text_layout.dart`,
   `packages/quark_slides/lib/src/canvas/slide_canvas.dart`, `packages/quark_slides/lib/src/canvas/slide_canvas_style.dart`,
   `packages/quark_slides/test/canvas/**`.
5. **Loading the fonts in the app.** `SlideFontLoader` with an injected fetch, the scoped family names, the wait of
   up to 2 seconds in `SlideEditorController.load`, the map passed to the canvas, the deck's families in the toolbar
   menu, the inline notice through `Errors`, and the import result's font count in the import summary. Tests: the
   controller registers a font from a fake fetch (`FontLoader` runs in `flutter_test`), two decks with different bytes
   for one family get two names, the same bytes are registered once, and a failed fetch shows the notice. Depends on
   3 and 4.
   **Difficulty: standard (a service and controller change on the existing pattern).**
   Claims: `lib/services/slide_font_loader.dart`, `lib/controllers/slide_editor_controller.dart`,
   `lib/widgets/slides/**`, `lib/utils/error_text.dart`, `test/services/slide_font_loader_test.dart`,
   `test/controllers/slide_editor_controller_test.dart`.
6. **Docs and journeys.** `JN-SL-028` (Import a PowerPoint file) gains the fonts and their warnings, a slides
   section in `docs/architecture/` records the license rule and the storage, and this page's decisions that stand
   move there. Folded into each layer as it lands, closed by the last one.
   **Difficulty: mechanical (writing down what the other tickets built).**
   Claims: `docs/user-journeys/slides.md`, `docs/architecture/read-only-documents.md`, `docs/design/pptx-embedded-fonts.md`.

## Open questions for the owner

1. **The license line.** Installable and editable only (recommended), or also preview & print in the read-only viewer?
2. **The schema version.** `fonts` as an additive field at version 6 (recommended), or version 7, so an older editor
   asks to be updated?
3. **Compressed EOT.** Keep the warning for MicroType Express fonts (recommended), or take on a decompressor? It is
   either a cgo dependency on `libeot` or a Go port of its codec, and either one needs the owner's approval.
4. **The read-only viewer.** Leave it in fallback fonts (recommended for now), or add a ticket for a
   `GET /previews/font` that runs the same check and deobfuscates on every request, since the viewer stores nothing?
5. **A font sanitizer.** Is the server's structural check enough for fonts that reach the client's parser, or should
   a full sanitizer (a Go port of OpenType Sanitizer's rules, or a new dependency) run before a font is stored?
6. **Export.** Should exporting to `.pptx` embed a deck's carried fonts again? Both permitted levels allow it. It is
   out of scope here and would be a follow-up ticket.
