import "./isolate.ts"
import "../tokens.css"
import * as THREE from "three"
import {ReplayBackend, replayGroundsOf} from "../backends/replayBackend.ts"
import {MapReel} from "../domain/mapHistory.ts"
import {piecesOf} from "../domain/eras.ts"
import {scaleOf} from "../domain/boardScale.ts"
import {createGlobe, Globe} from "../app/viewer/globe.ts"
import {loadPointGeometryData} from "../app/viewer/points.ts"
import {countryOfTile, loadBorders, loadProvinces} from "../app/viewer/borderField.ts"
import {boardOf} from "../app/viewer/boards.ts"
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
    // Each era of the 24 hours on its own board, one globe each, built before the first frame: the one in force shows.
    // The story is of the board they end on (domain/video/story.ts).
    const eras = await Promise.all(piecesOf(job.history).map(async (piece) => {
        const board = boardOf(piece.board)
        const bordersRead = loadBorders(board.urls.borders)
        const [{positions}, borders, provinces] = await Promise.all([
            loadPointGeometryData(board), bordersRead, loadProvinces(board, bordersRead),
        ])
        const tiles = borders.assignment.length
        const backend = ReplayBackend.ofHistory(piece.history, new MapReel(piece.history, tiles), piece.history.frames[0]?.at ?? 0,
            replayGroundsOf(borders, provinces))
        const inCountry = (tile: number) => countryOfTile(borders, tile) === story.country
        const countryTiles = Array.from({length: tiles}, (_, i) => i + 1).filter(inCountry)
        return {piece, board, positions, borders, backend, inCountry, countryTiles}
    }))
    const last = eras.at(-1)
    if (last === undefined) throw new Error("the 24 hours hold no frame")
    const since = eras[0].backend.since
    const until = last.backend.until

    // The camera dives where the flag fights in the country and comes back out over it, as a clip's front does.
    const framed = framedTilesOf(last.countryTiles, (tile) => last.borders.assignment[tile - 1])
    const overview = overviewOf(framingOf(framed.map((tile) => pointOf(last.positions, tile)), width / height))
    const fought = eras.flatMap(({backend, inCountry, positions}) => backend.changes()
        .filter(({tile, from, to}) => (from === story.flag || to === story.flag) && inCountry(tile))
        .map(({tile, at}) => ({at, point: pointOf(positions, tile)})))
    const pace = paceOf(fought.map(({at}) => at), since, until)
    const timeline = timelineFor(VIDEO_PLAY_SECONDS, VIDEO_SECONDS, VIDEO_SECONDS - VIDEO_PLAY_SECONDS)
    const camera = cameraOf(overview, fought.map(({at, point}) => ({share: pace.shareOf(at), point})), {
        close: Math.max(overview.zoom, tilesZoomOf(height, scaleOf(last.board)) * DIVE_DEPTH),
        seconds: VIDEO_PLAY_SECONDS,
        pullBack: "atEnd",
    })

    // The counter counts on each era's board, as many tiles of the country on the board the story is of.
    const counted = eras.map(({backend, inCountry, countryTiles}) => {
        const count = {held: [...backend.opening].filter(([tile, owner]) => owner === story.flag && inCountry(tile)).length}
        backend.listenForUpdates(({tile, newCountry, previousCountry}) => {
            if (!inCountry(tile)) return
            if (previousCountry === story.flag) count.held--
            if (newCountry === story.flag) count.held++
        })
        return () => Math.round(count.held * story.tiles / Math.max(1, countryTiles.length))
    })

    let played = 0
    const globes: {globe: Globe, stage: HTMLElement}[] = []
    try {
        for (const {board, backend} of eras) {
            const own = document.createElement("div")
            own.className = "video-era"
            own.style.visibility = globes.length === 0 ? "visible" : "hidden"
            stage.appendChild(own)
            globes.push({stage: own, globe: await createGlobe({
                board,
                tileClicker: {clickTile: () => Promise.resolve()},
                ownershipsGetter: backend,
                updatesListener: backend,
                bonusListener: backend,
                bomber: backend,
                container: own,
                country: Countries.get(story.flag) ?? {code: story.flag},
                mapView: "split",
                provinces: false,
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
            })})
        }
    } catch (e) {
        for (const {globe} of globes) globe.dispose()
        throw e
    }
    try {
        await Promise.all(globes.map(({globe}) => globe.tilesRead))
        const look = lookOfThePage()
        const [sound, logo, flag] = await Promise.all([soundFor(story.flag), imageOf("/static/logo.svg"), flagOf(story.flag), fontsLoaded(look)])
        await textures()
        const pictures: Pictures = {logo, flag: flag.image, flagAspect: flag.aspect}

        const glCanvases = globes.map(({stage: own}) => own.querySelector("canvas"))
        if (glCanvases.some((glCanvas) => glCanvas === null)) throw new Error("the globe drew no canvas")
        const canvas = document.createElement("canvas")
        canvas.width = config.width
        canvas.height = config.height
        const context = canvas.getContext("2d")
        if (!context) throw new Error("this browser would not give a 2D context for the video")
        const overlay = createOverlay(context, job, look, pictures)
        const recorder = await recorderOf(config, sound)

        for (let i = 0; i < WARM_UP_FRAMES; i++) clock.advance(1000 / VIDEO_FPS)
        const frames = Math.round(VIDEO_SECONDS * VIDEO_FPS)
        let shown = 0
        let cutAt: number | undefined
        for (let frame = 0; frame < frames; frame++) {
            const seconds = frame / VIDEO_FPS
            played = playedAt(timeline, seconds)
            const at = pace.timeAt(played)
            // Past an era's start its board is shown, cleared and drawn whole: the cut.
            let era = shown
            while (era + 1 < eras.length && eras[era + 1].piece.startedAt <= at) era++
            if (era !== shown) {
                globes[shown].stage.style.visibility = "hidden"
                globes[era].stage.style.visibility = "visible"
                shown = era
                cutAt = seconds
            }
            eras[shown].backend.advanceTo(at)
            clock.advance(1000 / VIDEO_FPS)
            // In the same task as the render: the drawing buffer is cleared once the page is composited.
            context.drawImage(glCanvases[shown]!, 0, 0, canvas.width, canvas.height)
            overlay({seconds, played, held: counted[shown](), cut: cutAt === undefined ? undefined : seconds - cutAt})
            await recorder.add(canvas, frame)
            progress((frame + 1) / frames)
        }
        return {video: await recorder.finish(), silent: sound === undefined}
    } finally {
        for (const {globe, stage: own} of globes) {
            globe.dispose()
            own.remove()
        }
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
