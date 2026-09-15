import { uploadPage } from "./page.mjs";

export const MAX_UPLOAD_BYTES = 5 * 1024 * 1024;
export const MAX_OUTPUT_BYTES = 8 * 1024 * 1024;
export const OUTPUT_MAX_DIMENSION = 2048;
export const MAX_INPUT_PIXELS = 25_000_000;

const assetProfiles = Object.freeze({
  "tts-card-back": { minimumSide: 256, minimumRatio: 0.5, maximumRatio: 0.85 },
  "simulator-card-art": { minimumSide: 128, minimumRatio: 0.25, maximumRatio: 4 },
});

const acceptedFormats = new Set(["avif", "gif", "jpeg", "jpg", "png", "webp"]);

const json = (payload, status = 200, headers = {}) => new Response(JSON.stringify(payload), {
  status,
  headers: {
    "Content-Type": "application/json; charset=utf-8",
    "Cache-Control": "no-store",
    ...securityHeaders(),
    ...headers,
  },
});

function securityHeaders() {
  return {
    "X-Content-Type-Options": "nosniff",
    "Referrer-Policy": "no-referrer",
    "Permissions-Policy": "camera=(), geolocation=(), microphone=()",
    "Cross-Origin-Resource-Policy": "cross-origin",
  };
}

function pageResponse(env) {
  if (!env.TURNSTILE_SITE_KEY) {
    return new Response("Upload service is not configured.", { status: 503, headers: securityHeaders() });
  }
  const nonceBytes = crypto.getRandomValues(new Uint8Array(16));
  const nonce = [...nonceBytes].map((value) => value.toString(16).padStart(2, "0")).join("");
  return new Response(uploadPage(env.TURNSTILE_SITE_KEY, nonce), {
    headers: {
      "Content-Type": "text/html; charset=utf-8",
      "Cache-Control": "public, max-age=300",
      "Content-Security-Policy": `default-src 'none'; script-src 'nonce-${nonce}' https://challenges.cloudflare.com; frame-src https://challenges.cloudflare.com; style-src 'nonce-${nonce}'; connect-src 'self' https://challenges.cloudflare.com; img-src 'self' data:; form-action 'self'; base-uri 'none'; frame-ancestors 'none'`,
      ...securityHeaders(),
    },
  });
}

function normalizeFormat(format) {
  return String(format || "").toLowerCase().replace(/^image\//, "");
}

function validOrigin(request, env) {
  const expected = String(env.PUBLIC_ORIGIN || "").replace(/\/$/, "");
  return expected !== "" && request.headers.get("Origin") === expected;
}

async function verifyTurnstile(token, request, env, fetchImpl) {
  if (!env.TURNSTILE_SECRET_KEY || !env.TURNSTILE_EXPECTED_HOSTNAME) return false;
  if (typeof token !== "string" || token.length < 1 || token.length > 2048) return false;

  const body = new URLSearchParams({
    secret: env.TURNSTILE_SECRET_KEY,
    response: token,
    remoteip: request.headers.get("CF-Connecting-IP") || "",
  });
  let response;
  try {
    response = await fetchImpl("https://challenges.cloudflare.com/turnstile/v0/siteverify", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body,
    });
  } catch {
    return false;
  }
  if (!response.ok) return false;
  const result = await response.json();
  return result.success === true
    && result.hostname === env.TURNSTILE_EXPECTED_HOSTNAME
    && result.action === "custom-asset-upload";
}

async function sha256Hex(bytes) {
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  return [...new Uint8Array(digest)].map((value) => value.toString(16).padStart(2, "0")).join("");
}

function imageStream(bytes) {
  return new Response(bytes).body;
}

async function sanitizeImage(bytes, env) {
  let info;
  try {
    info = await env.IMAGES.info(imageStream(bytes));
  } catch {
    throw new Error("The selected file is not a supported image.");
  }
  const width = Number(info.width);
  const height = Number(info.height);
  const format = normalizeFormat(info.format);
  if (!Number.isSafeInteger(width) || !Number.isSafeInteger(height) || width < 1 || height < 1) {
    throw new Error("The image dimensions could not be verified.");
  }
  if (width > 8000 || height > 8000 || width * height > MAX_INPUT_PIXELS) {
    throw new Error("The image dimensions are too large.");
  }
  if (!acceptedFormats.has(format)) {
    throw new Error("Use a PNG, JPEG, WebP, AVIF, or GIF image.");
  }

  let transformed;
  try {
    transformed = await env.IMAGES
      .input(imageStream(bytes))
      .transform({ width: OUTPUT_MAX_DIMENSION, height: OUTPUT_MAX_DIMENSION, fit: "scale-down" })
      .output({ format: "image/png", anim: false });
  } catch {
    throw new Error("The image could not be safely decoded.");
  }
  const response = transformed.response();
  if (!response.ok) throw new Error("The image could not be safely converted.");
  const output = await response.arrayBuffer();
  if (output.byteLength < 1 || output.byteLength > MAX_OUTPUT_BYTES) {
    throw new Error("The converted image is too large.");
  }
  return { output, width, height, format };
}

function validateProfile(kind, width, height) {
  const profile = assetProfiles[kind];
  if (!profile) return "Choose a supported image use.";
  if (Math.min(width, height) < profile.minimumSide) {
    return `The image must be at least ${profile.minimumSide} pixels on its shortest side.`;
  }
  const ratio = width / height;
  if (ratio < profile.minimumRatio || ratio > profile.maximumRatio) {
    return kind === "tts-card-back"
      ? "A TTS card back must be a portrait image with a card-like aspect ratio."
      : "The image aspect ratio is too extreme.";
  }
  return "";
}

async function uploadAsset(request, env, fetchImpl) {
  if (!validOrigin(request, env)) return json({ error: "Upload origin was rejected." }, 403);

  const declaredLengthHeader = request.headers.get("Content-Length");
  const declaredLength = Number(declaredLengthHeader);
  if (!declaredLengthHeader || !Number.isSafeInteger(declaredLength) || declaredLength < 1) {
    return json({ error: "Uploads must include a valid Content-Length." }, 411);
  }
  if (declaredLength > MAX_UPLOAD_BYTES + 128 * 1024) {
    return json({ error: "The upload exceeds the 5 MiB limit." }, 413);
  }
  if (!request.headers.get("Content-Type")?.toLowerCase().startsWith("multipart/form-data;")) {
    return json({ error: "Expected a multipart image upload." }, 415);
  }

  const actor = request.headers.get("CF-Connecting-IP") || "unknown";
  const rate = await env.UPLOAD_RATE_LIMITER.limit({ key: `upload:${actor}` });
  if (!rate.success) return json({ error: "Too many uploads. Please wait a minute and try again." }, 429);

  let form;
  try {
    form = await request.formData();
  } catch {
    return json({ error: "The upload form could not be read." }, 400);
  }
  if (form.get("rightsConfirmed") !== "yes") {
    return json({ error: "You must confirm that you may share this image." }, 400);
  }
  const kind = String(form.get("assetKind") || "");
  if (!assetProfiles[kind]) return json({ error: "Choose a supported image use." }, 400);

  const turnstileOK = await verifyTurnstile(
    form.get("cf-turnstile-response"), request, env, fetchImpl,
  );
  if (!turnstileOK) return json({ error: "Human verification failed. Please try again." }, 403);

  const upload = form.get("image");
  if (!upload || typeof upload.arrayBuffer !== "function") {
    return json({ error: "Choose an image file." }, 400);
  }
  if (upload.size < 1 || upload.size > MAX_UPLOAD_BYTES) {
    return json({ error: "The image must be between 1 byte and 5 MiB." }, 413);
  }
  const input = await upload.arrayBuffer();

  let sanitized;
  try {
    sanitized = await sanitizeImage(input, env);
  } catch (error) {
    return json({ error: error instanceof Error ? error.message : "Image validation failed." }, 400);
  }
  const profileError = validateProfile(kind, sanitized.width, sanitized.height);
  if (profileError) return json({ error: profileError }, 400);

  const hash = await sha256Hex(sanitized.output);
  const key = `v1/${kind}/${hash}.png`;
  const existing = await env.CUSTOM_ASSETS.head(key);
  if (!existing) {
    await env.CUSTOM_ASSETS.put(key, sanitized.output, {
      httpMetadata: {
        contentType: "image/png",
        cacheControl: "public, max-age=31536000, immutable",
      },
      customMetadata: {
        assetKind: kind,
        originalFormat: sanitized.format,
        originalWidth: String(sanitized.width),
        originalHeight: String(sanitized.height),
      },
      onlyIf: { etagDoesNotMatch: "*" },
    });
  }

  const origin = String(env.PUBLIC_ORIGIN).replace(/\/$/, "");
  return json({
    schemaVersion: 1,
    kind,
    url: `${origin}/assets/${kind}/${hash}.png`,
    sha256: hash,
  }, 201);
}

async function serveAsset(request, env, match) {
  const [, kind, hash] = match;
  const object = await env.CUSTOM_ASSETS.get(`v1/${kind}/${hash}.png`);
  if (!object) return new Response("Not found.", { status: 404, headers: securityHeaders() });

  const headers = new Headers(securityHeaders());
  if (typeof object.writeHttpMetadata === "function") object.writeHttpMetadata(headers);
  headers.set("Content-Type", "image/png");
  headers.set("Cache-Control", "public, max-age=31536000, immutable");
  headers.set("Access-Control-Allow-Origin", "*");
  if (object.httpEtag) headers.set("ETag", object.httpEtag);
  return new Response(request.method === "HEAD" ? null : object.body, { headers });
}

export function createHandler({ fetchImpl = fetch } = {}) {
  return {
    async fetch(request, env) {
      const url = new URL(request.url);
      if (url.pathname === "/healthz" && request.method === "GET") {
        return json({ status: "ok" });
      }
      if (url.pathname === "/" && request.method === "GET") return pageResponse(env);
      if (url.pathname === "/api/v1/assets" && request.method === "POST") {
        return uploadAsset(request, env, fetchImpl);
      }
      const assetMatch = url.pathname.match(/^\/assets\/(tts-card-back|simulator-card-art)\/([0-9a-f]{64})\.png$/);
      if (assetMatch && (request.method === "GET" || request.method === "HEAD")) {
        return serveAsset(request, env, assetMatch);
      }
      if (!["GET", "HEAD", "POST"].includes(request.method)) {
        return new Response("Method not allowed.", { status: 405, headers: { Allow: "GET, HEAD, POST", ...securityHeaders() } });
      }
      return new Response("Not found.", { status: 404, headers: securityHeaders() });
    },
  };
}

export default createHandler();
