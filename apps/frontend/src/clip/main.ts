import "../tokens.css"
import "./clip.css"
import * as THREE from "three"
import type {JsonValue} from "@bufbuild/protobuf"
import {Cast as CastWritten, Clip} from "../gen/grpc/clips/v1/clips_pb.ts"
import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {GetRoundResponse} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {GetTurnResponse} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {ReplayBackend} from "../backends/replayBackend.ts"
import {createGlobe} from "../app/viewer/globe.ts"
import {loadPointGeometryData} from "../app/viewer/points.ts"
import {countryOfTile, loadBorders} from "../app/viewer/borderField.ts"
import {BORDERS_URL} from "../app/viewer/bordersAsset.ts"
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
import {cameraOf, framingOf, overviewOf, PullBack, screensOf} from "../domain/clip/camera.ts"
import {paceOf, playedAt, PODIUM_ENDING_SECONDS, timelineFor, timelineOf} from "../domain/clip/pace.ts"
import {stopsOf, tourOf, tourSecondsOf, Visit} from "../domain/clip/tour.ts"
import {flagsZoomOf, tilesZoomOf} from "../app/viewer/pointSize.ts"
import {mulberry32} from "../app/viewer/stars.ts"
import {installVirtualClock} from "./virtualClock.ts"
import {createOverlay, hashtagsOf, nameOf, placeName, wordsOf} from "./overlay.ts"
import {findingsOf, FrontFound, Measured, PlaceFound, SeasonalFound, Told, toldOf, TourFound} from "./findings.ts"
import {clipRecordOf} from "./clipRecord.ts"
import {Variants, variantsOf} from "./experiments.ts"
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
    // The arm of each experiment the clip is in.
    variants: Variants
    // What to post with the clip, as a clips.v1.Clip.
    clip: JsonValue
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

    const focus = countryParam("focus")
    const {positions} = await loadPointGeometryData()
    const borders = await loadBorders(BORDERS_URL)
    const replay = ReplayBackend.of(recorded, {assignment: borders.assignment, count: borders.codes.length})
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
    const scopes = focus !== undefined ? [focus] : [undefined, ...foughtOver(everything, groundAt, since, until)]
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
        const flips = flipsOf(borders.assignment, backend.opening, changes, front.changes.map(({tile}) => tile))
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
        const flips = flipsOf(borders.assignment, backend.opening, changes, front.changes.map(({tile}) => tile))
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
    const places = focus !== undefined ? [] : placesOf(end, borders.assignment.length, groundAt, window, regionOf, castWithin(cast, since, until))
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
        : flipsOf(borders.assignment, ownersAfter(replay.opening, everything.filter(({at}) => at < since)), window, window.map(({tile}) => tile))
    const placeFound = (place: PlaceFacts): PlaceFound => {
        const flips = flippedInWindow.filter(({tile}) => inHolding(place.place, groundAt(tile), regionOf))
        return {...place, flippedByContinent: flippedByContinentOf(flips, place.holder, continentAt)}
    }
    // The clip of a day, or of a country called up after it, plays the run-up to its cutoff: its flag's fighting, else
    // everyone's. Nothing changed hands before the cutoff is no clip.
    const roundReviewOf = (story: Story): Review | undefined => {
        const cutoff = story.day?.endedAt ?? until
        const backend = replay.cut(Math.max(replay.since, cutoff - ROUND_PLAY_MS), cutoff)
        const played = backend.changes().filter(({from, to}) => from !== to)
        const own = played.filter(({from, to}) => from === story.attacker || to === story.attacker)
        const changes = own.length > 0 ? own : played
        if (changes.length === 0) return undefined
        const front = {heart: meanOf(changes.map(({tile}) => pointAt(tile))) ?? {x: 0, y: 0, z: 1}, changes}
        return {front, story, score: 0, scope: undefined, world: false, backend, solidity: 1, look: SEASONAL_LOOK}
    }
    const told = focus === undefined ? roundStoryOf(day, since, until) : undefined
    const seasonal = [...told === undefined ? [] : [told], ...focus === undefined ? callUpsOf(day, finaleAt, since, until) : []]
        .flatMap((story) => roundReviewOf(story) ?? [])

    // What a story is before any camera, for the findings and the clip's record alike: where it is counted, the counter
    // at its start and its end, and the effects in sight of it.
    const measuredOf = (review: Review) => {
        const {front, story, backend, solidity, scope, world, score} = review
        const changes = backend.changes().filter(({tile}) => inScope(scope)(tile))
        // The counter counts where the headline says: in the story's place, not in all that is near its front.
        const inArea = story.kind === "holding" ? (tile: number) => inHolding(story.place, groundAt(tile), regionOf)
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
    const reviewOf = (told: Told): Review => {
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

    const record = async (): Promise<Recording> => {
        // The arms the job put the clip in, read before anything is built: an arm nobody knows fails the clip.
        const variants = variantsOf(params.get("variants"))
        const pick = numberParam("pick") ?? 1
        const {told, written, fighters: chosen} = storyParam()
        const picked = reviewOf(told)
        const {front, story, backend, world, solidity} = picked
        const {measured, changes, inArea, placeTiles, sideOf, opening, key, drops} = measuredOf(picked)
        const touched = front.changes.map(({tile}) => tile)
        // The players named are counted on what the clip plays: the story's tiles, in the stretch it is cut to.
        const storyTiles = new Set(touched)
        const takes = castWithin(cast, backend.since, backend.until).takes.filter(({tile}) => storyTiles.has(tile))
        const named = namedSidesOf(cast, chosen, takes, sidesOf(story), sideOf)

        // A holding's counter is the share of the place its flag holds. A day's counter counts tiles, as any clip's,
        // and the podium says the points.
        const ground = story.kind === "holding" ? placeTiles : undefined
        // What the map is opens on all its flag holds there.
        const seasonal = story.kind === "round" || story.kind === "callup"
        const shared = story.kind === "holding" || seasonal
        const territory = shared ? [...ownersAfter(backend.opening, backend.changes())]
            .flatMap(([tile, owner]) => owner === story.attacker && inArea(tile) ? [pointAt(tile)] : []) : []

        const points = front.changes.map(({tile}) => pointAt(tile))
        const wide = framingOf(territory.length > 0 ? territory : points, aspect)
        const flips = flipsOf(borders.assignment, backend.opening, changes, touched)
        const look = lookParam() ?? picked.look
        const overview = overviewOf(wide)

        const close = Math.max(overview.zoom, tilesZoomOf(root.clientHeight) * DIVE_DEPTH)
        const screens = screensOf(front.changes.map(({tile}, i) => ({share: i / front.changes.length, point: pointAt(tile)})), close)
        const pace = paceOf(front.changes.map(({at}) => at), backend.since, backend.until)
        // A tour stays far out, where only a landmass's painted flag switching is seen: its stops are framed on the
        // landmasses that switched to the flag, and nowhere else.
        const switched = new Set(flippedTo(flips, story.attacker).map(({landmass}) => landmass))
        const stops = world ? stopsOf(visitsOf(front, story.attacker, pace.shareOf, (tile) => switched.has(borders.assignment[tile - 1])), aspect) : []
        if (world && stops.length === 0) throw new Error(`no painted flag switches to ${nameOf(story.attacker)}: a tour shows nothing`)
        const timeline = world ? timelineFor(tourSecondsOf(stops.length), numberParam("seconds"))
            : timelineOf(screens, numberParam("seconds"), seasonal ? PODIUM_ENDING_SECONDS : undefined)
        const camera = world ? tourOf(stops, aspect, timeline.seconds - timeline.ending, flagsZoomOf(root.clientHeight))
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

        const words = wordsOf(story, written, params.get("headline") ?? undefined)
        const overlay = createOverlay(root, story, words, opening, named, ground)
        // A seasonal story plays its flag's anthem: no editor chose one.
        const music = params.has("silent") ? undefined : musicOf(seasonal ? story.attacker : written?.anthem, timeline.seconds)
        const hashtags = hashtagsOf(story)

        let played = 0
        const globe = await createGlobe({
            tileClicker: {clickTile: () => Promise.resolve()},
            ownershipsGetter: backend,
            updatesListener: backend,
            bonusListener: backend,
            bomber: backend,
            container: stage,
            country: Countries.get(story.attacker) ?? {code: story.attacker, name: story.attacker},
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
            onBonusTaken: () => {},
            onBonusWon: () => {},
            onCharges: () => {},
            onRules: () => {},
            onSwitchesChange: () => {},
            onBombDropped: () => {},
            onArmedChange: () => {},
            signal: new AbortController().signal,
        })
        await globe.tilesRead

        await textures()
        await document.fonts.ready
        overlay.fit()
        for (let i = 0; i < WARM_UP_FRAMES; i++) clock.advance(1000 / fps)

        frame = (index) => {
            const seconds = index / fps
            played = playedAt(timeline, seconds)
            const replayAt = pace.timeAt(played)
            backend.advanceTo(replayAt)
            clock.advance(1000 / fps)

            const ending = timeline.seconds - timeline.ending
            overlay.show({replayAt, played, held, ending: seconds - ending})
        }

        const clipFighters = named.flatMap(({flag, fighters}) => fighters.map(({number, name, took}) => ({number, name, flag, took})))
        const clipRecord = clipRecordOf({
            ...measured, story, generator: params.get("commit") ?? "", seconds: timeline.seconds, music,
            look: world ? undefined : look, landmassesFlipped: flips.length, screens, tourStops: stops.length, variants,
            fighters: clipFighters.length,
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
                : `${look} (front framed at zoom ${wide.zoom.toFixed(1)}), ${drops.length} ${drops.length === 1 ? "bomb" : "bombs"}, `
                + `action crosses ${screens.toFixed(1)} screens, solidity ${solidity.toFixed(2)}, ${flipsLine(flips)}`,
            place: placeName(story.place),
            headline: words.headline,
            line: words.line ?? "",
            call: words.call,
            teasers: words.teasers,
            fighters: clipFighters,
            variants,
            clip: new Clip({
                pick, title: words.headline, key, teasers: words.teasers,
                countries: hashtags.countries, continentTag: hashtags.continent ?? "", credit: music?.credit ?? "", record: clipRecord,
                teaserNames: words.teaserNames,
                cast: clipFighters.length > 0 ? cast.id : "", fighters: clipFighters.map(({number, name, flag}) => ({number, name, flag})),
            }).toJson(),
            music,
        }
    }

    return {
        findings: findingsOf({
            since, until, fronts: fronts.map(frontFound), tours: toursFound, places: places.map(placeFound), seasonal: seasonalFound,
            anthems: Object.keys(RECORDED).sort(),
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
