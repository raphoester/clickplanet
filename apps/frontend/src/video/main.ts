import "./isolate.ts"
import "../tokens.css"
import * as THREE from "three"
import {ReplayBackend} from "../backends/replayBackend.ts"
import {MapReel} from "../domain/mapHistory.ts"
import {createGlobe} from "../app/viewer/globe.ts"
import {loadPointGeometryData} from "../app/viewer/points.ts"
import {countryOfTile, loadBorders} from "../app/viewer/borderField.ts"
import {BORDERS_URL} from "../app/viewer/bordersAsset.ts"
import {tilesZoomOf} from "../app/viewer/pointSize.ts"
import {Countries} from "../domain/countries.ts"
import {cameraOf, framingOf, overviewOf} from "../domain/clip/camera.ts"
import {pointOf} from "../domain/clip/geometry.ts"
import {paceOf, playedAt, timelineFor} from "../domain/clip/pace.ts"
import {framedTilesOf} from "../domain/clip/saga.ts"
import {startOf} from "../domain/clip/music.ts"
import {ANTHEMS} from "../app/anthem/anthemsAsset.ts"
import {HIGHLIGHTS} from "../clip/anthemHighlightsAsset.ts"
import {installVirtualClock} from "../clip/virtualClock.ts"
import {recorderOf, Sound, soundOf} from "./encode.ts"
import {encoderConfig, VIDEO_FPS, VIDEO_PLAY_SECONDS, VIDEO_SECONDS, VideoJob, VideoReply} from "./job.ts"
import {createOverlay, lookOfThePage, Pictures} from "./overlay.ts"

// A little past where the painted flags are gone, as the clips dive.
const DIVE_DEPTH = 1.1

const WARM_UP_FRAMES = 3

const clock = installVirtualClock()

function texturesLoaded(): () => Promise<void> {
    const manager = THREE.DefaultLoadingManager
    let loaded = Promise.resolve()
    manager.onStart = () => {
        loaded = new Promise((resolve) => {
            manager.onLoad = resolve
        })
    }
    return () => loaded
}

function imageOf(url: string): Promise<HTMLImageElement> {
    return new Promise((resolve, reject) => {
        const image = new Image()
        image.onload = () => resolve(image)
        image.onerror = () => reject(new Error(`${url} would not load`))
        image.src = url
    })
}

// A flag's SVG has a viewBox and no size, which a canvas will not draw everywhere: it is given one.
async function flagOf(code: string): Promise<{image: HTMLImageElement, aspect: number}> {
    const answer = await fetch(`/static/countries/svg/${code}.svg`)
    if (!answer.ok) throw new Error(`no flag for ${code}`)
    const svg = await answer.text()
    const box = /viewBox="[\d.-]+ [\d.-]+ ([\d.]+) ([\d.]+)"/.exec(svg)
    const aspect = box ? Number(box[1]) / Number(box[2]) : 3 / 2
    const sized = svg.replace("<svg ", `<svg width="${Math.round(480 * aspect)}" height="480" preserveAspectRatio="none" `)
    const url = URL.createObjectURL(new Blob([sized], {type: "image/svg+xml"}))
    try {
        return {image: await imageOf(url), aspect}
    } finally {
        URL.revokeObjectURL(url)
    }
}

async function fontsLoaded(look: ReturnType<typeof lookOfThePage>) {
    await Promise.all([
        document.fonts.load(`40px ${look.display}`),
        document.fonts.load(`700 20px ${look.text}`),
        document.fonts.load(`800 20px ${look.text}`),
        document.fonts.load(`900 20px ${look.text}`),
    ])
    await document.fonts.ready
}

// The flag's anthem from its best highlight, as the clips start it; none where the game has no recording, and a silent
// video when it cannot be read.
async function soundFor(flag: string): Promise<Sound | undefined> {
    const recording = ANTHEMS[flag]
    if (recording === undefined) return undefined
    try {
        return await soundOf(recording.url, startOf(HIGHLIGHTS[recording.url], VIDEO_SECONDS), VIDEO_SECONDS)
    } catch (e) {
        console.error("could not read the anthem: the video is silent", e)
        return undefined
    }
}

async function make(job: VideoJob, progress: (share: number) => void): Promise<{video: Blob, silent: boolean}> {
    const config = await encoderConfig()
    if (config === undefined) throw new Error("this browser cannot encode H.264")
    const textures = texturesLoaded()
    const stage = document.getElementById("video-globe")
    if (!stage) throw new Error("video.html is missing its stage")
    const {width, height} = stage.getBoundingClientRect()

    const {story} = job
    const [{positions}, borders] = await Promise.all([loadPointGeometryData(), loadBorders(BORDERS_URL)])
    const tiles = borders.assignment.length
    const landmasses = {assignment: borders.assignment, count: borders.codes.length}
    const since = job.history.frames[0]?.at ?? 0
    const backend = ReplayBackend.ofHistory(job.history, new MapReel(job.history, tiles), since, landmasses)
    const pointAt = (tile: number) => pointOf(positions, tile)
    const inCountry = (tile: number) => countryOfTile(borders, tile) === story.country

    // The camera dives where the flag fights in the country and comes back out over it, as a clip's front does.
    const countryTiles = Array.from({length: tiles}, (_, i) => i + 1).filter(inCountry)
    const framed = framedTilesOf(countryTiles, (tile) => borders.assignment[tile - 1])
    const overview = overviewOf(framingOf(framed.map(pointAt), width / height))
    const fought = backend.changes().filter(({tile, from, to}) => (from === story.flag || to === story.flag) && inCountry(tile))
    const pace = paceOf(fought.map(({at}) => at), backend.since, backend.until)
    const timeline = timelineFor(VIDEO_PLAY_SECONDS, VIDEO_SECONDS, VIDEO_SECONDS - VIDEO_PLAY_SECONDS)
    const camera = cameraOf(overview, fought.map(({tile, at}) => ({share: pace.shareOf(at), point: pointAt(tile)})), {
        close: Math.max(overview.zoom, tilesZoomOf(height) * DIVE_DEPTH),
        seconds: VIDEO_PLAY_SECONDS,
        pullBack: "atEnd",
    })

    let held = [...backend.opening].filter(([tile, owner]) => owner === story.flag && inCountry(tile)).length
    backend.listenForUpdates(({tile, newCountry, previousCountry}) => {
        if (!inCountry(tile)) return
        if (previousCountry === story.flag) held--
        if (newCountry === story.flag) held++
    })

    let played = 0
    const globe = await createGlobe({
        tileClicker: {clickTile: () => Promise.resolve()},
        ownershipsGetter: backend,
        updatesListener: backend,
        bonusListener: backend,
        bomber: backend,
        container: stage,
        country: Countries.get(story.flag) ?? {code: story.flag},
        mapView: "split",
        rendering: "sharp",
        director: () => camera(played),
        onLeaderboardChange: () => {},
        onLoadProgress: () => {},
        onRateLimited: () => {},
        onFrozen: () => {},
        onVPNBlocked: () => {},
        onBanned: () => {},
        onSessionUnavailable: () => {},
        onAllegianceLocked: () => {},
        onOutOfReach: () => {},
        onBonusWon: () => {},
        onCharges: () => {},
        onRules: () => {},
        onSwitchesChange: () => {},
        onBombDropped: () => {},
        onArmedChange: () => {},
        signal: new AbortController().signal,
    })
    try {
        await globe.tilesRead
        const look = lookOfThePage()
        const [sound, logo, flag] = await Promise.all([soundFor(story.flag), imageOf("/static/logo.svg"), flagOf(story.flag), fontsLoaded(look)])
        await textures()
        const pictures: Pictures = {logo, flag: flag.image, flagAspect: flag.aspect}

        const glCanvas = stage.querySelector("canvas")
        if (!glCanvas) throw new Error("the globe drew no canvas")
        const canvas = document.createElement("canvas")
        canvas.width = config.width
        canvas.height = config.height
        const context = canvas.getContext("2d")
        if (!context) throw new Error("this browser would not give a 2D context for the video")
        const overlay = createOverlay(context, job, look, pictures)
        const recorder = await recorderOf(config, sound)

        for (let i = 0; i < WARM_UP_FRAMES; i++) clock.advance(1000 / VIDEO_FPS)
        const frames = Math.round(VIDEO_SECONDS * VIDEO_FPS)
        for (let frame = 0; frame < frames; frame++) {
            const seconds = frame / VIDEO_FPS
            played = playedAt(timeline, seconds)
            backend.advanceTo(pace.timeAt(played))
            clock.advance(1000 / VIDEO_FPS)
            // In the same task as the render: the drawing buffer is cleared once the page is composited.
            context.drawImage(glCanvas, 0, 0, canvas.width, canvas.height)
            overlay({seconds, played, held})
            await recorder.add(canvas, frame)
            progress((frame + 1) / frames)
        }
        return {video: await recorder.finish(), silent: sound === undefined}
    } finally {
        globe.dispose()
    }
}

window.addEventListener("message", (event: MessageEvent<{kind: "job", job: VideoJob}>) => {
    if (event.origin !== location.origin || event.data?.kind !== "job") return
    const [port] = event.ports
    if (!port) return
    const reply = (message: VideoReply) => port.postMessage(message)
    make(event.data.job, (share) => reply({kind: "progress", share})).then(
        ({video, silent}) => reply({kind: "done", video, silent}),
        (e) => reply({kind: "failed", message: e instanceof Error ? e.message : String(e)}),
    )
})

window.parent.postMessage({kind: "ready"}, location.origin)
