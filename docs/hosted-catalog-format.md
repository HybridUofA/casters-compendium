# Hosted catalog format

The hosted catalog is a public, read-only distribution format for approved
Caster's Compendium card data, artwork, and Tabletop Simulator assets. R2 is
storage rather than an editing database: source data is normalized and
validated before publication.

## Publication layout

```text
backs/
  mtd-back-v1.png
catalog/
  current.json
  v1/
    release.json
    cards.json
    images/
      <stable-card-id>.png
    tts/
      manifest.json
      sheet-001.png
```

Every directory below `catalog/vN/` is immutable. Corrected or expanded data
must be published under a new version. `catalog/current.json` is the only
mutable object and is uploaded last.

## Current pointer

```json
{
  "schemaVersion": 1,
  "catalogVersion": "v1",
  "releaseURL": "https://tts.casterscompendium.com/catalog/v1/release.json"
}
```

Clients reject unsupported schemas, blank versions, non-HTTPS URLs, and a
pointer whose version differs from the referenced release.

## Release manifest

`release.json` records:

- Schema and catalog versions.
- UTC publication time.
- Canonical `cards.json` URL, byte size, and SHA-256.
- Stable per-card image base URL and deterministic collection SHA-256.
- TTS manifest and card-back URLs.

Clients download `cards.json` with a fixed size limit, compare its exact size
and SHA-256, decode the normalized records, and validate all card IDs before
installation.

## Tabletop Simulator manifest

The TTS manifest contains canonical sheets and a mapping keyed by stable
internal card ID. Each sheet specifies its public face URL, deck key,
dimensions, and populated card count. Each card mapping specifies a deck key
and zero-based slot.

Tabletop Simulator's physical card ID is:

```text
DeckID = DeckKey * 100 + Slot
```

Deck keys are unique positive integers no greater than 99. A sheet contains at
most 70 cards and every mapping must reference an existing populated slot.

## Building

Generate a release locally without uploading:

```sh
go run ./cmd/tools/catalogbuild \
  -version v1 \
  -base-url https://tts.casterscompendium.com
```

Output is written under the ignored `dist/hosted-catalog/` directory. The tool
refuses to replace an existing version directory.

## Publishing

The `Publish hosted card catalog` workflow requires:

- GitHub environment: `catalog-production`
- Secrets: `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`
- Variables: `R2_ACCOUNT_ID`, `R2_TTS_BUCKET`

The R2 token should have Object Read & Write permission for only the TTS asset
bucket. It does not need account administration permission. The workflow:

1. Tests and generates the complete release.
2. Refuses to replace a remotely existing version.
3. Uploads immutable objects with a one-year cache lifetime.
4. Verifies the release and TTS manifests through the public hostname.
5. Uploads `current.json` with a five-minute cache lifetime.
6. Verifies the newly active pointer through a cache-busting request.

Never place R2 credentials in the repository, desktop application, catalog
files, saved TTS objects, or support logs.
