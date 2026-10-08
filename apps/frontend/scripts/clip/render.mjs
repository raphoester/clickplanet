import {spawn} from "node:child_process"
import {createReadStream, existsSync, mkdtempSync, readFileSync, rmSync, statSync, writeFileSync} from "node:fs"
import {tmpdir} from "node:os"
import {basename, dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import {createServer} from "vite"

const args = process.argv.slice(2)
const flag = (name, fallback) => {
    const i = args.indexOf(`--${name}`)
    return i === -1 ? fallback : args[i + 1]
}
const has = (name) => args.includes(`--${name}`)

const replay = flag("replay")
if (!replay || has("help")) {
    console.error(`usage: npm run clip -- --replay <replay.json> [options]

  --out <path>        the video, default clip-<n>.mp4 beside the replay
  --count <n>         the n best stories, one clip each, default 1
  --pick <n>          only the n-th best story
  --plan              print what each clip would be, and make none
  --silent            no anthem under the clip, for a sound added where it is posted
  --preview <s>       only the first s seconds of each clip

Everything below is chosen from the replay when left out:
  --since, --until    the window to play, as ISO times; default the shortest one with a lot of action in one place
  --hours <n>         the window's length, its busiest stretch
  --seconds <n>       the clip's length, from 14 to 22 by how much happens
  --country <code>    the attacker, default the flag that took the most where the action is
  --focus <code>      only what changed hands on that country's ground
  --flags             come back out of the tiles halfway, to watch the painted flags change
  --dive              come back out of the tiles at the end
  --headline <text>   the headline
  --hold <s>          how long the opening holds on the whole front before diving in, default 0.3
  --fps <n>           frames a second, default 30
  --headed            show the browser

Writes the video and, beside it, a .txt with the caption. Needs ffmpeg.
CHROME_PATH overrides the browser binary.`)
    process.exit(1)
}

const APP = resolve(dirname(fileURLToPath(import.meta.url)), "../..")
const CHROME = process.env.CHROME_PATH ?? "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
const WIDTH = 540
const HEIGHT = 960
const SCALE = 2
const MUSIC_FADE_SECONDS = 1.2
const MUSIC_FADE_IN_SECONDS = 0.05
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

const replayPath = resolve(replay)
if (!existsSync(replayPath)) {
    console.error(`no replay at ${replayPath}`)
    process.exit(1)
}

const server = await createServer({
    root: APP,
    configFile: join(APP, "vite.config.ts"),
    logLevel: "warn",
    server: {host: "127.0.0.1", port: 5199},
    plugins: [{
        name: "clickplanet-clip-replay",
        configureServer(vite) {
            vite.middlewares.use("/__clip/replay.json", (_req, res) => {
                res.setHeader("Content-Type", "application/json")
                res.setHeader("Content-Length", statSync(replayPath).size)
                createReadStream(replayPath).pipe(res)
            })
        },
    }],
})
await server.listen()
const origin = server.resolvedUrls.local[0]

const PROFILE = mkdtempSync(join(tmpdir(), "clickplanet-clip-"))
const chrome = spawn(CHROME, [
    ...(has("headed") ? [] : ["--headless=new"]),
    "--remote-debugging-port=0",
    `--user-data-dir=${PROFILE}`,
    "--no-first-run",
    "--no-default-browser-check",
    "--hide-scrollbars",
    "--force-color-profile=srgb",
    "--enable-unsafe-swiftshader",
    "--disable-background-timer-throttling",
    "--disable-renderer-backgrounding",
    "--disable-backgrounding-occluded-windows",
    "about:blank",
], {stdio: "ignore"})

let ffmpeg
const done = async (code) => {
    ffmpeg?.kill()
    chrome.kill()
    await server.close()
    try {
        rmSync(PROFILE, {recursive: true, force: true, maxRetries: 20, retryDelay: 100})
    } catch {}
    process.exit(code)
}
process.on("SIGINT", () => done(130))

async function debuggerUrl() {
    for (let i = 0; i < 80; i++) {
        try {
            const port = readFileSync(join(PROFILE, "DevToolsActivePort"), "utf8").split("\n")[0]
            const targets = await fetch(`http://127.0.0.1:${port}/json/list`).then((r) => r.json())
            const page = targets.find((t) => t.type === "page")
            if (page) return page.webSocketDebuggerUrl
        } catch {}
        await sleep(250)
    }
    throw new Error("Chrome never exposed a debugging target")
}

const ws = new WebSocket(await debuggerUrl())
await new Promise((r) => ws.addEventListener("open", r, {once: true}))

let lastId = 0
const pending = new Map()
let loaded = false
ws.addEventListener("message", (m) => {
    const msg = JSON.parse(m.data)
    if (msg.id && pending.has(msg.id)) {
        const {resolve, reject} = pending.get(msg.id)
        pending.delete(msg.id)
        msg.error ? reject(new Error(JSON.stringify(msg.error))) : resolve(msg.result)
    } else if (msg.method === "Page.loadEventFired") {
        loaded = true
    } else if (msg.method === "Runtime.exceptionThrown") {
        console.error("page:", msg.params.exceptionDetails.exception?.description ?? msg.params.exceptionDetails.text)
    } else if (msg.method === "Runtime.consoleAPICalled" && ["error", "warning"].includes(msg.params.type)) {
        console.error(`page ${msg.params.type}:`, msg.params.args.map((a) => a.value ?? a.description).join(" "))
    }
})
const send = (method, params = {}) => new Promise((resolve, reject) => {
    const id = ++lastId
    pending.set(id, {resolve, reject})
    ws.send(JSON.stringify({id, method, params}))
})
const evaluate = async (expression) => {
    const {result, exceptionDetails} = await send("Runtime.evaluate", {
        expression, returnByValue: true, awaitPromise: true,
    })
    if (exceptionDetails) throw new Error(exceptionDetails.exception?.description ?? exceptionDetails.text)
    return result.value
}

let skippedTold = false

async function record(pick, chosenOut) {
    loaded = false
    const query = new URLSearchParams({fps: flag("fps", "30"), pick: String(pick)})
    for (const name of ["since", "until", "hours", "seconds", "country", "focus", "headline", "hold"]) {
        if (flag(name)) query.set(name, flag(name))
    }
    for (const name of ["flags", "dive"]) {
        if (has(name)) query.set(name, "")
    }
    await send("Page.navigate", {url: `${origin}clip.html?${query}`})
    for (let i = 0; i < 200 && !loaded; i++) await sleep(100)

    const recording = await evaluate("window.clip.ready")
    const out = chosenOut ?? join(dirname(replayPath), `clip-${pick}.mp4`)
    if (!skippedTold) {
        for (const reason of recording.skipped) console.log(`skipped ${reason}`)
        skippedTold = true
    }
    console.log([
        `#${recording.pick} of ${recording.stories}: ${recording.headline}`,
        ...recording.line ? [`  ${recording.line}`] : [],
        `  window: ${recording.since} to ${recording.until}`,
        `  place: ${recording.place}, ${recording.seconds}s`,
        `  look: ${recording.look}`,
        `  music: ${recording.music ? `${recording.music.title}, the anthem of ${recording.music.whose}, from ${recording.music.from.toFixed(1)}s` : "none"}`,
    ].join("\n"))
    if (has("plan")) return recording

    const frames = has("preview") ? Math.min(recording.frames, Math.round(Number(flag("preview")) * recording.fps)) : recording.frames
    const music = has("silent") ? undefined : recording.music
    const fadeFrom = Math.max(0, frames / recording.fps - MUSIC_FADE_SECONDS)
    ffmpeg = spawn("ffmpeg", [
        "-y", "-loglevel", "error",
        "-f", "image2pipe", "-framerate", String(recording.fps), "-i", "-",
        ...music ? ["-ss", music.from.toFixed(2), "-i", join(APP, music.url)] : [],
        "-c:v", "libx264", "-preset", "slow", "-crf", "18", "-pix_fmt", "yuv420p",
        ...music ? [
            "-map", "0:v", "-map", "1:a", "-c:a", "aac", "-b:a", "128k",
            "-af", `afade=t=in:d=${MUSIC_FADE_IN_SECONDS},afade=t=out:st=${fadeFrom.toFixed(2)}:d=${MUSIC_FADE_SECONDS},apad`, "-shortest",
        ] : [],
        "-movflags", "+faststart", out,
    ], {stdio: ["pipe", "inherit", "inherit"]})
    const encoded = new Promise((resolve, reject) => {
        ffmpeg.on("error", reject)
        ffmpeg.on("exit", (code) => code === 0 ? resolve() : reject(new Error(`ffmpeg exited with ${code}`)))
    })

    const started = Date.now()
    for (let frame = 0; frame < frames; frame++) {
        await evaluate(`window.clip.frame(${frame})`)
        const {data} = await send("Page.captureScreenshot", {format: "png", optimizeForSpeed: true})
        if (!ffmpeg.stdin.write(Buffer.from(data, "base64"))) {
            await new Promise((r) => ffmpeg.stdin.once("drain", r))
        }
        if (frame % (recording.fps * 5) === 0) {
            const rate = (frame + 1) / ((Date.now() - started) / 1000)
            console.log(`  frame ${frame}/${frames} (${rate.toFixed(1)} frames a second)`)
        }
    }
    ffmpeg.stdin.end()
    await encoded
    ffmpeg = undefined

    const caption = out.replace(/\.mp4$/, "") + ".txt"
    writeFileSync(caption, recording.caption + (music?.credit ? `\n🎵 ${music.credit}` : "") + "\n")
    console.log(`  saved ${out} and ${basename(caption)}`)
    return recording
}

try {
    await send("Page.enable")
    await send("Runtime.enable")
    await send("Emulation.setDeviceMetricsOverride", {
        width: WIDTH, height: HEIGHT, deviceScaleFactor: SCALE, mobile: false,
    })

    const first = Number(flag("pick", "1"))
    const last = has("pick") ? first : Number(flag("count", "1"))
    for (let pick = first; pick <= last; pick++) {
        const recorded = await record(pick, last > first || !has("out") ? undefined : resolve(flag("out")))
        if (recorded.stories <= pick) break
    }
} catch (error) {
    console.error(error.message.split("\n").filter((line) => !line.trimStart().startsWith("at ")).join("\n"))
    await done(1)
}

await done(0)
