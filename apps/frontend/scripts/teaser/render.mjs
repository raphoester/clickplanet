import {execFileSync, spawn} from "node:child_process"
import {copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, statSync, createReadStream, writeFileSync} from "node:fs"
import {tmpdir} from "node:os"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import {createServer} from "vite"

const args = process.argv.slice(2)
const flag = (name, fallback) => {
    const i = args.indexOf(`--${name}`)
    return i === -1 ? fallback : args[i + 1]
}
const has = (name) => args.includes(`--${name}`)

const teaserFile = flag("teaser")
if (!teaserFile || has("help")) {
    console.error(`usage: npm run teaser -- --teaser <teaser.json> [--out <folder>] [--seconds 3]

  --teaser <path>   the battle's teaser, a promo.v1.Teaser as promo's ListTeasers answers it, as JSON
  --out <folder>    where the files go, default the teaser's folder
  --seconds <n>     how long the video shows each card, default 3
  --headed          show the browser

Writes card-<n>.jpg, a still of 1080x1920 for each card in its frame, the photos of a TikTok carousel; teaser.mp4,
a slideshow of them with a silent track, for YouTube; and teaser.json, the teaser it was handed, for the job that
posts them. Needs ffmpeg. CHROME_PATH overrides the browser binary.`)
    process.exit(1)
}

const APP = resolve(dirname(fileURLToPath(import.meta.url)), "../..")
const CHROME = process.env.CHROME_PATH ?? "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
const WIDTH = 540
const HEIGHT = 960
const SCALE = 2
const SECONDS = Number(flag("seconds", "3"))
// A loaded runner starts Chrome slowly: a minute, as the clips found.
const CHROME_START_TRIES = 240
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

const teaserPath = resolve(teaserFile)
if (!existsSync(teaserPath)) {
    console.error(`no teaser at ${teaserPath}`)
    process.exit(1)
}
const out = resolve(flag("out", dirname(teaserPath)))
mkdirSync(out, {recursive: true})

const server = await createServer({
    root: APP,
    configFile: join(APP, "vite.config.ts"),
    logLevel: "warn",
    server: {host: "127.0.0.1", port: 5198},
    plugins: [{
        name: "clickplanet-teaser",
        configureServer(vite) {
            vite.middlewares.use("/__teaser/teaser.json", (_req, res) => {
                res.setHeader("Content-Type", "application/json")
                res.setHeader("Content-Length", statSync(teaserPath).size)
                createReadStream(teaserPath).pipe(res)
            })
        },
    }],
})
await server.listen()
const origin = server.resolvedUrls.local[0]

const profile = mkdtempSync(join(tmpdir(), "clickplanet-teaser-"))
const chrome = spawn(CHROME, [
    ...(has("headed") ? [] : ["--headless=new"]),
    "--remote-debugging-port=0",
    `--user-data-dir=${profile}`,
    "--no-first-run",
    "--no-default-browser-check",
    "--hide-scrollbars",
    "--force-color-profile=srgb",
    "about:blank",
], {stdio: "ignore"})

const done = async (code) => {
    chrome.kill()
    await server.close()
    try {
        rmSync(profile, {recursive: true, force: true, maxRetries: 20, retryDelay: 100})
    } catch {}
    process.exit(code)
}
process.on("SIGINT", () => done(130))

try {
    const page = await openPage()
    await page.open(`${origin}teaser.html`)
    const count = await page.ready()

    for (let i = 0; i < count; i++) {
        await page.evaluate(`window.teaser.show(${i})`)
        writeFileSync(join(out, `card-${i + 1}.jpg`), await page.still())
    }
    slideshow(count)
    if (resolve(join(out, "teaser.json")) !== teaserPath) copyFileSync(teaserPath, join(out, "teaser.json"))
    console.log(`${count} cards in ${out}: card-1.jpg to card-${count}.jpg, teaser.mp4 and teaser.json`)
    await done(0)
} catch (error) {
    console.error(error instanceof Error ? error.message : error)
    await done(1)
}

async function debuggerUrl() {
    for (let i = 0; i < CHROME_START_TRIES; i++) {
        try {
            const port = readFileSync(join(profile, "DevToolsActivePort"), "utf8").split("\n")[0]
            const targets = await fetch(`http://127.0.0.1:${port}/json/list`).then((r) => r.json())
            const target = targets.find((t) => t.type === "page")
            if (target) return target.webSocketDebuggerUrl
        } catch {}
        await sleep(250)
    }
    throw new Error("Chrome never exposed a debugging target")
}

// The page, over Chrome's own protocol, as the clips' recorder drives it: no browser library to install.
async function openPage() {
    const ws = new WebSocket(await debuggerUrl())
    await new Promise((r) => ws.addEventListener("open", r, {once: true}))

    let lastId = 0
    const pending = new Map()
    const requested = new Map()
    let loaded = false
    ws.addEventListener("message", (m) => {
        const msg = JSON.parse(m.data)
        if (msg.id && pending.has(msg.id)) {
            const {resolve, reject} = pending.get(msg.id)
            pending.delete(msg.id)
            msg.error ? reject(new Error(JSON.stringify(msg.error))) : resolve(msg.result)
        } else if (msg.method === "Page.loadEventFired") {
            loaded = true
        } else if (msg.method === "Network.requestWillBeSent") {
            requested.set(msg.params.requestId, msg.params.request.url)
        } else if (msg.method === "Network.responseReceived" && msg.params.response.status >= 400) {
            const {url, status, headers} = msg.params.response
            const mitigated = headers["cf-mitigated"] ? ` (cf-mitigated: ${headers["cf-mitigated"]})` : ""
            console.error(`page: ${url} answered ${status}${mitigated}`)
        } else if (msg.method === "Network.loadingFailed" && !msg.params.canceled) {
            console.error(`page: ${requested.get(msg.params.requestId)} failed: ${msg.params.errorText}`)
        } else if (msg.method === "Runtime.exceptionThrown") {
            console.error("page:", msg.params.exceptionDetails.exception?.description ?? msg.params.exceptionDetails.text)
        }
    })
    const send = (method, params = {}) => new Promise((resolve, reject) => {
        const id = ++lastId
        pending.set(id, {resolve, reject})
        ws.send(JSON.stringify({id, method, params}))
    })
    const evaluate = async (expression) => {
        const {result, exceptionDetails} = await send("Runtime.evaluate", {expression, returnByValue: true, awaitPromise: true})
        if (exceptionDetails) throw new Error(exceptionDetails.exception?.description ?? exceptionDetails.text)
        return result.value
    }

    await send("Page.enable")
    await send("Runtime.enable")
    await send("Network.enable")
    // Cloudflare's Bot Fight Mode, on for the whole zone and skipped by no rule on its plan, answers 403 to
    // "HeadlessChrome" from GitHub's runners: every background of media.clickplanet.lol.
    const {userAgent} = await send("Browser.getVersion")
    await send("Network.setUserAgentOverride", {userAgent: userAgent.replace("HeadlessChrome/", "Chrome/")})
    await send("Emulation.setDeviceMetricsOverride", {width: WIDTH, height: HEIGHT, deviceScaleFactor: SCALE, mobile: false})

    return {
        evaluate,
        async open(url) {
            loaded = false
            await send("Page.navigate", {url})
            for (let i = 0; i < 200 && !loaded; i++) await sleep(100)
        },
        // Once every background and both fonts are in: the number of cards.
        async ready() {
            for (let i = 0; i < 600; i++) {
                const state = await evaluate("({count: window.teaser?.count, failed: window.teaserFailed})")
                if (state.failed) throw new Error(`the teaser page failed: ${state.failed}`)
                if (state.count) return state.count
                await sleep(100)
            }
            throw new Error("the teaser page never got ready")
        },
        // JPEG, which TikTok's photo posts take and PNG is not.
        async still() {
            const {data} = await send("Page.captureScreenshot", {format: "jpeg", quality: 92})
            return Buffer.from(data, "base64")
        },
    }
}

// Each card for SECONDS, then the next, at 30 frames a second, with a silent track: YouTube plays a Short with no
// sound, and some players refuse a file with no audio stream.
function slideshow(count) {
    const list = join(out, "slideshow.txt")
    const lines = []
    for (let i = 1; i <= count; i++) lines.push(`file 'card-${i}.jpg'`, `duration ${SECONDS}`)
    // The concat demuxer takes the last duration from the next entry: the last card is named once more.
    lines.push(`file 'card-${count}.jpg'`)
    writeFileSync(list, lines.join("\n") + "\n")
    try {
        execFileSync("ffmpeg", [
            "-y", "-loglevel", "error",
            "-f", "concat", "-safe", "0", "-i", list,
            "-f", "lavfi", "-i", "anullsrc=r=44100:cl=stereo",
            "-shortest",
            "-vf", "fps=30,format=yuv420p",
            "-c:v", "libx264", "-preset", "veryfast", "-crf", "20",
            "-c:a", "aac", "-b:a", "64k",
            "-movflags", "+faststart",
            join(out, "teaser.mp4"),
        ], {stdio: ["ignore", "inherit", "inherit"]})
    } finally {
        rmSync(list, {force: true})
    }
}
