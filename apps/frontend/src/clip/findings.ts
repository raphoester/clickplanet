import {Timestamp} from "@bufbuild/protobuf"
import {Contender as ContenderWritten, Findings, Front, Path as PathWritten, Pick, Place as PlaceWritten, Reign as ReignWritten, Saga, Seasonal, StoryFacts, Tour} from "../gen/grpc/clips/v1/clips_pb.ts"
import {Counts, FrontFacts, PlaceFacts, TourFacts} from "../domain/clip/findings.ts"
import {KINDS, Place, Story, storyCountriesOf, storyKeyOf} from "../domain/clip/story.ts"
import {Look, LOOKS} from "../domain/clip/look.ts"
import {Path, SagaFacts} from "../domain/clip/saga.ts"
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

// flippedByContinent: the tiles of the landmasses whose painted flag switched to the holder in the window.
export type PlaceFound = PlaceFacts & {flippedByContinent: Counts}

export type SeasonalFound = {story: Story, measured: Measured}

export type Found = {
    since: number
    until: number
    fronts: FrontFound[]
    tours: TourFound[]
    places: PlaceFound[]
    sagas: SagaFacts[]
    paths: Path[]
    seasonal: SeasonalFound[]
    anthems: string[]
    // Each continent's size in tiles.
    continentTiles: Counts
}

const rounded = (value: number, places: number) => Math.round(value * 10 ** places) / 10 ** places

const at = (ms: number) => Timestamp.fromDate(new Date(ms))

const atMaybe = (ms: number | undefined) => ms === undefined ? undefined : at(ms)

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
    for (const saga of found.sagas) {
        noted(saga.code, saga.holder, ...saga.reigns.flatMap(({flag, tookFrom, lostTo}) => [flag, tookFrom, lostTo]))
        noted(...saga.contenders.map(({flag}) => flag))
    }
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
                tookFrom: flag.tookFrom, solidity: rounded(flag.solidity, 2), flippedByContinent: flag.flippedByContinent,
                fighters: flag.fighters,
            })),
            bombs: front.bombs, spreads: front.spreads, enclosures: front.enclosures, score: rounded(front.score, 1),
        })),
        tours: found.tours.map((tour) => new Tour({
            flag: tour.flag, since: at(tour.since), until: at(tour.until), took: tour.took, byContinent: tour.byContinent,
            tookFrom: tour.tookFrom, nextTook: tour.nextTook, solidity: rounded(tour.solidity, 2), bombs: tour.bombs,
            flippedByContinent: tour.flippedByContinent, fighters: tour.fighters,
        })),
        places: found.places.map((place) => new PlaceWritten({
            kind: place.kind, code: place.code ?? "", continent: place.continent ?? "", tiles: place.tiles, holder: place.holder,
            held: place.held, share: rounded(place.share, 3), rivals: place.rivals, changed: place.changed,
            flippedByContinent: place.flippedByContinent, fighters: place.fighters,
        })),
        sagas: found.sagas.map((saga) => new Saga({
            kind: saga.kind, code: saga.code ?? "", continent: saga.continent ?? "", tiles: saga.tiles,
            reigns: saga.reigns.map((reign) => new ReignWritten({
                flag: reign.flag, pushedAt: atMaybe(reign.pushedAt), since: at(reign.since), until: atMaybe(reign.until),
                peak: rounded(reign.peak, 3), share: rounded(reign.share, 3), tookFrom: reign.tookFrom ?? "",
                tookFromShare: rounded(reign.tookFromShare, 3), lostTo: reign.lostTo ?? "",
            })),
            holder: saga.holder ?? "", share: rounded(saga.share, 3), since: at(saga.since), until: at(saga.until),
            contenders: saga.contenders.map((contender) => new ContenderWritten({
                flag: contender.flag, since: at(contender.since), until: atMaybe(contender.until), peak: rounded(contender.peak, 3),
                share: rounded(contender.share, 3),
            })),
        })),
        paths: found.paths.map((path) => new PathWritten({
            flag: path.flag, marks: path.marks.map((mark) => ({at: at(mark.at), tilesByContinent: mark.tilesByContinent})),
        })),
        seasonal: found.seasonal.map(({story, measured}) => {
            const words = wordsOf(story)
            return new Seasonal({
                title: words.headline, key: storyKeyOf(story, regionOf), countries: storyCountriesOf(story),
                facts: storyFactsOf(story, words.line, measured), teasers: words.teasers,
            })
        }),
        names: Object.fromEntries([...codes].sort().map((code) => [code, said(nameOf(code))])),
        continents: continents().map((name) => ({name, flag: flagOfContinent(name) ?? "", tiles: found.continentTiles[name] ?? 0})),
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

// What --story names: one of the findings, for a front what the editor made of it, and for a front or a place the look
// the editor chose.
export type Told =
    | {front: number, story: Story, look: Look}
    | {tour: number}
    | {place: number, look: Look}
    | {saga: number, since: number, attacker: string, rival: string | undefined}
    | {seasonal: number}

// The story the editor wrote, as a clips.v1.Pick says it, with its words; a seasonal story has words of its own. Nothing
// is judged here: a pick that names no front, no kind, no flag, no words or no look is refused as it is read, and the
// job checked it against the findings before the render. fighters: the players of the cast it names, by number; a
// seasonal story names nobody.
export function toldOf(json: string): {told: Told, written: Written | undefined, fighters: number[]} {
    const pick = Pick.fromJson(JSON.parse(json), {ignoreUnknownFields: true})
    const fighters = pick.fighters
    const named = [pick.front, pick.tour, pick.place, pick.saga, pick.seasonal].filter((number) => number > 0)
    if (named.length !== 1) throw new Error("a story names one front, one tour, one place, one saga or one seasonal story")
    if (pick.seasonal > 0) return {told: {seasonal: pick.seasonal}, written: undefined, fighters: []}
    if (pick.headline === "" || pick.call === "") throw new Error("a story the editor tells has its headline and its call")
    const written: Written = {
        headline: pick.headline, teasers: pick.teasers, line: pick.line === "" ? undefined : pick.line, call: pick.call,
        callFlags: pick.callFlags, anthem: pick.anthem === "" ? undefined : pick.anthem,
    }
    if (pick.tour > 0) return {told: {tour: pick.tour}, written, fighters}
    if (pick.saga > 0) {
        if (pick.since === undefined || pick.attacker === "") throw new Error("a saga is told from a time, about one of its flags")
        return {
            told: {saga: pick.saga, since: pick.since.toDate().getTime(), attacker: pick.attacker, rival: pick.rival === "" ? undefined : pick.rival},
            written, fighters: [],
        }
    }
    const look = LOOKS.find((known) => known === pick.look)
    if (look === undefined) throw new Error(`a front or a place is told with the look "flags" or "dive", not "${pick.look}"`)
    if (pick.place > 0) return {told: {place: pick.place, look}, written, fighters}

    const kind = KINDS.find((known) => known === pick.kind)
    if (kind === undefined || kind === "holding" || kind === "saga" || kind === "round" || kind === "callup") {
        throw new Error(`a front's story is an attack, an invasion, a comeback, a kickout, a rout or a battle, not "${pick.kind}"`)
    }
    if (pick.attacker === "") throw new Error("a front's story has an attacker")
    const place = placeOfPick(pick.countries, pick.continent)
    const story: Story = {
        kind, attacker: pick.attacker, rival: pick.rival === "" ? undefined : pick.rival, victims: pick.takenFrom, place,
        ...pick.team === "" ? {} : {team: pick.team},
    }
    return {told: {front: pick.front, story, look}, written, fighters}
}

function placeOfPick(countries: string[], continent: string): Place {
    if (countries.length === 1) return {country: countries[0]}
    if (countries.length > 1) return {countries}
    if (continent !== "") return {region: continent}
    throw new Error("a story is told in countries, or on a continent")
}
