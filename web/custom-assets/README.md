# Player custom-asset service

This Cloudflare Worker accepts player-provided card backs and simulator card
art. It validates Turnstile server-side, rate limits uploads, fully decodes and
re-encodes images through Cloudflare Images, and stores only the normalized PNG
in a dedicated R2 bucket. The official hosted catalog bucket is not used.

Objects are content-addressed and intentionally retained: deleting an object
would break existing Tabletop Simulator saves and multiplayer simulator data.
Objects are not listed by the service. The private GitHub security-report form
is the initial abuse and takedown contact.

## One-time Cloudflare setup

1. Create the R2 Standard bucket `casters-compendium-user-assets`.
2. Enable Images transformations for the account. The free plan currently
   includes 5,000 unique transformations per calendar month.
3. Create a Turnstile widget restricted to
   `custom-assets.casterscompendium.com`.
4. Create a scoped Cloudflare API token that can deploy this Worker and bind
   only the custom-assets bucket.
5. Add GitHub environment `custom-assets-production` with:
   - secret `CLOUDFLARE_API_TOKEN`
   - secret `CUSTOM_ASSETS_TURNSTILE_SECRET_KEY`
   - repository variable `R2_ACCOUNT_ID` (already used by catalog publishing)
   - variable `CUSTOM_ASSETS_TURNSTILE_SITE_KEY`
6. Keep the committed R2 bucket name, Worker route, expected hostname, and
   public origin aligned if any resource uses a different name.
7. No paid WAF rule is required. The Worker rejects missing, invalid, and
   oversized `Content-Length` values before parsing multipart data, then checks
   the extracted image size again. This deliberately rejects chunked uploads in
   exchange for bounded parsing on plans without Enterprise body-size fields.

The deployment workflow tests the Worker before publishing it. The Turnstile
secret is installed as a Worker secret and is never placed in a static page,
desktop binary, log, or repository variable.

## Local checks

```sh
npm ci
npm test
npx wrangler dev
```

Wrangler's local Images binding provides enough fidelity for resize and output
format development. Use `wrangler dev --remote` only when a deliberate remote
integration test is required.
