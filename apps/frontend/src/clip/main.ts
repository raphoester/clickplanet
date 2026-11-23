import "../tokens.css"
import "./clip.css"
import * as THREE from "three"
import type {JsonValue} from "@bufbuild/protobuf"
import {Cast as CastWritten, Clip} from "../gen/grpc/clips/v1/clips_pb.ts"
import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {GetMapHistoryResponse} from "../gen/grpc/planet/v1/internal_pb.ts"
import {GetRoundResponse} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {GetTurnResponse} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {replayErasOf, ReplayBackend, replayGroundsOf} from "../backends/replayBackend.ts"
import {takenOf} from "../backends/planetBackend.ts"
import {historyOf} from "../backends/turnBackend.ts"
import {MapReel} from "../domain/mapHistory.ts"
import {lastEraOf} from "../domain/eras.ts"
import {scaleOf} from "../domain/boardScale.ts"
import {createGlobe} from "../app/viewer/globe.ts"
import {loadPointGeometryData} from "../app/viewer/points.ts"
import {countryOfTile, loadBorders, loadProvinces} from "../app/viewer/borderField.ts"
import {boardOf} from "../app/viewer/boards.ts"
import {Countries} from "../domain/countries.ts"
import {continents, flagOfContinent, regionOf} from "../domain/regions.ts"
import {foughtOver, ownersAfter, ranked as rankedBy, tally, TileChange} from "../domain/clip/changes.ts"
import {cellOf, meanOf, pointOf} from "../domain/clip/geometry.ts"
import {startOf} from "../domain/clip/music.ts"
import {ANTHEMS} from "../app/anthem/anthemsAsset.ts"
import {HIGHLIGHTS} from "./anthemHighlightsAsset.ts"
import {CLIP_ANTHEMS} from "./clipAnthemsAsset.ts"
import {Candidate, candidatesOf, inCandidate, newsSince, Window} from "../domain/clip/window.ts"
import {Front, frontOf, inSightOf, sameFront, spanOf} from "../domain/clip/front.ts"
import {flippedByContinentOf, frontFactsOf, Ground, holdingStoryOf, inHolding, PlaceFacts, placesOf, solidityAfter, tourFactsOf} from "../domain/clip/findings.ts"
import {ROUND_PLAY_MS, roundStoryOf} from "../domain/clip/round.ts"
import {callUpsOf} from "../domain/clip/callup.ts"
import {castWithin, namedSidesOf, NO_CAST} from "../domain/clip/cast.ts"
import {fightersOf, inPlace, sidesOf, Story, storyKeyOf, teamFlagOf, THE_WORLD} from "../domain/clip/story.ts"
import {scoreOf} from "../domain/clip/score.ts"
import {Flip, flippedTo, flipsOf, Look, LOOKS} from "../domain/clip/look.ts"
import {blendedPaceOf, paceOf, playedAt, PODIUM_ENDING_SECONDS, timelineFor, timelineOf} from "../domain/clip/pace.ts"
import {framedTilesOf, pathsOf, sagasOf} from "../domain/clip/saga.ts"
import {stopsOf, tourOf, tourSecondsOf, Visit} from "../domain/clip/tour.ts"
import {Said, subtitlesOf, VOICE_AT_SECONDS, voicedTimelineOf} from "../domain/clip/voice.ts"
import {flagsZoomOf, tilesZoomOf} from "../app/viewer/pointSize.ts"
import {mulberry32} from "../app/viewer/stars.ts"
import {installVirtualClock} from "./virtualClock.ts"
import {createOverlay, hashtagsOf, placeName, tagOf, wordsOf} from "./overlay.ts"
import {nameOf} from "./names.ts"
import {findingsOf, FrontFound, Measured, PlaceFound, SeasonalFound, Told, toldOf, TourFound} from "./findings.ts"
import {carouselRecordOf, clipRecordOf} from "./clipRecord.ts"
import {aimedTilesOf, codeOf, entriesOf, LIST_LENGTH, ListGround, ListName, listsOf} from "../domain/clip/lists.ts"
import {createCarousel, photosOf, Rank} from "./carousel.ts"
import {cameraOf, framingOf, overviewOf, PullBack, sagaCameraOf, screensOf, Shot, widestZoomOf} from "../domain/clip/camera.ts"
import {dayOf, finaleAtOf} from "./race.ts"
import {castOf} from "./cast.ts"

export type Recording = {
    pick: number
    frames: number
    fps: number
    seconds: number
    since: string
    until: string
    look: string
    // The anthem to play under the clip: where it is, its title, whose it is, its country's code, and the credit its
    // licence asks for.
    music: {url: string, title: string, whose: string, code: string, from: number, credit?: string} | undefined
    place: string
    headline: string
    line: string
    call: string
    teasers: string[]
    // The players the end card names, with the flag each fought for and the tiles it took.
    fighters: {number: number, name: string, flag: string, took: number}[]
    // What to post with the clip, as a clips.v1.Clip.
    clip: JsonValue
    // A carousel's photos, each a still frame, and how long the slideshow shows each; none for a video.
    photos: number
    secondsPerPhoto: number
    // A war report's voice: when it starts in the clip, how long it says, and the pieces of its subtitles.
    voice: {at: number, seconds: number, subtitles: number} | undefined
}

export type Recorder = {
    // What the replay holds, for the editor to make the stories from, as a clips.v1.Findings.
    findings: Promise<JsonValue>
    record(): Promise<Recording>
    frame(index: number): void
}

type Chosen = {findings: JsonValue, record(): Promise<Recording>}

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

// A day won and a call-up are the generator's own, their look as their words: no editor tells them.
const SEASONAL_LOOK: Look = "dive"

// The flags taking the most across the map in a window that may be on a tour of the world.
const WORLD_TAKERS = 3

// The busiest fronts, whatever window found them: enough for a day of war, few enough to read.
const MOST_FRONTS = 30

// A saga plays this long before its call: a place's days, its conquests and its falls.
const SAGA_PLAY_SECONDS = 11

// The share of a saga's play spent evenly on the calendar, so the days a flag held still go by on the clock.
const SAGA_CALENDAR = 0.4

// How long before each change a saga's clock starts to run into it: enclosures a minute apart flow, none freezes it.
const SAGA_LEAD_MS = 5 * 60_000

// What fight-the-most counts: the last 3 days of the replay, or all of it when it is shorter.
const FOUGHT_MS = 72 * 3_600_000

// A carousel's slideshow on YouTube, as the teasers'.
const SECONDS_PER_PHOTO = 3

// The frames a still is drawn over before it is taken, so the painted flags and the tiles are in.
const STILL_FRAMES = 8

// A rank's place framed a little wider, so its neighbours say where on the globe it is.
const RANK_CONTEXT = 0.7

const clock = installVirtualClock()
// Several recorders take one clip's frames in turn: each has to draw the same debris and the same shake.
Math.random = mulberry32(1)
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

// The overlay's flags and logo drawn, so fit() measures each flag at its own shape.
function picturesLoaded(root: HTMLElement): Promise<unknown> {
    return Promise.all([...root.querySelectorAll("img")].map((image) => image.decode().catch(() => undefined)))
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

// A front of all the fighting in one stretch of time, before any story: the changes of the window around it (inside),
// the ones near where most changed hands (front), and how busy it was. scope: the country whose ground alone it is
// measured on, or none for the whole map.
type Spotted = {scope: string | undefined, candidate: Candidate, inside: TileChange[], front: Front, score: number}

type SpottedTour = {candidate: Candidate, flag: string, front: Front, score: number, nextTook: number}

// What a clip is made of: the fighting the camera follows, the story told over it, the replay cut to it, and when the
// camera comes back out of the tiles.
type Review = {
    front: Front, story: Story, score: number, scope: string | undefined, world: boolean, backend: ReplayBackend, solidity: number, look: Look
}

const RECORDED: Readonly<Record<string, {url: string, title: string, credit?: string}>> = {...ANTHEMS, ...CLIP_ANTHEMS}

function musicOf(code: string | undefined, seconds: number): Recording["music"] {
    if (code === undefined || !(code in RECORDED)) return undefined
    const recording = RECORDED[code]
    const whose = continents().find((continent) => flagOfContinent(continent) === code) ?? nameOf(code)
    return {...recording, whose, code, from: startOf(HIGHLIGHTS[recording.url], seconds)}
}

function lookParam(): Look | undefined {
    return LOOKS.find((look) => params.has(look))
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

// The lines of a war report's voice, as voice.mjs timed them.
function voiceParam(): Said[] | undefined {
    const json = params.get("voice")
    if (json === null) return undefined
    const lines: unknown = JSON.parse(json)
    const read = Array.isArray(lines) && lines.length > 0
        && lines.every((line) => typeof line?.text === "string" && typeof line.from === "number" && typeof line.to === "number")
    if (!read) throw new Error(`a voice's lines are each a text, a from and a to: ${json.slice(0, 200)}`)
    return lines
}

function storyParam(): ReturnType<typeof toldOf> {
    const json = params.get("story")
    if (json === null) throw new Error("no story to tell: --story names one, and --findings lists what the replay holds")
    return toldOf(json)
}

async function choose(): Promise<Chosen> {
    const textures = texturesLoaded()
    const root = document.getElementById("clip")
    const stage = document.getElementById("clip-globe")
    if (!root || !stage) throw new Error("clip.html is missing its elements")

    const answer = await fetch("/__clip/replay.json")
    if (!answer.ok) throw new Error(`no replay to play: ${answer.status}`)
    const recorded = GetReplayResponse.fromJson(await answer.json(), {ignoreUnknownFields: true})
    // The gamemaster's turn and the round it debriefs, when the run fetched them: that round is a story of the run whose
    // window holds its end, and the countries that can still win are called up to the finale.
    const calendar = await fetch("/__clip/season.json")
    const turn = calendar.ok ? GetTurnResponse.fromJson(await calendar.json(), {ignoreUnknownFields: true}) : undefined
    const race = await fetch("/__clip/race.json")
    const round = race.ok ? GetRoundResponse.fromJson(await race.json(), {ignoreUnknownFields: true}).round : undefined
    const day = dayOf(turn?.debrief, round)
    const finaleAt = finaleAtOf(turn?.timeline)
    // Who took which tiles, when the run made a cast: without one, a clip names nobody.
    const roster = await fetch("/__clip/cast.json")
    const cast = roster.ok ? castOf(CastWritten.fromJson(await roster.json(), {ignoreUnknownFields: true})) : NO_CAST
    // The map every hour since the season started, when the run fetched it: each place's saga is told from it.
    const mapped = await fetch("/__clip/history.json")
    const kept = mapped.ok ? GetMapHistoryResponse.fromJson(await mapped.json(), {ignoreUnknownFields: true}).history : undefined
    const whole = kept === undefined ? undefined : historyOf(kept)

    // A clip tells the map the replay ends on, on its board: over a change of board, from the change on, as after a
    // reset. The sagas too, when the history ends on the same board.
    const board = boardOf(replayErasOf(recorded).at(-1)!.board)
    const scale = scaleOf(board)
    const ended = whole === undefined ? undefined : lastEraOf(whole)
    const history = ended?.board === board.name ? ended : undefined
    if (ended !== undefined && history === undefined) console.error(`the history ends on ${ended.board}, the replay on ${board.name}: no saga`)

    const focus = countryParam("focus")
    const {positions} = await loadPointGeometryData(board)
    const bordersRead = loadBorders(board.urls.borders)
    const [borders, provinces] = await Promise.all([bordersRead, loadProvinces(board, bordersRead)])
    const grounds = replayGroundsOf(borders, provinces)
    const replay = ReplayBackend.of(recorded, grounds)
    const pointAt = (tile: number) => pointOf(positions, tile)
    const groundAt = (tile: number) => countryOfTile(borders, tile)
    const continentAt = (tile: number) => {
        const ground = groundAt(tile)
        return ground === undefined ? undefined : regionOf(ground)
    }
    const tilesByCountry = new Map<string, number[]>()
    for (let tile = 1; tile <= borders.assignment.length; tile++) {
        const ground = groundAt(tile)
        if (ground === undefined) continue
        const tiles = tilesByCountry.get(ground)
        if (tiles) tiles.push(tile)
        else tilesByCountry.set(ground, [tile])
    }
    const ground: Ground = {groundOf: groundAt, tilesOf: (country) => tilesByCountry.get(country) ?? [], regionOf}
    const aspect = root.clientWidth / root.clientHeight
    const everything = replay.changes()
    const asked = windowParam(replay)
    const since = asked?.since ?? newsSince(replay.since, replay.until)
    const until = asked?.until ?? replay.until
    const hours = asked ? (until - since) / 3_600_000 : numberParam("hours")
    const inScope = (scope: string | undefined) => (tile: number) => scope === undefined || groundAt(tile) === scope
    // Every scale a front is measured at: the whole map, and each country fought over the most, on its own ground.
    const scopes = focus !== undefined ? [focus] : [undefined, ...foughtOver(everything, groundAt, since, until, scale)]
    const hoursOf = (candidate: Candidate) => (candidate.until - candidate.since) / 3_600_000
    // The replay from the front's first change to its last, within the window that found it.
    const cutOf = (front: Front, candidate: Candidate) => {
        const trimmed = spanOf(front, MARGIN_MS)
        return replay.cut(Math.max(candidate.since, trimmed.since), Math.min(candidate.until, trimmed.until))
    }

    // Every front of a scope, the busiest first: one per place, whatever window found it.
    const frontsIn = (scope: string | undefined): Spotted[] => {
        const all = everything.filter(({tile}) => inScope(scope)(tile))
        const cells = all.map(({tile}) => cellOf(pointAt(tile)))
        const spotted = candidatesOf(all.map(({at, from}, i) => ({at, cell: cells[i], captured: from !== undefined})), since, until, hours)
            .flatMap((candidate): Spotted[] => {
                const inside = all.filter(({at}, i) => at >= candidate.since && at <= candidate.until && inCandidate(candidate, cells[i]))
                const front = frontOf(inside, pointAt)
                return front ? [{scope, candidate, inside, front, score: scoreOf(front.changes, groundAt, hoursOf(candidate))}] : []
            })
            .sort((a, b) => b.score - a.score)
        return spotted.filter((one, i) => spotted.findIndex((other) => sameFront(other.front, one.front)) === i)
    }
    const fronts = scopes.flatMap(frontsIn).sort((a, b) => b.score - a.score).slice(0, MOST_FRONTS)
    const frontFound = ({scope, candidate, front, score}: Spotted): FrontFound => {
        const backend = cutOf(front, candidate)
        const changes = backend.changes()
        const flips = flipsOf(borders.assignment, backend.opening, changes, front.changes.map(({tile}) => tile), scale)
        const facts = frontFactsOf(front.changes, backend.opening, ownersAfter(backend.opening, changes), ground, pointAt, flips,
            castWithin(cast, backend.since, backend.until))
        const inSight = inSightOf(front, pointAt, groundAt, scope)
        const effects = backend.effects().filter(({tile}) => inSight(tile))
        return {
            scope, since: backend.since, until: backend.until, ...facts,
            bombs: backend.drops().filter(({drop}) => drop.tile !== undefined && inSight(drop.tile)).length,
            spreads: effects.filter(({kind}) => kind === "spread").length,
            enclosures: effects.filter(({kind}) => kind === "enclosure").length, score,
        }
    }

    // The flags taking the most across the whole map in each stretch, once each, in their busiest stretch.
    const sweeps = candidatesOf(everything.map(({at, from}) => ({at, cell: 0, captured: from !== undefined})), since, until, hours)
        .flatMap((candidate): SpottedTour[] => {
            const inside = everything.filter(({at}) => at >= candidate.since && at <= candidate.until)
            const takers = rankedBy(tally(inside.flatMap(({from, to}) => from === undefined || to === undefined || to === from ? [] : [to])))
            return takers.slice(0, WORLD_TAKERS).flatMap(([flag]) => {
                const changes = inside.filter(({from, to}) => to === flag || from === flag)
                const heart = meanOf(changes.flatMap(({tile, to}) => to === flag ? [pointAt(tile)] : []))
                if (heart === undefined) return []
                const nextTook = takers.find(([other]) => other !== flag)?.[1] ?? 0
                return [{candidate, flag, front: {heart, changes}, score: scoreOf(changes, groundAt, hoursOf(candidate)), nextTook}]
            })
        })
        .sort((a, b) => b.score - a.score)
    const tours = focus === undefined ? sweeps.filter((one, i) => sweeps.findIndex((other) => other.flag === one.flag) === i) : []
    const tourFound = ({candidate, flag, front, nextTook}: SpottedTour): TourFound => {
        const backend = cutOf(front, candidate)
        const changes = backend.changes()
        const flips = flipsOf(borders.assignment, backend.opening, changes, front.changes.map(({tile}) => tile), scale)
        const facts = tourFactsOf(flag, front.changes, ownersAfter(backend.opening, changes), continentAt, pointAt, flips,
            castWithin(cast, backend.since, backend.until))
        return {since: backend.since, until: backend.until, ...facts, flag, nextTook, bombs: backend.drops().filter(({drop}) => drop.tile !== undefined).length}
    }
    // on: the tiles a visit counts on, when not all of them.
    const visitsOf = (front: Front, flag: string, shareOf: (at: number) => number, on: (tile: number) => boolean = () => true): Visit[] =>
        front.changes.flatMap(({tile, to, at}) => to === flag && on(tile) ? [{share: shareOf(at), point: pointAt(tile), region: continentAt(tile)}] : [])

    // What the map is, beside what changed on it: the flag holding the most of each place, as the map stands at the end.
    const window = everything.filter(({at}) => at >= since && at <= until)
    const end = ownersAfter(replay.opening, everything.filter(({at}) => at <= until))
    const places = focus !== undefined ? [] : placesOf(end, borders.assignment.length, groundAt, window, regionOf, castWithin(cast, since, until), scale)
    // A carousel's lists, of the map at the end, and of the last 3 days for fight-the-most.
    const foughtSince = Math.max(replay.since, until - FOUGHT_MS)
    const fights = everything.filter(({at}) => at > foughtSince && at <= until)
    const listGround: ListGround = {owners: end, tiles: borders.assignment.length, groundOf: groundAt, pointOf: pointAt}
    const lists = focus !== undefined ? {fightTheMost: [], farFromHome: [], biggestEmpires: []} : listsOf(listGround, fights, scale)
    // The clip of a holding plays its flag's fighting for it, all over the place.
    const holdingReviewOf = (story: Story, look: Look): Review => {
        const changes = window.filter(({tile, from, to}) => from !== to && (from === story.attacker || to === story.attacker)
            && inHolding(story.place, groundAt(tile), regionOf))
        const front = {heart: meanOf(changes.map(({tile}) => pointAt(tile))) ?? {x: 0, y: 0, z: 1}, changes}
        const span = spanOf(front, MARGIN_MS)
        const backend = replay.cut(Math.max(since, span.since), Math.min(until, span.until))
        return {front, story, score: 0, scope: undefined, world: false, backend, solidity: solidityAfter(story.attacker, changes, end, pointAt), look}
    }
    // A landmass is one country's: the window's flips, read once, are each place's.
    const flippedInWindow = places.length === 0 ? []
        : flipsOf(borders.assignment, ownersAfter(replay.opening, everything.filter(({at}) => at < since)), window, window.map(({tile}) => tile), scale)
    // Each place over the season: who held a tenth of it and who held over half of it, from when to when; and the land of
    // each flag that held over half of a place, on each continent.
    const sagas = focus !== undefined || history === undefined ? [] : sagasOf(history, borders.assignment.length, groundAt, regionOf, scale)
    const reigning = new Set(sagas.flatMap(({reigns}) => reigns.map(({flag}) => flag)))
    const paths = history === undefined ? [] : pathsOf(history, borders.assignment.length, reigning, groundAt, regionOf)
    // The clip of a saga plays its place from the time the editor chose to the end of the history.
    const sagaReviewOf = (told: {saga: number, since: number, attacker: string, rival: string | undefined}): Review => {
        const found = sagas[told.saga - 1]
        if (!found || history === undefined) throw new Error(`no saga number ${told.saga}: the findings hold ${sagas.length}`)
        if (told.since < found.since || told.since >= found.until) {
            throw new Error(`saga ${told.saga} runs from ${new Date(found.since).toISOString()} to ${new Date(found.until).toISOString()}, `
                + `not from ${new Date(told.since).toISOString()}`)
        }
        const backend = ReplayBackend.ofHistory(history, new MapReel(history, borders.assignment.length), told.since, grounds)
        const inSaga = (tile: number) => inHolding(found.place, groundAt(tile), regionOf)
        // With no rival, the flag holding the most of the place after the attacker when the clip starts.
        const [next] = rankedBy(tally([...backend.opening].flatMap(([tile, owner]) => owner !== told.attacker && inSaga(tile) ? [owner] : [])))
        const victims = told.rival !== undefined ? [told.rival] : next === undefined ? [] : [next[0]]
        const story: Story = {kind: "saga", attacker: told.attacker, rival: told.rival, victims, place: found.place, share: found.share}
        const sides = fightersOf(story, regionOf)
        const played = backend.changes()
        const changes = played.filter(({tile, from, to}) => (sides(from) || sides(to)) && inSaga(tile))
        if (changes.length === 0) {
            throw new Error(`neither side of the saga of ${placeName(story.place)} held any of it from ${new Date(told.since).toISOString()}`)
        }
        const front = {heart: meanOf(changes.map(({tile}) => pointAt(tile))) ?? {x: 0, y: 0, z: 1}, changes}
        const solidity = solidityAfter(story.attacker, changes, ownersAfter(backend.opening, played), pointAt)
        return {front, story, score: 0, scope: undefined, world: false, backend, solidity, look: "flags"}
    }
    const placeFound = (place: PlaceFacts): PlaceFound => {
        const flips = flippedInWindow.filter(({tile}) => inHolding(place.place, groundAt(tile), regionOf))
        return {...place, flippedByContinent: flippedByContinentOf(flips, place.holder, continentAt)}
    }
    // The clip of a day, or of a country called up after it, plays its flag's takes before the cutoff where it took the
    // most (anyone's when it took none): never its losses, never a flight across an ocean. No change is no clip.
    const roundReviewOf = (story: Story): Review | undefined => {
        const cutoff = story.day?.endedAt ?? until
        const backend = replay.cut(Math.max(replay.since, cutoff - ROUND_PLAY_MS), cutoff)
        const played = backend.changes().filter(({from, to}) => from !== to)
        const taken = played.filter(({to}) => to === story.attacker)
        const front = frontOf(taken.length > 0 ? taken : played, pointAt)
        if (front === undefined) return undefined
        return {front, story, score: 0, scope: undefined, world: false, backend, solidity: 1, look: SEASONAL_LOOK}
    }
    const asides = recorded.asides.map(({openedAt, closedAt}) => ({
        openedAt: openedAt?.toDate().getTime() ?? 0, closedAt: closedAt?.toDate().getTime(),
    }))
    const told = focus === undefined ? roundStoryOf(day, since, until, asides) : undefined
    const seasonal = [...told === undefined ? [] : [told], ...focus === undefined ? callUpsOf(day, finaleAt, since, until) : []]
        .flatMap((story) => roundReviewOf(story) ?? [])

    // What a story is before any camera, for the findings and the clip's record alike: where it is counted, the counter
    // at its start and its end, and the effects in sight of it.
    const measuredOf = (review: Review) => {
        const {front, story, backend, solidity, scope, world, score} = review
        const changes = backend.changes().filter(({tile}) => inScope(scope)(tile))
        // The counter counts where the headline says: in the story's place, not in all that is near its front.
        const inArea = story.kind === "holding" || story.kind === "saga" ? (tile: number) => inHolding(story.place, groundAt(tile), regionOf)
            : (tile: number) => inPlace(story.place, groundAt(tile), regionOf)
        const placeTiles = Array.from({length: borders.assignment.length}, (_, i) => i + 1).filter(inArea).length
        // A continent striking back together is held as one side.
        const team = teamFlagOf(story)
        const sideOf = (owner: string) => team !== undefined && regionOf(owner) === story.team ? team : owner
        const opening = new Map<string, number>()
        for (const [tile, owner] of backend.opening) if (inArea(tile)) opening.set(sideOf(owner), (opening.get(sideOf(owner)) ?? 0) + 1)
        const key = storyKeyOf(story, regionOf)
        const flagTilesAtEnd = [...ownersAfter(backend.opening, backend.changes())]
            .filter(([tile, owner]) => inArea(tile) && sideOf(owner) === key.flag).length
        const inSight = world ? () => true : inSightOf(front, pointAt, groundAt, scope)
        const drops = backend.drops().filter(({drop}) => drop.tile !== undefined && inSight(drop.tile))
        const effects = backend.effects().filter(({tile}) => inSight(tile))
        const measured: Measured = {
            world, since: backend.since, until: backend.until, tilesChanged: changes.length,
            storyTilesChanged: front.changes.length, placeTiles, flagTiles: {start: opening.get(key.flag) ?? 0, end: flagTilesAtEnd},
            bombs: drops.length, spreads: effects.filter(({kind}) => kind === "spread").length,
            enclosures: effects.filter(({kind}) => kind === "enclosure").length, solidity, score,
        }
        return {measured, changes, inArea, placeTiles, sideOf, opening, key, drops}
    }
    const seasonalFound = seasonal.map((review): SeasonalFound => ({story: review.story, measured: measuredOf(review).measured}))
    const toursFound = tours.map(tourFound)

    // The clip of the story --story names: the fighting of its sides where it is told, the story as the editor wrote it.
    const reviewOf = (told: Exclude<Told, {carousel: unknown}>): Review => {
        if ("tour" in told) {
            const found = tours[told.tour - 1]
            const facts = toursFound[told.tour - 1]
            if (!found || !facts) throw new Error(`no tour number ${told.tour}: the findings hold ${tours.length}`)
            const story: Story = {kind: "attack", attacker: found.flag, rival: undefined, victims: Object.keys(facts.tookFrom), place: {region: THE_WORLD}}
            // A tour stays out among the painted flags all along: the look is never read.
            return {
                front: found.front, story, score: found.score, scope: undefined, world: true, backend: cutOf(found.front, found.candidate),
                solidity: facts.solidity, look: "flags",
            }
        }
        if ("place" in told) {
            const found = places[told.place - 1]
            if (!found) throw new Error(`no place number ${told.place}: the findings hold ${places.length}`)
            return holdingReviewOf(holdingStoryOf(found), told.look)
        }
        if ("saga" in told) return sagaReviewOf(told)
        if ("seasonal" in told) {
            const found = seasonal[told.seasonal - 1]
            if (!found) throw new Error(`no seasonal story number ${told.seasonal}: the findings hold ${seasonal.length}`)
            return found
        }
        const found = fronts[told.front - 1]
        if (!found) throw new Error(`no front number ${told.front}: the findings hold ${fronts.length}`)
        const {story} = told
        const fighters = fightersOf(story, regionOf)
        const own = frontOf(found.inside.filter(({tile, from, to}) =>
            (fighters(to) || fighters(from)) && inPlace(story.place, groundAt(tile), regionOf)), pointAt)
        if (!own) throw new Error(`no side of the ${story.kind} of ${nameOf(story.attacker)} fought in ${placeName(story.place)} on front ${told.front}`)
        const backend = cutOf(own, found.candidate)
        const solidity = solidityAfter(story.attacker, own.changes, ownersAfter(backend.opening, backend.changes()), pointAt)
        return {front: own, story, score: found.score, scope: found.scope, world: false, backend, solidity, look: told.look}
    }

    // The game's globe over a replay, its camera the clip's: country, the flag it plays for.
    const globeOf = (backend: ReplayBackend, country: string, director: () => Shot) => createGlobe({
        board,
        tileClicker: {clickTile: () => Promise.resolve()},
        ownershipsGetter: backend,
        updatesListener: backend,
        bonusListener: backend,
        bomber: backend,
        container: stage,
        country: Countries.get(country) ?? {code: country},
        mapView: "split",
        provinces: false,
        rendering: "sharp",
        director,
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

    // The map at the end, still: each rank aimed at its place no closer than its painted flags, one photo a frame.
    const carouselOf = async (pick: number, told: {list: ListName, hook: string, ranks: Rank[]}, experiment: string): Promise<Recording> => {
        const entries = entriesOf(lists, told.list)
        const photos = photosOf(told.list, entries, told.ranks, told.hook)
        const aimed = entries.slice(0, LIST_LENGTH).map((entry) => {
            return framingOf(aimedTilesOf(entry, listGround, fights).map(pointAt), aspect)
        })
        const widest = widestZoomOf(aspect)
        const flagsZoom = flagsZoomOf(root.clientHeight, scale)
        const whole = {direction: meanOf(aimed.map(({direction}) => direction)) ?? aimed[0].direction, zoom: widest}
        const shots: Shot[] = [
            whole,
            ...aimed.map(({direction, zoom}) => ({direction, zoom: Math.max(widest, Math.min(zoom * RANK_CONTEXT, flagsZoom))})).reverse(),
            {direction: aimed[0].direction, zoom: widest},
        ]
        const backend = replay.cut(until + 1, until)
        let shown = 0
        const first = codeOf(entries[0])
        const globe = await globeOf(backend, first, () => shots[shown])
        await globe.tilesRead
        await textures()
        await document.fonts.ready
        const carousel = createCarousel(root, photos)
        await picturesLoaded(root)
        carousel.fit()
        carousel.show(0)

        frame = (index) => {
            shown = index
            carousel.show(index)
            for (let i = 0; i < STILL_FRAMES; i++) clock.advance(1000 / fps)
        }

        const codes = told.ranks.map(({code}) => code)
        const countries = codes.filter((code) => Countries.has(code)).map((code) => ({code, tag: tagOf(nameOf(code))}))
        return {
            pick, frames: 0, fps, seconds: photos.length * SECONDS_PER_PHOTO,
            since: new Date(foughtSince).toISOString(), until: new Date(until).toISOString(),
            look: `carousel of ${told.list}: ${photos.length} photos, ${photos.flatMap((photo) => photo.kind === "rank" ? [`#${photo.rank} ${photo.name} ${photo.numbers.join(" ")}`] : []).join(", ")}`,
            music: undefined, place: told.list, headline: told.hook, line: "", call: "", teasers: [told.hook], fighters: [],
            clip: new Clip({
                pick, title: told.hook, key: {flag: first, continent: THE_WORLD, angle: "list"}, teasers: [told.hook], teaserNames: ["editor"],
                countries, record: carouselRecordOf(told.list, codes, photos.length, SECONDS_PER_PHOTO, params.get("commit") ?? ""),
                experiment, list: told.list,
            }).toJson(),
            photos: photos.length,
            secondsPerPhoto: SECONDS_PER_PHOTO,
            voice: undefined,
        }
    }

    const record = async (): Promise<Recording> => {
        const pick = numberParam("pick") ?? 1
        const {told, written, fighters: chosen, experiment} = storyParam()
        if ("carousel" in told) return carouselOf(pick, told.carousel, experiment)
        const picked = reviewOf(told)
        const {front, story, backend, world, solidity} = picked
        const {measured, changes, inArea, placeTiles, sideOf, opening, key, drops} = measuredOf(picked)
        const touched = front.changes.map(({tile}) => tile)
        // The players named are counted on what the clip plays: the story's tiles, in the stretch it is cut to.
        const storyTiles = new Set(touched)
        const takes = castWithin(cast, backend.since, backend.until).takes.filter(({tile}) => storyTiles.has(tile))
        const named = namedSidesOf(cast, chosen, takes, sidesOf(story), sideOf)

        // A holding's and a saga's counters are the share of the place each flag holds. A day's counter counts tiles, as
        // any clip's, and the podium says the points.
        const saga = story.kind === "saga"
        const ground = story.kind === "holding" || saga ? placeTiles : undefined
        // What the map is opens on all its flag holds there.
        const seasonal = story.kind === "round" || story.kind === "callup"
        const shared = story.kind === "holding" || seasonal
        const territory = shared ? [...ownersAfter(backend.opening, backend.changes())]
            .flatMap(([tile, owner]) => owner === story.attacker && inArea(tile) ? [pointAt(tile)] : []) : []

        const points = front.changes.map(({tile}) => pointAt(tile))
        // A saga is framed on its place, a country on its main landmasses.
        const placed = saga ? Array.from({length: borders.assignment.length}, (_, i) => i + 1).filter(inArea) : []
        const framed = "country" in story.place ? framedTilesOf(placed, (tile) => borders.assignment[tile - 1]) : placed
        const wide = framingOf(saga ? framed.map(pointAt) : territory.length > 0 ? territory : points, aspect)
        const flips = flipsOf(borders.assignment, backend.opening, changes, touched, scale)
        const look = lookParam() ?? picked.look
        const overview = overviewOf(wide)

        const close = Math.max(overview.zoom, tilesZoomOf(root.clientHeight, scale) * DIVE_DEPTH)
        const screens = screensOf(front.changes.map(({tile}, i) => ({share: i / front.changes.length, point: pointAt(tile)})), close)
        const pace = saga ? blendedPaceOf(front.changes.map(({at}) => at), backend.since, backend.until, SAGA_CALENDAR, SAGA_LEAD_MS)
            : paceOf(front.changes.map(({at}) => at), backend.since, backend.until)
        // A tour stays far out, where only a landmass's painted flag switching is seen: its stops are framed on the
        // landmasses that switched to the flag, and nowhere else.
        const switched = new Set(flippedTo(flips, story.attacker).map(({landmass}) => landmass))
        const stops = world ? stopsOf(visitsOf(front, story.attacker, pace.shareOf, (tile) => switched.has(borders.assignment[tile - 1])), aspect) : []
        if (world && stops.length === 0) throw new Error(`no painted flag switches to ${nameOf(story.attacker)}: a tour shows nothing`)
        // A war report is a clip given a voice: it lasts as long as the voice says.
        const said = voiceParam()
        const timeline = said !== undefined ? voicedTimelineOf(said, seasonal ? PODIUM_ENDING_SECONDS : undefined)
            : world ? timelineFor(tourSecondsOf(stops.length), numberParam("seconds"))
            : saga ? timelineFor(SAGA_PLAY_SECONDS, numberParam("seconds"))
            : timelineOf(screens, numberParam("seconds"), seasonal ? PODIUM_ENDING_SECONDS : undefined)
        const camera = world ? tourOf(stops, aspect, timeline.seconds - timeline.ending, flagsZoomOf(root.clientHeight, scale))
            : saga ? sagaCameraOf(wide, flagsZoomOf(root.clientHeight, scale))
            : cameraOf(overview, front.changes.map(({tile, at}) => ({share: pace.shareOf(at), point: pointAt(tile)})), {
                close,
                seconds: timeline.seconds - timeline.ending,
                pullBack: PULL_BACKS[look],
            })

        const owners = new Map([...backend.opening].filter(([tile]) => inArea(tile)))
        const held = new Map(opening)
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
        backend.listenForBonuses({
            onEnclosed: (enclosure) => takenOf(enclosure).forEach((tile) => own(tile, enclosure.countryId)),
            onVirusSpread: (spread) => spread.taken.forEach((tile) => own(tile, spread.countryId)),
            onMinesExploded: (explosion) => explosion.blasts.forEach((blast) => blast.taken.forEach((tile) => own(tile, blast.countryId))),
            onMines: () => {}, onMineLaid: () => {}, onTraps: () => {}, onTrapLaid: () => {}, onTrapSprung: () => {}, onRallies: () => {}, onRallyCalled: () => {}, onSpread: () => {}, onCharges: () => {}, onRules: () => {},
        })

        const subtitles = said === undefined ? [] : subtitlesOf(said)
        const words = {...wordsOf(story, written, params.get("headline") ?? undefined), ...said === undefined ? {} : {subtitles}}
        const overlay = createOverlay(root, story, words, opening, named, ground)
        // A seasonal story plays its flag's anthem: no editor chose one.
        const music = params.has("silent") ? undefined : musicOf(seasonal ? story.attacker : written?.anthem, timeline.seconds)
        const hashtags = hashtagsOf(story)

        let played = 0
        const globe = await globeOf(backend, story.attacker, () => camera(played))
        await globe.tilesRead

        await textures()
        await document.fonts.ready
        await picturesLoaded(root)
        overlay.fit()
        for (let i = 0; i < WARM_UP_FRAMES; i++) clock.advance(1000 / fps)

        frame = (index) => {
            const seconds = index / fps
            played = playedAt(timeline, seconds)
            const replayAt = pace.timeAt(played)
            backend.advanceTo(replayAt)
            clock.advance(1000 / fps)

            const ending = timeline.seconds - timeline.ending
            overlay.show({seconds, replayAt, played, held, ending: seconds - ending})
        }

        const clipFighters = named.flatMap(({flag, fighters}) => fighters.map(({number, name, took}) => ({number, name, flag, took})))
        const clipRecord = clipRecordOf({
            ...measured, story, generator: params.get("commit") ?? "", seconds: timeline.seconds, music,
            camera: world ? "tour" : saga ? "saga" : "front", look: world || saga ? undefined : look,
            landmassesFlipped: flips.length, screens, tourStops: stops.length,
            fighters: clipFighters.length, line: words.line,
            ...written?.language === undefined ? {} : {language: written.language},
        })

        return {
            pick,
            frames: Math.round(timeline.seconds * fps),
            fps,
            seconds: timeline.seconds,
            since: new Date(backend.since).toISOString(),
            until: new Date(backend.until).toISOString(),
            look: world ? `tour of ${stops.map(({region}) => region).join(", ")}, ${drops.length} ${drops.length === 1 ? "bomb" : "bombs"}, `
                + `solidity ${solidity.toFixed(2)}`
                : saga ? `saga of ${placeName(story.place)} over ${((backend.until - backend.since) / 86_400_000).toFixed(1)} days, `
                + `framed at zoom ${wide.zoom.toFixed(1)}, ${flipsLine(flips)}`
                : `${look} (front framed at zoom ${wide.zoom.toFixed(1)}), ${drops.length} ${drops.length === 1 ? "bomb" : "bombs"}, `
                + `action crosses ${screens.toFixed(1)} screens, solidity ${solidity.toFixed(2)}, ${flipsLine(flips)}`,
            place: placeName(story.place),
            headline: words.headline,
            line: words.line ?? "",
            call: words.call,
            teasers: words.teasers,
            fighters: clipFighters,
            clip: new Clip({
                pick, title: words.headline, key, teasers: words.teasers,
                countries: hashtags.countries, continentTag: hashtags.continent ?? "", credit: music?.credit ?? "", record: clipRecord,
                teaserNames: words.teaserNames,
                cast: clipFighters.length > 0 ? cast.id : "", fighters: clipFighters.map(({number, name, flag}) => ({number, name, flag})),
                experiment,
            }).toJson(),
            music,
            photos: 0,
            secondsPerPhoto: 0,
            voice: said === undefined ? undefined : {at: VOICE_AT_SECONDS, seconds: said.at(-1)?.to ?? 0, subtitles: subtitles.length},
        }
    }

    return {
        findings: findingsOf({
            since, until, fronts: fronts.map(frontFound), tours: toursFound, places: places.map(placeFound), sagas, paths,
            seasonal: seasonalFound, anthems: Object.keys(RECORDED).sort(), lists, foughtSince,
            continentTiles: Object.fromEntries(tally(Array.from({length: borders.assignment.length}, (_, i) => continentAt(i + 1) ?? []).flat())),
        }).toJson(),
        record,
    }
}

const chosen = choose()
window.clip = {
    findings: chosen.then(({findings}) => findings),
    record: () => chosen.then(({record}) => record()),
    frame: (index) => frame(index),
}
