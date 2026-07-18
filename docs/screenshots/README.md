# Website screenshot checklist

Use real application captures rather than mockups so the website always matches
the shipped interface. PNG is preferred.

Recommended captures:

1. `landing-editor.png` — a 1600×1000 or similarly wide dark-mode editor view
   with a representative deck, search results, and a card preview. This is the
   best candidate for the landing page.
2. `guide-main-menu.png` — the full main menu without open file dialogs.
3. `guide-search.png` — the editor showing several active filters and a hovered
   card preview.
4. `guide-deck-editing.png` — the editor with populated main and side decks.
5. `guide-export.png` — a generated Tabletop Simulator sheet or the export
   controls, with personal filesystem paths cropped out.

Before capturing:

- Use a sample deck that is safe to publish.
- Hide personal filenames, usernames, filesystem paths, and notifications.
- Capture at 100% interface scale when possible.
- Keep the application window aspect ratio consistent across guide images.
- Take both a clean full-window image and any close crop that may be useful.

When the images are ready, place them in this directory. The Pages workflow
automatically publishes this folder, so the landing and guide markup can then
reference them from `screenshots/<filename>`.
