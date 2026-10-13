# Public Verifier — Zero-Cost Edge Deployment

Sovereign, fully offline WebCrypto verifier: Ed25519 DID envelopes (RFC 6962
Merkle inclusion proofs). One static HTML file, zero backend, zero network
calls, no build step.

`index.html` is the single source of truth — `pkg/verification/verifier_page.go`
mirrors it byte-for-byte (asserted by
`TestPublicVerifierHTMLMatchesToolsIndex`). To change the page, edit
`index.html` and the Go copy together.

## Security headers

Cloudflare Pages (and other static hosts that honor `_headers`) apply these
response headers via the `_headers` file in this directory:

```text
Content-Security-Policy: default-src 'self' 'unsafe-inline'; frame-ancestors 'none';
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
```

`'unsafe-inline'` is required because the logic ships as inline `<script>` and
`<style>` in the single file. `frame-ancestors 'none'` plus the URL-fragment
bootstrap (`#sig=…`) keep the verifier unembeddable and referrer-free.

## Option A — Cloudflare Pages (free plan, CLI)

```bash
npm install -g wrangler          # one-time, or use npx
wrangler login                   # or set CLOUDFLARE_API_TOKEN
wrangler pages deploy tools/public-verifier --project-name INTEGIN-public-verifier
```

Deploys `index.html` + `_headers` as-is (static, no build command needed).
Output includes a `*.pages.dev` URL. Verify the headers:

```bash
curl -sI https://<project>.pages.dev/ \
  | grep -iE 'content-security|x-content-type|referrer-policy'
```

Alternative without the CLI: push the folder to a GitHub repo, then enable
**Cloudflare Pages → Import a Git repository**; set Build command to empty and
Output directory to `tools/public-verifier`.

## Option B — GitHub Pages (free, no headers)

1. Push `index.html` (and nothing else) to the `gh-pages` branch, or
2. Enable **Settings → Pages → Deploy from a branch** and set the source
   folder to `/tools/public-verifier` (same branch).

GitHub Pages cannot set custom HTTP response headers, so the CSP/nosniff header
triplet is `_headers`-only there; the page is still fully functional — the
`did:key` fragment bootstrap, envelope verification, and Merkle inclusion walk
all run client-side. If header enforcement matters, use Option A.

## Verification-of-page sanity check

```bash
# hash must equal the payload after any deploy; the page recomputes it client-side
# independent of transport headers, so no deploy-time check is strictly required.
```

The page integrity itself is covered by Go tests in `pkg/verification`, which
compile a serving copy and pin the tools pages to it.