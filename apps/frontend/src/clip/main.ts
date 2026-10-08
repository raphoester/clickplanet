import "../tokens.css"
import "./clip.css"
import * as THREE from "three"
import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {ReplayBackend} from "../backends/replayBackend.ts"
import {createGlobe} from "../app/viewer/globe.ts"
import {loadPointGeometryData} from "../app/viewer/points.ts"
import {countryOfTile, loadBorders} from "../app/viewer/borderField.ts"
import {BORDERS_URL} from "../app/viewer/bordersAsset.ts"
import {Countries} from "../domain/countries.ts"
import {regionOf} from "../domain/regions.ts"
import {foughtOver, losersOf, ownersAfter, ranked as rankedBy, takersFrom, tally} from "../domain/clip/changes.ts"
import {cellOf, dot, Point, pointOf} from "../domain/clip/geometry.ts"
import {SCRIBBLE_BELOW, solidityOf} from "../domain/clip/solidity.ts"
import {anthemOf, startOf} from "../domain/clip/music.ts"
import {ANTHEMS} from "../app/anthem/anthemsAsset.ts"
import {HIGHLIGHTS} from "./anthemHighlightsAsset.ts"
import {CLIP_ANTHEMS} from "./clipAnthemsAsset.ts"
import {Candidate, candidatesOf, inCandidate, Window} from "../domain/clip/window.ts"
import {Front, frontOf, FRONT_RADIANS, sameFront, spanOf} from "../domain/clip/front.ts"
import {castOf, inPlace, placeOf, routOf, sameStory, Story, storyOf, THE_WORLD} from "../domain/clip/story.ts"
import {scoreOf} from "../domain/clip/score.ts"
import {Flip, flipsOf, Look, lookOf} from "../domain/clip/look.ts"
import {blastZoomOf, cameraOf, framingOf, openingOf, PullBack, screensOf} from "../domain/clip/camera.ts"
import {bombShareOf, momentsOf, paceOf, playedAt, timelineOf} from "../domain/clip/pace.ts"
import {tilesZoomOf} from "../app/viewer/pointSize.ts"
import {installVirtualClock} from "./virtualClock.ts"
import {createOverlay, nameOf, placeName, teamFlagOf, wordsOf} from "./overlay.ts"

export type Recording = {
    pick: number
    stories: number
    frames: number
    fps: number
    seconds: number
    since: string
    until: string
    look: string
    skipped: string[]
    // The anthem to play under the clip: where it is, its title, whose it is, and the credit its licence asks for.
    music: {url: string, title: string, whose: string, from: number, credit?: string} | undefined
    place: string
    headline: string
    line: string
    call: string
    link: string
    caption: string
}

export type Recorder = {
    ready: Promise<Recording>
    frame(index: number): void
}

declare global {
    interface Window {
        clip?: Recorder
    }
}

const WARM_UP_FRAMES = 3

const MARGIN_MS = 30_000

// A little past where the painted flags are gone, so the dive's detail is all tiles.
const DIVE_DEPTH = 1.1

const PULL_BACKS: Record<Look, PullBack> = {dive: "atEnd", flags: "midway"}

const clock = installVirtualClock()
const params = new URLSearchParams(location.search)
const fps = Number(params.get("fps") ?? 30)

let frame: (index: number) => void = () => {
    throw new Error("the clip is not ready yet")
}

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

function countryParam(name: string): string | undefined {
    const code = params.get(name) ?? undefined
    if (code !== undefined && !Countries.has(code)) throw new Error(`no country "${code}"`)
    return code
}

function numberParam(name: string): number | undefined {
    const value = params.get(name)
    return value === null ? undefined : Number(value)
}

// scope: the country whose ground alone the story is told on, or none for the whole map.
type Take = {candidate: Candidate, front: Front, story: Story, score: number, scope: string | undefined}

type Review = Take & {backend: ReplayBackend, solidity: number, skipped: string | undefined}

const RECORDED: Readonly<Record<string, {url: string, title: string, credit?: string}>> = {...ANTHEMS, ...CLIP_ANTHEMS}

function musicOf(story: Story, seconds: number): Recording["music"] {
    const code = anthemOf(story, (anthem) => anthem in RECORDED)
    if (code === undefined) return undefined
    const recording = RECORDED[code]
    return {...recording, whose: story.team ?? nameOf(code), from: startOf(HIGHLIGHTS[recording.url], seconds)}
}

function lookParam(): Look | undefined {
    return (["flags", "dive"] as const).find((look) => params.has(look))
}

function flipsLine(flips: readonly Flip[]): string {
    if (flips.length === 0) return "no landmass changed hands"
    const named = flips.slice(0, 4).map(({tiles, holders}) => `${holders.map((holder) => holder || "nobody").join(" to ")} (${tiles} tiles)`)
    return `landmasses changed hands: ${named.join(", ")}${flips.length > 4 ? ` and ${flips.length - 4} more` : ""}`
}

function windowParam(replay: ReplayBackend): Window | undefined {
    const since = params.get("since")
    const until = params.get("until")
    if (since === null && until === null) return undefined
    return {
        since: since === null ? replay.since : Math.max(replay.since, Date.parse(since)),
        until: until === null ? replay.until : Math.min(replay.until, Date.parse(until)),
    }
}

async function prepare(): Promise<Recording> {
    const textures = texturesLoaded()
    const root = document.getElementById("clip")
    const stage = document.getElementById("clip-globe")
    if (!root || !stage) throw new Error("clip.html is missing its elements")

    const answer = await fetch("/__clip/replay.json")
    if (!answer.ok) throw new Error(`no replay to play: ${answer.status}`)
    const replay = ReplayBackend.of(GetReplayResponse.fromJson(await answer.json(), {ignoreUnknownFields: true}))

    const attacker = countryParam("country")
    const focus = countryParam("focus")
    const {positions} = await loadPointGeometryData()
    const borders = await loadBorders(BORDERS_URL)
    const pointAt = (tile: number) => pointOf(positions, tile)
    const groundAt = (tile: number) => countryOfTile(borders, tile)
    const everything = replay.changes()
    const asked = windowParam(replay)
    const since = asked?.since ?? replay.since
    const until = asked?.until ?? replay.until
    // Every scale a story is told at: the whole map, and each country fought over the most, on its own ground.
    const scopes = focus !== undefined ? [focus] : [undefined, ...foughtOver(everything, groundAt, since, until)]
    const inScope = (scope: string | undefined) => (tile: number) => scope === undefined || groundAt(tile) === scope

    const takesIn = (scope: string | undefined): Take[] => {
        const all = everything.filter(({tile}) => inScope(scope)(tile))
        const cells = all.map(({tile}) => cellOf(pointAt(tile)))
        const takeOf = (candidate: Candidate): Take | undefined => {
            const inside = all.filter(({at}, i) =>
                at >= candidate.since && at <= candidate.until && inCandidate(candidate, cells[i]))
            // The fighting of some flags only, wherever it is densest: not another war next door.
            const fightOf = (sides: ReadonlySet<string | undefined>) => frontOf(inside.filter(({from, to}) =>
                (to !== undefined && sides.has(to)) || (from !== undefined && sides.has(from))), pointAt)
            const front = attacker === undefined ? frontOf(inside, pointAt) : fightOf(new Set([attacker]))
            const story = front && storyOf(front.changes, groundAt, regionOf, attacker)
            if (!front || !story) return undefined
            const own = fightOf(new Set([story.attacker, story.rival])) ?? front
            const told = storyOf(own.changes, groundAt, regionOf, story.attacker) ?? story
            const hours = (candidate.until - candidate.since) / 3_600_000
            return {candidate, front: own, story: told, score: scoreOf(own.changes, groundAt, hours), scope}
        }
        // A window asked for is still split into its places, so a war next door is a story of its own.
        const takes = candidatesOf(
            all.map(({at, from}, i) => ({at, cell: cells[i], captured: from !== undefined})),
            since, until, asked ? (until - since) / 3_600_000 : numberParam("hours"))
            .flatMap((candidate) => takeOf(candidate) ?? [])
            .sort((a, b) => b.score - a.score)
        return takes.filter((take, i) => takes.findIndex((other) =>
            other.story.attacker === take.story.attacker && sameFront(other.front, take.front)) === i)
    }
    const stories = scopes.flatMap(takesIn).sort((a, b) => b.score - a.score)

    // The flag that held most of a country's ground in a map.
    const holderIn = (owners: ReadonlyMap<number, string>) => (country: string) =>
        rankedBy(tally([...owners].flatMap(([tile, owner]) => groundAt(tile) === country ? [owner] : [])))[0]?.[0]
    const reviewOf = (take: Take): Review => {
        const trimmed = spanOf(take.front, MARGIN_MS)
        const backend = replay.cut(
            Math.max(take.candidate.since, trimmed.since), Math.min(take.candidate.until, trimmed.until))
        const told = storyOf(take.front.changes, groundAt, regionOf, attacker, holderIn(backend.opening)) ?? take.story
        const near = Math.cos(FRONT_RADIANS)
        const around = everything.filter(({tile, at}) => at >= backend.since && at <= backend.until
            && inScope(take.scope)(tile) && dot(pointAt(tile), take.front.heart) >= near)
        const cast = castOf(told, around, regionOf)
        // When nobody leads, the flag that lost the most around it may be the story, thrown out by all of them.
        const loser = cast?.victims[0] ?? losersOf(around)[0]
        // Told where the loser lost its land, not where its top taker took it.
        const lost = around.filter(({from, to}) => from === loser && to !== undefined)
        const base = cast ?? {
            ...told,
            attacker: takersFrom(around, loser ?? "")[0] ?? told.attacker,
            victims: [loser ?? told.victims[0], ...told.victims.filter((victim) => victim !== loser)],
            place: lost.length > 0 ? placeOf(lost.map(({tile}) => groundAt(tile)), regionOf) : told.place,
        }
        const after = ownersAfter(backend.opening, backend.changes())
        const heldBy = (owners: ReadonlyMap<number, string>) => [...owners].filter(([tile, owner]) =>
            owner === loser && inPlace(base.place, groundAt(tile), regionOf)).length
        const home = loser !== undefined && inPlace(base.place, loser, regionOf)
        const story = routOf(base, {before: heldBy(backend.opening), after: heldBy(after)}, cast !== undefined, home)
        const held: Point[] = []
        for (const [tile, owner] of after) if (owner === story.attacker) held.push(pointAt(tile))
        const taken = [...new Set(take.front.changes.flatMap(({tile, to}) =>
            to === story.attacker && after.get(tile) === story.attacker ? [tile] : []))]
        const solidity = solidityOf(taken.map(pointAt), held)
        const skipped = "region" in story.place && story.place.region === THE_WORLD
            ? "spread over several continents, no one place to show"
            : cast === undefined && story.kind !== "rout"
                ? "nobody leads it: its flags took too little of what changed hands around them"
                : solidity < SCRIBBLE_BELOW ? "lines drawn on someone else's land, not land taken" : undefined
        return {...take, story, backend, solidity, skipped}
    }
    const reviewed = stories.map(reviewOf)
    const kept = reviewed.filter(({skipped}) => skipped === undefined)
    // One story per flag and what it did, whatever window or scale found it: the best one.
    const worth = kept.filter((review, i) => kept.findIndex((other) => sameStory(other.story, review.story, placeName)) === i)
    const skipped = reviewed.flatMap(({story: told, skipped: why, solidity}) =>
        why === undefined ? [] : [`${wordsOf(told).headline}: ${why} (solidity ${solidity.toFixed(2)})`])

    const pick = numberParam("pick") ?? 1
    const picked = worth[pick - 1]
    if (!picked) {
        throw new Error([`no story number ${pick}: this replay has ${worth.length} worth a clip`, ...skipped].join("\n  skipped "))
    }
    const {front, story, backend, solidity, scope} = picked
    const changes = backend.changes().filter(({tile}) => inScope(scope)(tile))

    const reach = Math.cos(FRONT_RADIANS)
    const inArea = (tile: number) => scope === undefined ? dot(pointAt(tile), front.heart) >= reach : groundAt(tile) === scope

    const aspect = root.clientWidth / root.clientHeight
    const points = front.changes.map(({tile}) => pointAt(tile))
    const wide = framingOf(points, aspect)
    const touched = front.changes.map(({tile}) => tile)
    const flips = flipsOf(borders.assignment, backend.opening, changes, touched)
    const look = lookParam() ?? lookOf(flips, wide.zoom)
    const first = openingOf(wide)

    const drops = backend.drops().filter(({drop}) => drop.tile !== undefined && inArea(drop.tile))
    const close = Math.max(first.zoom, tilesZoomOf(root.clientHeight) * DIVE_DEPTH)
    const screens = screensOf(front.changes.map(({tile}, i) => ({share: i / front.changes.length, point: pointAt(tile)})), close)
    const timeline = timelineOf(screens, drops.length, numberParam("seconds"))
    const moments = momentsOf(front.changes.map(({at}) => at), drops.map(({at}) => at), bombShareOf(timeline))
    const pace = paceOf(moments, backend.since, backend.until)
    const camera = cameraOf(first, front.changes.map(({tile, at}) => ({share: pace.shareOf(at), point: pointAt(tile)})), {
        close,
        seconds: timeline.seconds - timeline.ending,
        pullBack: PULL_BACKS[look],
        hold: numberParam("hold"),
        blasts: drops.map(({at, drop}) => ({
            ...pace.heldOn(at), point: pointAt(drop.tile ?? 0), zoom: blastZoomOf(drop.radius, aspect),
        })),
    })

    // A continent striking back together is counted as one side.
    const team = teamFlagOf(story)
    const sideOf = (owner: string) => team !== undefined && regionOf(owner) === story.team ? team : owner
    const owners = new Map([...backend.opening].filter(([tile]) => inArea(tile)))
    const held = new Map<string, number>()
    for (const owner of owners.values()) held.set(sideOf(owner), (held.get(sideOf(owner)) ?? 0) + 1)
    const opening = new Map(held)
    const own = (tile: number, owner: string | undefined) => {
        if (!inArea(tile)) return
        const was = owners.get(tile)
        if (was !== undefined) held.set(sideOf(was), (held.get(sideOf(was)) ?? 0) - 1)
        if (owner === undefined) owners.delete(tile)
        else {
            owners.set(tile, owner)
            held.set(sideOf(owner), (held.get(sideOf(owner)) ?? 0) + 1)
        }
    }
    backend.listenForUpdates((update) => own(update.tile, update.newCountry))
    backend.listenForBombs((drop) => drop.cleared.forEach((tile) => own(tile, undefined)))

    const words = wordsOf(story, params.get("headline") ?? undefined)
    const overlay = createOverlay(root, story, words, opening)

    let played = 0
    await createGlobe({
        tileClicker: {clickTile: () => Promise.resolve()},
        ownershipsGetter: backend,
        updatesListener: backend,
        bonusListener: backend,
        bomber: backend,
        container: stage,
        country: Countries.get(story.attacker) ?? {code: story.attacker, name: story.attacker},
        mapView: "flags",
        rendering: "sharp",
        director: () => camera(played),
        onLeaderboardChange: () => {},
        onLoadProgress: () => {},
        onRateLimited: () => {},
        onVPNBlocked: () => {},
        onSessionUnavailable: () => {},
        onBonusTaken: () => {},
        onBonusWon: () => {},
        onCharges: () => {},
        onRules: () => {},
        onSwitchesChange: () => {},
        onBombDropped: () => {},
        onArmedChange: () => {},
        signal: new AbortController().signal,
    })

    await textures()
    await document.fonts.ready
    for (let i = 0; i < WARM_UP_FRAMES; i++) clock.advance(1000 / fps)

    frame = (index) => {
        const seconds = index / fps
        played = playedAt(timeline, seconds)
        const replayAt = pace.timeAt(played)
        backend.advanceTo(replayAt)
        clock.advance(1000 / fps)

        const ending = timeline.seconds - timeline.ending
        overlay.show({replayAt, played, held, ending: Math.min(1, Math.max(0, (seconds - ending) / 0.5))})
    }

    return {
        pick,
        stories: worth.length,
        frames: Math.round(timeline.seconds * fps),
        fps,
        seconds: timeline.seconds,
        since: new Date(backend.since).toISOString(),
        until: new Date(backend.until).toISOString(),
        look: `${look} (front framed at zoom ${wide.zoom.toFixed(1)}), ${drops.length} ${drops.length === 1 ? "bomb" : "bombs"}, `
            + `action crosses ${screens.toFixed(1)} screens, solidity ${solidity.toFixed(2)}, ${flipsLine(flips)}`,
        skipped,
        place: placeName(story.place),
        headline: words.headline,
        line: words.line ?? "",
        call: words.call,
        link: words.link,
        caption: words.caption,
        music: musicOf(story, timeline.seconds),
    }
}

window.clip = {ready: prepare(), frame: (index) => frame(index)}
