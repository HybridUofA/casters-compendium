# Website screenshot checklist

Use real application captures rather than mockups so the website always matches
the shipped interface. PNG is preferred.

Current captures:

1. `casters-compendium-deck-editor.png` — the primary editor view used in the
   README and landing page.
2. `casters-compendium-main-menu.png` — the complete default-theme main menu.
3. `casters-compendium-multi-selection.png` — several individually selected
   card copies in a populated deck.
4. `casters-compendium-deck-export-text.png` — the Speedrobo-compatible text
   decklist beside its corresponding editor state.

Recommended remaining capture:

- `casters-compendium-tts-export.png` — Tabletop Simulator showing the exported
  deck fully loaded from the hosted sheets. This demonstrates the project's
  strongest integration and should be captured after the custom-domain
  certificate is active.

Before capturing:

- Use a sample deck that is safe to publish.
- Hide personal filenames, usernames, filesystem paths, and notifications.
- Capture at 100% interface scale when possible, preferably at 1600×900 or
  1920×1080.
- Keep the application window aspect ratio consistent across guide images.
- Take both a clean full-window image and any close crop that may be useful.
- Use lossless PNGs and avoid artificial frames, device mockups, or retouching
  that changes application behavior.

When the images are ready, place them in this directory. The Pages workflow
automatically publishes this folder, so the landing and guide markup can then
reference them from `screenshots/<filename>`.
