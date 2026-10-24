import {Timestamp} from "@bufbuild/protobuf"
import {Shortlist, StoryFacts} from "../gen/grpc/clips/v1/clips_pb.ts"
import {Story, storyCountriesOf, storyKeyOf} from "../domain/clip/story.ts"
import {regionOf} from "../domain/regions.ts"
import {nameOf, placeName, said, wordsOf} from "./overlay.ts"
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

export type Listed = {story: Story, measured?: Measured}

const rounded = (value: number, places: number) => Math.round(value * 10 ** places) / 10 ** places

// Every story worth a clip, best first, with what the job's editor judges it by: the facts and the teasers.
export function shortlistOf(listed: readonly Listed[]): Shortlist {
    return new Shortlist({
        stories: listed.map(({story, measured}) => {
            const words = wordsOf(story)
            return {
                title: words.headline, key: storyKeyOf(story, regionOf), countries: storyCountriesOf(story),
                facts: measured === undefined ? undefined : storyFactsOf(story, words.line, measured), teasers: words.teasers,
            }
        }),
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
        storyEndedAt: Timestamp.fromDate(new Date(measured.until)),
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
