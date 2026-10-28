import {Timestamp} from "@bufbuild/protobuf"
import {Findings, Front, Pick, Place as PlaceWritten, Seasonal, StoryFacts, Tour} from "../gen/grpc/clips/v1/clips_pb.ts"
import {FrontFacts, PlaceFacts, TourFacts} from "../domain/clip/findings.ts"
import {KINDS, Place, Story, storyCountriesOf, storyKeyOf} from "../domain/clip/story.ts"
import {continents, flagOfContinent, regionOf} from "../domain/regions.ts"
import {nameOf, placeName, said, wordsOf, Written} from "./overlay.ts"
import {languageOf} from "./clipRecord.ts"

// What the generator measured of a story to find it, before any camera: the numbers its clip would record.
export type Measured = {
    world: boolean
    since: number
    until: number
    tilesChanged: number
    storyTilesChanged: number
    placeTiles: number
    flagTiles: {start: number, end: number}
    bombs: number
    spreads: number
    enclosures: number
    solidity: number
    score: number
}

export type Window = {since: number, until: number}

export type FrontFound = Window & FrontFacts & {scope: string | undefined, bombs: number, spreads: number, enclosures: number, score: number}

export type TourFound = Window & TourFacts & {flag: string, nextTook: number, bombs: number}

export type SeasonalFound = {story: Story, measured: Measured}

export type Found = {
    since: number
    until: number
    fronts: FrontFound[]
    tours: TourFound[]
    places: PlaceFacts[]
    seasonal: SeasonalFound[]
    anthems: string[]
}

const rounded = (value: number, places: number) => Math.round(value * 10 ** places) / 10 ** places

const at = (ms: number) => Timestamp.fromDate(new Date(ms))

// Everything the editor makes the run's stories from, as a clips.v1.Findings: the fronts, the tours, the places, the
// stories the season's data tells, and how every code in them is said.
export function findingsOf(found: Found): Findings {
    const codes = new Set<string>()
    const noted = (...more: (string | undefined)[]) => more.forEach((code) => code !== undefined && codes.add(code))
    for (const front of found.fronts) {
        for (const country of front.countries) noted(country.code, ...Object.keys(country.holdersStart), ...Object.keys(country.holdersEnd), ...Object.keys(country.takers))
        for (const flag of front.flags) noted(flag.code, ...Object.keys(flag.tookFrom))
    }
    for (const tour of found.tours) noted(tour.flag, ...Object.keys(tour.tookFrom))
    for (const place of found.places) noted(place.code, place.holder, ...Object.keys(place.rivals))
    for (const {story} of found.seasonal) noted(story.attacker, story.rival, ...story.victims, ...storyCountriesOf(story))

    return new Findings({
        since: at(found.since),
        until: at(found.until),
        fronts: found.fronts.map((front) => new Front({
            scope: front.scope ?? "", since: at(front.since), until: at(front.until), tilesChanged: front.tilesChanged, captured: front.captured,
            countries: front.countries.map((country) => ({
                code: country.code, continent: country.continent ?? "", tiles: country.tiles, changed: country.changed,
                holdersStart: country.holdersStart, holdersEnd: country.holdersEnd, takers: country.takers,
            })),
            flags: front.flags.map((flag) => ({
                code: flag.code, continent: flag.continent ?? "", took: flag.took, filled: flag.filled, lost: flag.lost,
                tookFrom: flag.tookFrom, solidity: rounded(flag.solidity, 2),
            })),
            bombs: front.bombs, spreads: front.spreads, enclosures: front.enclosures, score: rounded(front.score, 1),
        })),
        tours: found.tours.map((tour) => new Tour({
            flag: tour.flag, since: at(tour.since), until: at(tour.until), took: tour.took, byContinent: tour.byContinent,
            tookFrom: tour.tookFrom, nextTook: tour.nextTook, solidity: rounded(tour.solidity, 2), bombs: tour.bombs,
            flippedByContinent: tour.flippedByContinent,
        })),
        places: found.places.map((place) => new PlaceWritten({
            kind: place.kind, code: place.code ?? "", continent: place.continent ?? "", tiles: place.tiles, holder: place.holder,
            held: place.held, share: rounded(place.share, 3), rivals: place.rivals, changed: place.changed,
        })),
        seasonal: found.seasonal.map(({story, measured}) => {
            const words = wordsOf(story)
            return new Seasonal({
                title: words.headline, key: storyKeyOf(story, regionOf), countries: storyCountriesOf(story),
                facts: storyFactsOf(story, words.line, measured), teasers: words.teasers,
            })
        }),
        names: Object.fromEntries([...codes].sort().map((code) => [code, said(nameOf(code))])),
        continents: continents().map((name) => ({name, flag: flagOfContinent(name) ?? ""})),
        anthems: found.anthems,
    })
}

export function storyFactsOf(story: Story, line: string | undefined, measured: Measured): StoryFacts {
    const codes = [...new Set([story.attacker, story.rival, ...story.victims, ...storyCountriesOf(story)])]
        .filter((code): code is string => code !== undefined)
    return new StoryFacts({
        kind: story.kind,
        place: placeName(story.place),
        camera: measured.world ? "tour" : "front",
        attacker: story.attacker,
        rival: story.rival ?? "",
        takenFrom: story.victims,
        names: Object.fromEntries(codes.map((code) => [code, said(nameOf(code))])),
        line: line ?? "",
        language: languageOf(story),
        spanMinutes: rounded((measured.until - measured.since) / 60_000, 1),
        storyEndedAt: at(measured.until),
        tilesChanged: measured.tilesChanged,
        storyTilesChanged: measured.storyTilesChanged,
        placeTiles: measured.placeTiles,
        flagTilesStart: measured.flagTiles.start,
        flagTilesEnd: measured.flagTiles.end,
        bombs: measured.bombs,
        spreads: measured.spreads,
        enclosures: measured.enclosures,
        solidity: rounded(measured.solidity, 2),
        share: rounded(story.share ?? 0, 3),
        score: rounded(measured.score, 1),
    })
}

// What --story names: one of the findings, and for a front what the editor made of it.
export type Told =
    | {front: number, story: Story}
    | {tour: number}
    | {place: number}
    | {seasonal: number}

// The story the editor wrote, as a clips.v1.Pick says it, with its words; a seasonal story has words of its own. Nothing
// is judged here: a pick that names no front, no kind, no flag or no words is refused as it is read, and the job checked
// it against the findings before the render.
export function toldOf(json: string): {told: Told, written: Written | undefined} {
    const pick = Pick.fromJson(JSON.parse(json), {ignoreUnknownFields: true})
    const named = [pick.front, pick.tour, pick.place, pick.seasonal].filter((number) => number > 0)
    if (named.length !== 1) throw new Error("a story names one front, one tour, one place or one seasonal story")
    if (pick.seasonal > 0) return {told: {seasonal: pick.seasonal}, written: undefined}
    if (pick.headline === "" || pick.call === "") throw new Error("a story the editor tells has its headline and its call")
    const written: Written = {
        headline: pick.headline, teasers: pick.teasers, line: pick.line === "" ? undefined : pick.line, call: pick.call,
        callFlags: pick.callFlags, anthem: pick.anthem === "" ? undefined : pick.anthem,
    }
    if (pick.tour > 0) return {told: {tour: pick.tour}, written}
    if (pick.place > 0) return {told: {place: pick.place}, written}

    const kind = KINDS.find((known) => known === pick.kind)
    if (kind === undefined || kind === "holding" || kind === "round" || kind === "callup") {
        throw new Error(`a front's story is an attack, an invasion, a comeback, a kickout, a rout or a battle, not "${pick.kind}"`)
    }
    if (pick.attacker === "") throw new Error("a front's story has an attacker")
    const place = placeOfPick(pick.countries, pick.continent)
    const story: Story = {
        kind, attacker: pick.attacker, rival: pick.rival === "" ? undefined : pick.rival, victims: pick.takenFrom, place,
        ...pick.team === "" ? {} : {team: pick.team},
    }
    return {told: {front: pick.front, story}, written}
}

function placeOfPick(countries: string[], continent: string): Place {
    if (countries.length === 1) return {country: countries[0]}
    if (countries.length === 2) return {countries: [countries[0], countries[1]]}
    if (countries.length === 0 && continent !== "") return {region: continent}
    throw new Error("a story is told in one or two countries, or on a continent")
}
