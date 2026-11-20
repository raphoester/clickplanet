import {Timestamp} from "@bufbuild/protobuf"
import {ClipRecord} from "../gen/grpc/clips/v1/clips_pb.ts"
import {Story, THE_WORLD} from "../domain/clip/story.ts"
import {Look} from "../domain/clip/look.ts"
import {ListName} from "../domain/clip/lists.ts"
import {voiceOf} from "./callUp.ts"

// What a clip was, as the generator chose it and the replay played it: numbers it already worked out, never new ones.
export type Shown = {
    story: Story
    // The commit that made it.
    generator: string
    seconds: number
    music: {code: string, from: number} | undefined
    // A front's camera follows its fighting, a tour's flies over the world, a saga's stays out over its place.
    camera: "front" | "tour" | "saga"
    // undefined on a tour and a saga.
    look: Look | undefined
    since: number
    until: number
    tilesChanged: number
    storyTilesChanged: number
    placeTiles: number
    flagTiles: {start: number, end: number}
    landmassesFlipped: number
    bombs: number
    spreads: number
    enclosures: number
    screens: number
    tourStops: number
    solidity: number
    // The players its end card names.
    fighters: number
    // The small line under the headline.
    line: string | undefined
    // The language the editor wrote a lab story in; none is the story's own (languageOf).
    language?: string
}

const NONE = "none"

const rounded = (value: number, places: number) => Math.round(value * 10 ** places) / 10 ** places

// Of the stories the generator words itself, only a call-up speaks another language than English.
export function languageOf(story: Story): string {
    return story.kind === "callup" && story.finaleAt !== undefined ? voiceOf(story.attacker, story.finaleAt).language : "en"
}

// Photos on TikTok and their slideshow on YouTube: codes, the ranks from the 1st.
export function carouselRecordOf(list: ListName, codes: readonly string[], photos: number, secondsPerPhoto: number, generator: string): ClipRecord {
    return new ClipRecord({
        format: "carousel",
        generator: generator || "unknown",
        common: {durationSeconds: photos * secondsPerPhoto, music: NONE, musicStartSeconds: 0, language: "en", fighters: 0, line: ""},
        carousel: {list, codes: [...codes], photos, secondsPerPhoto},
    })
}

export function clipRecordOf(shown: Shown): ClipRecord {
    const tour = shown.camera === "tour"
    const place = tour ? {region: THE_WORLD} : shown.story.place
    return new ClipRecord({
        format: "globe-replay",
        generator: shown.generator || "unknown",
        common: {
            durationSeconds: rounded(shown.seconds, 2),
            music: shown.music?.code ?? NONE,
            musicStartSeconds: rounded(shown.music?.from ?? 0, 2),
            language: shown.language ?? languageOf(shown.story),
            fighters: shown.fighters,
            line: shown.line ?? "",
        },
        globe: {
            camera: shown.camera,
            look: shown.look ?? NONE,
            storyKind: shown.story.kind,
            spanMinutes: rounded((shown.until - shown.since) / 60_000, 1),
            tilesChanged: shown.tilesChanged,
            storyTilesChanged: shown.storyTilesChanged,
            placeTiles: shown.placeTiles,
            flagTilesStart: shown.flagTiles.start,
            flagTilesEnd: shown.flagTiles.end,
            landmassesFlipped: shown.landmassesFlipped,
            bombs: shown.bombs,
            spreads: shown.spreads,
            enclosures: shown.enclosures,
            screensCrossed: shown.camera === "front" ? rounded(shown.screens, 2) : 0,
            tourStops: tour ? shown.tourStops : 0,
            solidity: rounded(shown.solidity, 2),
            storyEndedAt: Timestamp.fromDate(new Date(shown.until)),
            attacker: shown.story.attacker,
            rival: shown.story.rival ?? "",
            placeCountries: "country" in place ? [place.country] : "countries" in place ? place.countries : [],
            placeRegion: "region" in place ? place.region : "",
        },
    })
}
