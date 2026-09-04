# quark_widgets

Quark's reusable Flutter widgets and design tokens. The app keeps its pages,
controllers, and services; everything it renders that is none of those lives
here. The package depends on Flutter and `quark_icons` and nothing else — no
services, no router, no HTTP — so a widget test starts in milliseconds and the
gallery can render every widget with fake data.

## The rules

Data in, callbacks out: a widget takes immutable values and handlers, never
fetches, never navigates, and never reads global state, so selection, loading,
and error live in the caller. Colors, radii, and spacing come from `QuarkTokens`
through the theme, never a hardcoded value — the gallery's theme panel edits the
tokens live, which turns a hardcoded color into something you can see. Every
widget ships as a set: a file in `lib/src/<group>/`, an export from the barrel,
`///` docs on the class, a test with a 360x640 and a 1280x800 case, and a
gallery entry.

The full contract is the **Widget package rules** section of
[`AGENTS.md`](../../AGENTS.md#widget-package-rules-packagesquark_widgets-always-follow-this),
and it is what reviewers hold changes to.

## The gallery

`examples/widget_gallery` renders every widget next to its documentation over a
theme you can edit while it runs: light/dark, a hex field per color token, and a
slider per radius and spacing step. Callbacks land in the event panel at the
bottom, so a callback that never fires is visible.

```sh
make -C packages/quark_widgets/examples/widget_gallery watch   # Chrome, hot reload
make -C packages/quark_widgets/examples/widget_gallery serve   # local web server
make -C packages/quark_widgets/examples/widget_gallery build   # static web build
```

## Adding a widget

1. **File.** `lib/src/<group>/<name>.dart`, one public class, `super.key`, and a
   deterministic `ValueKey` on every tappable part.
2. **Export.** Add it to `lib/quark_widgets.dart`.
3. **Docs.** A `///` block on the class: what it is for, the `ValueKey` prefixes
   it emits, and one usage snippet. Document every parameter and callback.
4. **Test.** `test/<group>/<name>_test.dart`, with a narrow (360x640) and a wide
   (1280x800) case.
5. **Registry.** An entry in `examples/widget_gallery/lib/registry.dart` with
   fake data, wiring every callback to the `log` function it is handed.
   `test/gallery_registry_test.dart` fails on an export with no entry.
6. **Regenerate.** `make -C packages/quark_widgets generate/docs`, which rewrites
   `examples/widget_gallery/lib/docs.g.dart` from the `///` blocks. The same test
   fails when it is stale.

```sh
make -C packages/quark_widgets test/unit
make -C packages/quark_widgets check
```
