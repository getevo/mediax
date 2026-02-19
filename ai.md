# MediaX — AI Agent Integration Guide

This file is written for AI coding agents (Claude Code, Copilot, Cursor, etc.).
Follow the instructions below precisely when upgrading a frontend project to serve media through MediaX.

---

## What MediaX is

MediaX is a self-hosted media proxy and processing server. It sits between a frontend and the underlying file storage (local disk, S3, GCS, HTTP CDN). When the frontend requests a media URL, MediaX fetches the original file from storage, applies on-the-fly transformations (resize, reformat, compress, transcode, thumbnail), and streams the result to the browser.

The frontend **never talks to storage directly**. Every media URL points at MediaX.

---

## How MediaX identifies a request

MediaX routes requests using the **HTTP `Host` header**, not the URL path alone.
When a request arrives, MediaX looks up the incoming hostname in its `Origins` table. Each Origin maps a domain to a `Project`, which has one or more `Storage` backends. If the hostname is not in the `Origins` table, MediaX returns `403 Forbidden`.

**Consequence for frontend integration:** every media URL the frontend constructs must reach MediaX with a `Host` header that matches a configured Origin domain. The safest and most common way to achieve this is through a reverse-proxy rule that forwards a dedicated URL prefix to MediaX while overriding the `Host` header.

---

## The path-base rule (critical)

A frontend application has its own path base — the prefix under which it serves its own pages and API calls.
For example, a frontend running at `https://app.example.com` might serve:

```
/              → React/Vue/Angular app shell
/api/          → backend API proxy
/static/       → bundled assets
```

**The media path base must be a completely separate prefix that does NOT overlap with any of the frontend's own routes.**

Recommended convention: use `/media/` as the dedicated media prefix.

```
/media/**      → proxied to MediaX (separate domain header)
everything else → served by the frontend / backend as usual
```

If the frontend's path base is already `/app` (i.e. it is mounted at `/app`), the media prefix must still be outside of it, for example `/media/`. Never use a sub-path of the frontend path base for media.

---

## Reverse-proxy configuration

The frontend must proxy its dedicated media prefix to MediaX, forwarding the correct `Host` header.

### Nginx (production)

```nginx
server {
    listen 443 ssl;
    server_name example.com;

    # All other frontend routes
    location / {
        proxy_pass http://frontend:3000;
        proxy_set_header Host $host;
    }

    # Media prefix → MediaX
    location /media/ {
        proxy_pass http://mediax:8080/media/;
        proxy_set_header Host media.example.com;   # must match the Origin "domain"
        proxy_set_header X-Real-IP $remote_addr;
        proxy_cache_valid 200 1d;
    }
}
```

### Next.js / Nuxt dev proxy (`next.config.js`)

```js
module.exports = {
  async rewrites() {
    return [
      {
        source: '/media/:path*',
        destination: 'http://localhost:8080/media/:path*',
      },
    ]
  },
}
```

Because a dev proxy rewrite does not change the `Host` header automatically, configure a MediaX Origin with `domain: "localhost"` or use a tool like nginx even in dev.

### Vite dev proxy (`vite.config.js`)

```js
export default {
  server: {
    proxy: {
      '/media': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        headers: { Host: 'media.example.com' },
      },
    },
  },
}
```

---

## URL structure

Every media URL follows this pattern:

```
https://<frontend-domain>/media/<path-to-file-in-storage>?<parameters>
```

The `/media/` prefix is stripped by MediaX (via `prefix_path`) before the file is looked up in storage. So if a file is stored at `images/hero.jpg` in the storage backend, the frontend URL is:

```
/media/images/hero.jpg
```

---

## Complete parameter reference (from source)

All parameters are query-string parameters appended to the media URL.

### Image parameters

| Parameter | Type    | Description                                                        | Example              |
|-----------|---------|--------------------------------------------------------------------|----------------------|
| `width`   | integer | Output width in pixels (snapped to nearest standard size)          | `?width=800`         |
| `height`  | integer | Output height in pixels                                            | `?height=600`        |
| `size`    | string  | Shorthand `WxH` — sets both width and height                      | `?size=800x600`      |
| `format`  | string  | Output format: `jpg`, `png`, `gif`, `webp`, `avif`. **Always prefer `webp`** — it is supported by all modern browsers and produces 25–35% smaller files than JPEG at the same visual quality. Only fall back to `jpg`/`png` when the browser or use-case explicitly requires it. | `?format=webp` |
| `q`       | integer | Quality 1–100                                                      | `?q=85`              |
| `crop`    | string  | Enable cropping (set to any non-empty value). Without this parameter, aspect ratio is preserved automatically | `?crop=center` |
| `dir`     | string  | Crop anchor: `center`, `top`, `bottom`, `left`, `right`           | `?dir=top`           |
| `download`| bool    | Force `Content-Disposition: attachment`                            | `?download=true`     |

When both `width` and `height` are specified, cropping is applied automatically unless `crop` is omitted.
When only one dimension is given, the other scales proportionally.

**Supported input formats:** JPG, PNG, GIF, WebP, AVIF
**Supported output formats:** JPG, PNG, GIF, WebP, AVIF

### Video parameters

| Parameter   | Type    | Description                                                          | Example               |
|-------------|---------|----------------------------------------------------------------------|-----------------------|
| `width`     | integer | Output width                                                         | `?width=1280`         |
| `height`    | integer | Output height                                                        | `?height=720`         |
| `format`    | string  | Output format: `mp4`, `webm`, `avi`, `mov`, `mkv`, `flv`, `wmv`, `m4v`, `3gp`, `ogv`, `jpg`, `png`, `webp`, `avif` | `?format=webm` |
| `q`         | integer | Quality 1–100                                                        | `?q=75`               |
| `profile`   | string  | Named encoding profile (configured in admin)                         | `?profile=hd`         |
| `preview`   | string  | Stream a lower-resolution preview: `480p`, `720p`, `1080p`, `4k`, or `WxH` | `?preview=720p` |
| `thumbnail` | string  | Extract a still frame as an image (`480p`, `720p`, `1080p`, `4k`, or `WxH`) | `?thumbnail=480p&format=jpg` |
| `ss`        | integer | Timestamp in seconds for thumbnail extraction                        | `?ss=30`              |
| `download`  | bool    | Force download                                                       | `?download=true`      |

**Supported input formats:** MP4, WebM, AVI, MOV, MKV, FLV, WMV, M4V, 3GP, OGV
**Thumbnail output:** JPG, PNG, WebP, AVIF

### Audio parameters

| Parameter | Type   | Description                                                                    | Example           |
|-----------|--------|--------------------------------------------------------------------------------|-------------------|
| `format`  | string | Output format: `mp3`, `wav`, `flac`, `aac`, `ogg`, `m4a`, `wma`, `opus`; or `jpg`/`png`/`webp`/`avif` to extract album art | `?format=flac` |
| `q`       | integer| Quality 1–100                                                                  | `?q=100`          |
| `detail`  | bool   | Return JSON metadata instead of audio data (title, artist, album, bitrate, …) | `?detail=true`    |

**Supported input formats:** MP3, WAV, FLAC, AAC, OGG, M4A, WMA, Opus

### Document parameters

| Parameter   | Type   | Description                                                             | Example                        |
|-------------|--------|-------------------------------------------------------------------------|--------------------------------|
| `thumbnail` | string | Generate a thumbnail image at `WxH` resolution (e.g. `1200x1700`)     | `?thumbnail=1200x1700&format=jpg` |
| `format`    | string | Thumbnail format: `jpg`, `png`, `webp`, `avif`                         | `?format=webp`                 |
| `q`         | integer| Thumbnail quality 1–100                                                 | `?q=90`                        |

**Supported input formats:** PDF, DOCX, XLSX, PPTX, DOC, XLS, PPT, ODT, ODS, ODP, TXT, RTF, CSV, EPUB, XML

---

## Frontend implementation instructions

When upgrading a frontend codebase, follow these steps in order.

### Step 1 — Identify the media base URL

Find where the project stores the base URL for media/CDN assets. It may be in:
- An environment variable (`VITE_CDN_URL`, `NEXT_PUBLIC_MEDIA_URL`, `VUE_APP_MEDIA_URL`, etc.)
- A config file (`config.ts`, `constants.ts`, `settings.js`, etc.)
- Hardcoded `<img src="https://cdn.example.com/...">` tags scattered in components

Set the base to the frontend's own media prefix path:

```ts
// config.ts  (or .env)
export const MEDIA_BASE = '/media'   // always a root-relative path, never the CDN domain directly
```

The frontend must NEVER send media requests directly to the storage bucket or CDN. All requests go through `/media/`.

### Step 2 — Create a URL builder utility

Create a single utility function that all media URLs in the project flow through. Place it in a shared location (e.g., `src/utils/media.ts` or `src/lib/media.js`).

```ts
// src/utils/media.ts

const MEDIA_BASE = import.meta.env.VITE_MEDIA_BASE ?? '/media'

interface ImageOptions {
  width?: number
  height?: number
  size?: string          // "WxH" shorthand, overrides width/height
  format?: 'jpg' | 'png' | 'gif' | 'webp' | 'avif'  // default: 'webp' — best compression for modern browsers
  quality?: number       // 1–100
  crop?: string          // crop anchor: 'center' | 'top' | 'bottom' | 'left' | 'right'
  download?: boolean
}

interface VideoOptions {
  width?: number
  height?: number
  format?: 'mp4' | 'webm' | 'avi' | 'mov' | 'mkv' | 'flv' | 'wmv' | 'm4v' | '3gp' | 'ogv' | 'jpg' | 'png' | 'webp' | 'avif'
  quality?: number
  profile?: string
  preview?: string       // '480p' | '720p' | '1080p' | '4k' | 'WxH'
  thumbnail?: string     // '480p' | '720p' | '1080p' | '4k' | 'WxH'
  ss?: number            // timestamp in seconds
  download?: boolean
}

interface AudioOptions {
  format?: 'mp3' | 'wav' | 'flac' | 'aac' | 'ogg' | 'm4a' | 'wma' | 'opus' | 'jpg' | 'png' | 'webp' | 'avif'
  quality?: number
  detail?: boolean
}

interface DocumentOptions {
  thumbnail?: string     // 'WxH' e.g. '1200x1700'
  format?: 'jpg' | 'png' | 'webp' | 'avif'
  quality?: number
}

function buildParams(opts: Record<string, string | number | boolean | undefined>): string {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(opts)) {
    if (v !== undefined && v !== null && v !== '') {
      p.set(k, String(v))
    }
  }
  const s = p.toString()
  return s ? '?' + s : ''
}

/** Constructs a proxied image URL with on-the-fly processing parameters.
 *  Defaults to webp format for best compression. Override only when necessary. */
export function imageUrl(path: string, opts: ImageOptions = {}): string {
  const { width, height, size, format = 'webp', quality, crop, download } = opts
  return (
    MEDIA_BASE +
    '/' +
    path.replace(/^\//, '') +
    buildParams({ width, height, size, format, q: quality, crop, download })
  )
}

/** Constructs a proxied video URL. Use thumbnail+format to request a still image. */
export function videoUrl(path: string, opts: VideoOptions = {}): string {
  const { width, height, format, quality, profile, preview, thumbnail, ss, download } = opts
  return (
    MEDIA_BASE +
    '/' +
    path.replace(/^\//, '') +
    buildParams({ width, height, format, q: quality, profile, preview, thumbnail, ss, download })
  )
}

/** Constructs a proxied audio URL. Use format=jpg/png/webp/avif to extract album art. */
export function audioUrl(path: string, opts: AudioOptions = {}): string {
  const { format, quality, detail } = opts
  return MEDIA_BASE + '/' + path.replace(/^\//, '') + buildParams({ format, q: quality, detail })
}

/** Constructs a proxied document URL. Always specify thumbnail+format to get an image rendition. */
export function documentUrl(path: string, opts: DocumentOptions = {}): string {
  const { thumbnail, format, quality } = opts
  return MEDIA_BASE + '/' + path.replace(/^\//, '') + buildParams({ thumbnail, format, q: quality })
}
```

### Step 3 — Replace hardcoded media URLs in components

Search the codebase for all patterns that produce media URLs:

```
grep -r "cdn.example.com" src/
grep -r "storage.googleapis.com" src/
grep -r "s3.amazonaws.com" src/
grep -r 'src="http' src/
grep -r "https://.*\.(jpg|png|gif|webp|mp4|mp3|pdf)" src/
```

Replace each occurrence. Examples:

**Before**
```tsx
<img src="https://cdn.example.com/photos/hero.jpg" width={1200} height={600} />
```

**After**
```tsx
import { imageUrl } from '@/utils/media'

<img src={imageUrl('photos/hero.jpg', { width: 1200, height: 600, format: 'webp', quality: 85 })} />
```

**Before** (avatar thumbnail)
```tsx
<img src={`https://cdn.example.com/avatars/${user.id}.jpg`} />
```

**After**
```tsx
<img src={imageUrl(`avatars/${user.id}.jpg`, { size: '96x96', crop: 'center', format: 'webp' })} />
```

**Before** (video)
```tsx
<video src="https://cdn.example.com/videos/intro.mp4" />
```

**After**
```tsx
<video src={videoUrl('videos/intro.mp4', { format: 'mp4' })} />
<img src={videoUrl('videos/intro.mp4', { thumbnail: '480p', format: 'jpg', ss: 5 })} />
```

**Before** (document)
```tsx
<a href="https://cdn.example.com/docs/report.pdf">Report</a>
```

**After**
```tsx
{/* Link serves the original PDF */}
<a href={documentUrl('docs/report.pdf')}>Report</a>
{/* Preview thumbnail */}
<img src={documentUrl('docs/report.pdf', { thumbnail: '800x1100', format: 'jpg' })} />
```

### Step 4 — Handle responsive images with `srcset`

For responsive images, call `imageUrl` multiple times with different widths:

```tsx
function ResponsiveImage({ path, alt }: { path: string; alt: string }) {
  return (
    <img
      src={imageUrl(path, { width: 800, format: 'webp', quality: 85 })}
      srcSet={[
        imageUrl(path, { width: 480, format: 'webp', quality: 85 }) + ' 480w',
        imageUrl(path, { width: 800, format: 'webp', quality: 85 }) + ' 800w',
        imageUrl(path, { width: 1200, format: 'webp', quality: 85 }) + ' 1200w',
      ].join(', ')}
      sizes="(max-width: 600px) 480px, (max-width: 1024px) 800px, 1200px"
      alt={alt}
    />
  )
}
```

### Step 5 — Environment variable wiring

Add the variable to all relevant environment files:

```bash
# .env.development
VITE_MEDIA_BASE=/media          # frontend dev server proxies /media → MediaX

# .env.production
VITE_MEDIA_BASE=/media          # nginx/CDN proxies /media → MediaX

# Next.js
NEXT_PUBLIC_MEDIA_BASE=/media
```

Never put the raw MediaX server URL here — the `/media` prefix is always served from the same domain as the frontend, via the reverse proxy.

---

## Debugging

Add the `X-Debug: 1` request header to any media request. MediaX will return detailed headers explaining what happened:

```
X-Trace-ID         – unique request identifier
X-Debug-Host       – resolved hostname
X-Debug-Extension  – detected file extension
X-Debug-MediaType  – matched media type config
X-Debug-Options    – parsed processing parameters
X-Debug-Error      – error detail (if any)
X-Debug-Storage-N-Type     – storage type tried (index N)
X-Debug-Storage-N-BasePath – base path for that storage
X-Debug-Storage-N-Error    – error from that storage attempt
```

Use this to diagnose 404s (wrong storage path), 403s (unregistered domain), or processing errors.

---

## Common mistakes to avoid

| Mistake | Correct approach |
|---------|-----------------|
| Pointing `MEDIA_BASE` to the raw MediaX host (`http://mediax:8080`) | Always go through the frontend's proxy at `/media` |
| Using the same path prefix for both frontend routes and media | Reserve `/media/` exclusively for MediaX; keep it out of the frontend router |
| Sending requests to `storage.googleapis.com` or S3 directly | All media goes through MediaX |
| Forgetting to set the `Host` header in the proxy rule | MediaX returns 403 if the hostname doesn't match an Origin |
| Using `filepath.Join`-style paths with backslashes in storage config (Windows) | Always use forward slashes in DSN and config paths |
| Not stripping the prefix in `prefix_path` (Origin config) | The `prefix_path` must match the proxy location prefix exactly |
