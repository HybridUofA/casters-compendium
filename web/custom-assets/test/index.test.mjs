import assert from "node:assert/strict";
import { webcrypto } from "node:crypto";
import test from "node:test";

globalThis.crypto ??= webcrypto;

const { createHandler, MAX_UPLOAD_BYTES } = await import("../src/index.mjs");

const origin = "https://custom-assets.casterscompendium.com";

function testEnvironment(overrides = {}) {
  const objects = new Map();
  const bucket = {
    async head(key) { return objects.has(key) ? {} : null; },
    async put(key, bytes, options) { objects.set(key, { bytes, options }); },
    async get(key) {
      const value = objects.get(key);
      if (!value) return null;
      return {
        body: value.bytes,
        httpEtag: '"test-etag"',
        writeHttpMetadata(headers) { headers.set("Content-Type", value.options.httpMetadata.contentType); },
      };
    },
    objects,
  };
  const imageOutput = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]).buffer;
  const images = {
    async info() { return { width: 700, height: 1000, format: "image/jpeg" }; },
    input() {
      return {
        transform() { return this; },
        async output() {
          return { response: () => new Response(imageOutput, { headers: { "Content-Type": "image/png" } }) };
        },
      };
    },
  };
  return {
    PUBLIC_ORIGIN: origin,
    TURNSTILE_SITE_KEY: "test-site-key",
    TURNSTILE_SECRET_KEY: "test-secret",
    TURNSTILE_EXPECTED_HOSTNAME: "custom-assets.casterscompendium.com",
    UPLOAD_RATE_LIMITER: { async limit() { return { success: true }; } },
    CUSTOM_ASSETS: bucket,
    IMAGES: images,
    ...overrides,
  };
}

function uploadRequest(fields = {}) {
  const form = new FormData();
  form.set("assetKind", fields.kind || "tts-card-back");
  form.set("rightsConfirmed", fields.rightsConfirmed ?? "yes");
  form.set("cf-turnstile-response", fields.token || "valid-token");
  form.set("image", fields.file || new Blob([new Uint8Array([1, 2, 3])], { type: "image/jpeg" }), "back.jpg");
  return new Request(`${origin}/api/v1/assets`, {
    method: "POST",
    headers: {
      Origin: origin,
      "CF-Connecting-IP": "192.0.2.10",
      "Content-Length": String((fields.file?.size || 3) + 2048),
    },
    body: form,
  });
}

const passingTurnstile = async () => new Response(JSON.stringify({
  success: true,
  hostname: "custom-assets.casterscompendium.com",
  action: "custom-asset-upload",
}));

test("serves an upload page containing the configured Turnstile key", async () => {
  const response = await createHandler().fetch(new Request(`${origin}/`), testEnvironment());
  assert.equal(response.status, 200);
  assert.match(await response.text(), /test-site-key/);
  assert.match(response.headers.get("Content-Security-Policy"), /challenges\.cloudflare\.com/);
  assert.doesNotMatch(response.headers.get("Content-Security-Policy"), /unsafe-inline/);
});

test("rejects uploads from another origin", async () => {
  const request = uploadRequest();
  request.headers.set("Origin", "https://attacker.example");
  const response = await createHandler({ fetchImpl: passingTurnstile }).fetch(request, testEnvironment());
  assert.equal(response.status, 403);
});

test("rejects uploads without a declared length before parsing multipart data", async () => {
  const request = uploadRequest();
  request.headers.delete("Content-Length");
  const response = await createHandler({ fetchImpl: passingTurnstile }).fetch(request, testEnvironment());
  assert.equal(response.status, 411);
});

test("requires successful server-side Turnstile validation", async () => {
  const fetchImpl = async () => new Response(JSON.stringify({ success: false }));
  const response = await createHandler({ fetchImpl }).fetch(uploadRequest(), testEnvironment());
  assert.equal(response.status, 403);
  assert.match((await response.json()).error, /verification failed/i);
});

test("rejects an upload that exceeds the byte limit", async () => {
  const oversized = new Blob([new Uint8Array(MAX_UPLOAD_BYTES + 1)], { type: "image/png" });
  const response = await createHandler({ fetchImpl: passingTurnstile }).fetch(
    uploadRequest({ file: oversized }), testEnvironment(),
  );
  assert.equal(response.status, 413);
});

test("rejects non-card-shaped TTS backs", async () => {
  const env = testEnvironment();
  env.IMAGES.info = async () => ({ width: 1000, height: 500, format: "png" });
  const response = await createHandler({ fetchImpl: passingTurnstile }).fetch(uploadRequest(), env);
  assert.equal(response.status, 400);
  assert.match((await response.json()).error, /portrait image/i);
});

test("rejects excessive decoded dimensions before transformation", async () => {
  const env = testEnvironment();
  env.IMAGES.info = async () => ({ width: 8000, height: 8000, format: "png" });
  let transformations = 0;
  env.IMAGES.input = () => { transformations++; throw new Error("must not transform"); };
  const response = await createHandler({ fetchImpl: passingTurnstile }).fetch(uploadRequest(), env);
  assert.equal(response.status, 400);
  assert.match((await response.json()).error, /dimensions are too large/i);
  assert.equal(transformations, 0);
});

test("sanitizes, stores, and serves an accepted asset", async () => {
  const env = testEnvironment();
  const handler = createHandler({ fetchImpl: passingTurnstile });
  const response = await handler.fetch(uploadRequest(), env);
  assert.equal(response.status, 201);
  const result = await response.json();
  assert.equal(result.kind, "tts-card-back");
  assert.match(result.url, /^https:\/\/custom-assets\.casterscompendium\.com\/assets\/tts-card-back\/[0-9a-f]{64}\.png$/);
  assert.equal(env.CUSTOM_ASSETS.objects.size, 1);

  const stored = [...env.CUSTOM_ASSETS.objects.values()][0];
  assert.equal(stored.options.httpMetadata.contentType, "image/png");
  assert.equal(stored.options.customMetadata.originalFormat, "jpeg");

  const served = await handler.fetch(new Request(result.url), env);
  assert.equal(served.status, 200);
  assert.equal(served.headers.get("Content-Type"), "image/png");
  assert.equal(served.headers.get("Access-Control-Allow-Origin"), "*");
});

test("content-addressed uploads do not rewrite an existing object", async () => {
  const env = testEnvironment();
  let writes = 0;
  const originalPut = env.CUSTOM_ASSETS.put;
  env.CUSTOM_ASSETS.put = async (...args) => { writes++; return originalPut(...args); };
  const handler = createHandler({ fetchImpl: passingTurnstile });
  assert.equal((await handler.fetch(uploadRequest(), env)).status, 201);
  assert.equal((await handler.fetch(uploadRequest(), env)).status, 201);
  assert.equal(writes, 1);
});

test("rate-limited uploads stop before Turnstile and image processing", async () => {
  const env = testEnvironment({
    UPLOAD_RATE_LIMITER: { async limit() { return { success: false }; } },
  });
  let turnstileCalls = 0;
  const response = await createHandler({ fetchImpl: async () => { turnstileCalls++; return passingTurnstile(); } })
    .fetch(uploadRequest(), env);
  assert.equal(response.status, 429);
  assert.equal(turnstileCalls, 0);
});
