const escapeHTML = (value) => String(value)
  .replaceAll("&", "&amp;")
  .replaceAll('"', "&quot;")
  .replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;");

export function uploadPage(siteKey, nonce) {
  const safeSiteKey = escapeHTML(siteKey);
  const safeNonce = escapeHTML(nonce);
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="description" content="Upload a custom image for Caster's Compendium and Tabletop Simulator.">
    <meta name="theme-color" content="#17122b">
    <title>Custom image upload · Caster's Compendium</title>
    <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
    <style nonce="${safeNonce}">
      :root { color-scheme: dark; font-family: Inter, system-ui, sans-serif; background: #100c20; color: #f5f0ff; }
      * { box-sizing: border-box; }
      body { margin: 0; min-height: 100vh; background: radial-gradient(circle at top, #312050, #100c20 58%); }
      main { width: min(720px, calc(100% - 32px)); margin: 0 auto; padding: 56px 0; }
      article { padding: clamp(24px, 5vw, 44px); border: 1px solid #6f5799; border-radius: 18px; background: rgba(24, 17, 44, .94); box-shadow: 0 22px 60px #080511aa; }
      h1 { margin-top: 0; font-size: clamp(2rem, 6vw, 3.5rem); line-height: 1; }
      p, label, select, input, button { font-size: 1rem; line-height: 1.55; }
      label { display: block; margin: 22px 0 8px; font-weight: 700; }
      input[type=file], select { width: 100%; padding: 12px; color: inherit; background: #17112b; border: 1px solid #8069a8; border-radius: 8px; }
      .confirmation { display: flex; gap: 10px; align-items: flex-start; margin: 24px 0; font-weight: 400; }
      .confirmation input { margin-top: 6px; }
      button { padding: 11px 18px; border: 0; border-radius: 8px; color: #160e25; background: #e8c7ff; font-weight: 800; cursor: pointer; }
      button:disabled { cursor: wait; opacity: .65; }
      .note { color: #d5c9e8; }
      .result { display: none; margin-top: 24px; padding: 16px; border-radius: 10px; background: #151f19; border: 1px solid #53a96e; overflow-wrap: anywhere; }
      .error { display: none; margin-top: 20px; color: #ffd2d2; }
      code { user-select: all; }
      a { color: #e8c7ff; }
    </style>
  </head>
  <body>
    <main>
      <article>
        <p class="note">Caster's Compendium player assets</p>
        <h1>Upload a custom image.</h1>
        <p>Your image is decoded, resized when necessary, stripped of metadata, and converted to a non-animated PNG. The permanent URL can be used by Tabletop Simulator or a multiplayer simulator session.</p>
        <form id="upload-form">
          <label for="asset-kind">Image use</label>
          <select id="asset-kind" name="assetKind">
            <option value="tts-card-back">Tabletop Simulator card back</option>
            <option value="simulator-card-art">Simulator card art</option>
          </select>

          <label for="image">Image file</label>
          <input id="image" name="image" type="file" accept="image/png,image/jpeg,image/webp,image/avif,image/gif" required>
          <p class="note">Maximum upload: 5 MiB. Card backs must be portrait-oriented. Output is limited to 2048 pixels on either side.</p>

          <label class="confirmation">
            <input name="rightsConfirmed" type="checkbox" value="yes" required>
            <span>I own this image or have permission to upload and share it. I understand the resulting URL is public to anyone who receives it.</span>
          </label>

          <div class="cf-turnstile" data-sitekey="${safeSiteKey}" data-action="custom-asset-upload"></div>
          <p><button id="submit" type="submit">Upload image</button></p>
        </form>
        <p id="error" class="error" role="alert"></p>
        <div id="result" class="result" role="status">
          <strong>Upload complete</strong>
          <p><code id="asset-url"></code></p>
          <button id="copy" type="button">Copy URL</button>
        </div>
        <p class="note">Uploaded files are not listed publicly. They are retained so existing multiplayer saves do not break. Report unlawful or abusive content through the project's <a href="https://github.com/HybridUofA/casters-compendium/security/advisories/new">private report form</a>.</p>
      </article>
    </main>
    <script nonce="${safeNonce}">
      const form = document.getElementById("upload-form");
      const submit = document.getElementById("submit");
      const error = document.getElementById("error");
      const result = document.getElementById("result");
      const assetURL = document.getElementById("asset-url");
      form.addEventListener("submit", async (event) => {
        event.preventDefault();
        error.style.display = "none";
        result.style.display = "none";
        submit.disabled = true;
        try {
          const response = await fetch("/api/v1/assets", { method: "POST", body: new FormData(form) });
          const payload = await response.json();
          if (!response.ok) throw new Error(payload.error || "Upload failed.");
          assetURL.textContent = payload.url;
          result.style.display = "block";
        } catch (uploadError) {
          error.textContent = uploadError instanceof Error ? uploadError.message : "Upload failed.";
          error.style.display = "block";
          if (window.turnstile) window.turnstile.reset();
        } finally {
          submit.disabled = false;
        }
      });
      document.getElementById("copy").addEventListener("click", async () => {
        await navigator.clipboard.writeText(assetURL.textContent);
      });
    </script>
  </body>
</html>`;
}
