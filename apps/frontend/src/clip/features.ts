import {ClipFeatures} from "../gen/grpc/clips/v1/clips_pb.ts"
import {Story} from "../domain/clip/story.ts"
import {Look} from "../domain/clip/look.ts"
import {voiceOf} from "./callUp.ts"

// What a clip was, as the generator chose it and the replay played it: numbers it already worked out, never new ones.
export type Shown = {
    story: Story
    seconds: number
    music: {code: string, from: number} | undefined
    // undefined on a tour.
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
}

const rounded = (value: number, places: number) => Math.round(value * 10 ** places) / 10 ** places

// Only a call-up speaks another language than English.
export function languageOf(story: Story): string {
    return story.kind === "callup" && story.finaleAt !== undefined ? voiceOf(story.attacker, story.finaleAt).language : "en"
}

export function featuresOf(shown: Shown): ClipFeatures {
    const tour = shown.look === undefined
    return new ClipFeatures({
        durationSeconds: rounded(shown.seconds, 2),
        music: shown.music?.code ?? "",
        musicStartSeconds: rounded(shown.music?.from ?? 0, 2),
        camera: tour ? "tour" : "front",
        look: shown.look ?? "",
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
        screensCrossed: tour ? 0 : rounded(shown.screens, 2),
        tourStops: tour ? shown.tourStops : 0,
        language: languageOf(shown.story),
        solidity: rounded(shown.solidity, 2),
    })
}
