// Package ai provides pluggable solo opponents for the local simulator.
//
// Agents never import engine mutators. They read view.MatchView (same fog as a
// human) and return Action values; Driver applies those via session.PlayerSession.
//
// Captain roadmap (persona packs, same Agent interface):
//   - Ignus: Tsubasa (Passion Wing) → Saaya
//   - Other seven elements: one captain each
//   - Archetypes: TENGU, Dream 7 / Idol
package ai
