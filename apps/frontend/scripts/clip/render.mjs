import {execFileSync, spawn} from "node:child_process"
import {createReadStream, existsSync, mkdtempSync, readFileSync, rmSync, statSync, writeFileSync} from "node:fs"
import {availableParallelism, tmpdir} from "node:os"
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

  --story <json>      the story to tell, a clips.v1.Pick: one front, tour, place or seasonal story of the findings by
                      its number, and for a front its kind, sides and place as the editor wrote them, with its headline.
                      The job's choose prints one for each clip of a run; clips edit prints the model's by hand
  --findings <path>   write what the replay holds for the editor (clips.v1.Findings), as JSON, and make no clip
  --pick <n>          the clip's number in its run, which names its files: clip-<n>.mp4, default 1
  --out <path>        the video, default clip-<n>.mp4 beside the replay
  --plan              print what the clip would be, and make none
  --race <path>       the season's race (scoring.v1.GetRaceResponse): the day it closed last is a story when its
                      cutoff is in the window
  --season <path>     the season (scoring.v1.GetSeasonResponse): with the race, each country that can still win it is
                      called up to its Final Battle
  --cast <path>       who took which tiles (clips.v1.Cast): the fighters the story names are on its end card, and the
                      findings name each front's, tour's and place's top fighters
  --silent            no anthem under the clip, for a sound added where it is posted
  --preview <s>       only the first s seconds of the clip
  --variants <json>   the arm of each experiment the clip is in, as the posting job assigned it: {"length":"short"}.
                      An experiment or an arm the generator does not know fails the clip

Everything below is chosen from the replay when left out:
  --since, --until    the window searched, as ISO times; default the last 12 hours
  --hours <n>         the window's length, its busiest stretches
  --seconds <n>       the clip's length, at most 22; default 8.5 to 17.5 by how far the fighting moves (10 to 19 with a
                      day's podium), 6.5 to 19 for a tour by its stops. Not with the experiment "length"
  --focus <code>      only what changed hands on that country's ground
  --flags             come back out of the tiles halfway, to watch the painted flags change
  --dive              come back out of the tiles at the end
  --headline <text>   the headline, over the story's own
  --fps <n>           frames a second, default 30
  --recorders <n>     the browsers taking the frames in turn, default one for every 3 cores, 4 at most
  --headed            show the browser

Writes the video and, beside it, a .json with what to post (clips.v1.Clip), and a .txt with the headline said other
ways, one a line: the posting job writes the caption from them. Needs ffmpeg.
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
const RECORDERS = has("plan") || has("findings") ? 1 : Number(flag("recorders", Math.max(1, Math.min(4, Math.floor(availableParallelism() / 3)))))
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

const replayPath = resolve(replay)
if (!existsSync(replayPath)) {
    console.error(`no replay at ${replayPath}`)
    process.exit(1)
}
const optionalPath = (name) => {
    const path = flag(name) === undefined ? undefined : resolve(flag(name))
    if (path !== undefined && !existsSync(path)) {
        console.error(`no ${name} at ${path}`)
        process.exit(1)
    }
    return path
}
const racePath = optionalPath("race")
const seasonPath = optionalPath("season")
const castPath = optionalPath("cast")

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
            // Answered either way: with no race or no season, a day or a call-up is no story, with no cast nobody is named,
            // and the page must not read the app's fallback page.
            const optional = (path) => (_req, res) => {
                if (path === undefined) {
                    res.statusCode = 404
                    res.end()
                    return
                }
                res.setHeader("Content-Type", "application/json")
                res.setHeader("Content-Length", statSync(path).size)
                createReadStream(path).pipe(res)
            }
            vite.middlewares.use("/__clip/race.json", optional(racePath))
            vite.middlewares.use("/__clip/season.json", optional(seasonPath))
            vite.middlewares.use("/__clip/cast.json", optional(castPath))
        },
    }],
})
await server.listen()
const origin = server.resolvedUrls.local[0]

const launched = []
let ffmpeg
const done = async (code) => {
    ffmpeg?.kill()
    for (const {chrome} of launched) chrome.kill()
    await server.close()
    for (const {profile} of launched) {
        try {
            rmSync(profile, {recursive: true, force: true, maxRetries: 20, retryDelay: 100})
        } catch {}
    }
    process.exit(code)
}
process.on("SIGINT", () => done(130))

// A loaded runner starts Chrome slowly: a minute, where 20s failed a run on 2026-10-26.
const CHROME_START_TRIES = 240

async function debuggerUrl(profile) {
    for (let i = 0; i < CHROME_START_TRIES; i++) {
        try {
            const port = readFileSync(join(profile, "DevToolsActivePort"), "utf8").split("\n")[0]
            const targets = await fetch(`http://127.0.0.1:${port}/json/list`).then((r) => r.json())
            const page = targets.find((t) => t.type === "page")
            if (page) return page.webSocketDebuggerUrl
        } catch {}
        await sleep(250)
    }
    throw new Error("Chrome never exposed a debugging target")
}

async function openRecorder(speaks) {
    const profile = mkdtempSync(join(tmpdir(), "clickplanet-clip-"))
    const chrome = spawn(CHROME, [
        ...(has("headed") ? [] : ["--headless=new"]),
        "--remote-debugging-port=0",
        `--user-data-dir=${profile}`,
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
    launched.push({chrome, profile})

    const ws = new WebSocket(await debuggerUrl(profile))
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
        } else if (!speaks) {
            return
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

    await send("Page.enable")
    await send("Runtime.enable")
    await send("Emulation.setDeviceMetricsOverride", {
        width: WIDTH, height: HEIGHT, deviceScaleFactor: SCALE, mobile: false,
    })

    const open = async (url) => {
        loaded = false
        await send("Page.navigate", {url})
        for (let i = 0; i < 200 && !loaded; i++) await sleep(100)
    }

    return {
        async prepare(url) {
            await open(url)
            return evaluate("window.clip.record()")
        },
        async findings(url) {
            await open(url)
            return evaluate("window.clip.findings")
        },
        // Every recorder plays every frame, so each holds the same map, and takes the picture of its own only.
        async shoot(from, frame) {
            for (let i = from; i <= frame; i++) await evaluate(`window.clip.frame(${i})`)
            const {data} = await send("Page.captureScreenshot", {format: "png", optimizeForSpeed: true})
            return Buffer.from(data, "base64")
        },
    }
}

// The commit that makes the clip, kept in what the clip was: the workflow's, else the checkout's.
function commitOf() {
    if (process.env.GITHUB_SHA) return process.env.GITHUB_SHA
    try {
        return execFileSync("git", ["rev-parse", "HEAD"], {encoding: "utf8", stdio: ["ignore", "pipe", "ignore"]}).trim()
    } catch {
        return "unknown"
    }
}

const COMMIT = commitOf()

function pageOf() {
    const query = new URLSearchParams({fps: flag("fps", "30"), commit: COMMIT})
    for (const name of ["since", "until", "hours", "seconds", "focus", "headline", "variants", "story", "pick"]) {
        if (flag(name)) query.set(name, flag(name))
    }
    for (const name of ["flags", "dive", "silent"]) {
        if (has(name)) query.set(name, "")
    }
    return `${origin}clip.html?${query}`
}

async function record(recorders) {
    const [recording] = await Promise.all(recorders.map((recorder) => recorder.prepare(pageOf())))
    const out = has("out") ? resolve(flag("out")) : join(dirname(replayPath), `clip-${recording.pick}.mp4`)
    console.log([
        `#${recording.pick}: ${recording.headline}`,
        ...recording.line ? [`  ${recording.line}`] : [],
        `  window: ${recording.since} to ${recording.until}`,
        `  place: ${recording.place}, ${recording.seconds}s`,
        `  look: ${recording.look}`,
        `  music: ${recording.music ? `${recording.music.title}, the anthem of ${recording.music.whose}, from ${recording.music.from.toFixed(1)}s` : "none"}`,
        ...recording.fighters.length > 0
            ? [`  fighters: ${recording.fighters.map(({number, name, flag, took}) => `#${number} ${name} for ${flag} (${took} ${took === 1 ? "tile" : "tiles"})`).join(", ")}`] : [],
        ...Object.keys(recording.variants).length > 0
            ? [`  arms: ${Object.entries(recording.variants).map(([name, arm]) => `${name} ${arm}`).join(", ")}`] : [],
    ].join("\n"))
    if (has("plan")) return recording

    const frames = has("preview") ? Math.min(recording.frames, Math.round(Number(flag("preview")) * recording.fps)) : recording.frames
    const music = recording.music
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
    const shots = new Map()
    const shoot = (frame) => {
        if (frame >= frames) return
        const shot = recorders[frame % recorders.length].shoot(Math.max(0, frame - recorders.length + 1), frame)
        shot.catch(() => {})
        shots.set(frame, shot)
    }
    for (let frame = 0; frame < recorders.length; frame++) shoot(frame)
    for (let frame = 0; frame < frames; frame++) {
        const shot = await shots.get(frame)
        shots.delete(frame)
        shoot(frame + recorders.length)
        if (!ffmpeg.stdin.write(shot)) {
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

    const text = out.replace(/\.mp4$/, "") + ".txt"
    writeFileSync(text, recording.teasers.join("\n") + "\n")
    const clip = out.replace(/\.mp4$/, "") + ".json"
    writeFileSync(clip, JSON.stringify(recording.clip, null, 2) + "\n")
    console.log(`  saved ${out}, ${basename(text)} and ${basename(clip)}`)
    return recording
}

try {
    const recorders = await Promise.all(Array.from({length: RECORDERS}, (_, i) => openRecorder(i === 0)))

    if (has("findings")) {
        const findings = await recorders[0].findings(pageOf())
        writeFileSync(resolve(flag("findings")), JSON.stringify(findings, null, 2) + "\n")
        const count = (name) => findings[name]?.length ?? 0
        console.log(`${count("fronts")} fronts, ${count("tours")} tours, ${count("places")} places and ${count("seasonal")} seasonal stories, in ${flag("findings")}`)
        await done(0)
    }

    await record(recorders)
} catch (error) {
    console.error(error.message.split("\n").filter((line) => !line.trimStart().startsWith("at ")).join("\n"))
    await done(1)
}

await done(0)
