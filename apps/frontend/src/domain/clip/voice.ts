import {ENDING_SECONDS, Timeline} from "./pace.ts"

// A line of the voice as voice.mjs writes it: when it is said, in seconds into the voice.
export type Said = {text: string, from: number, to: number}

// A piece of the subtitles: seconds into the clip.
export type Subtitle = {text: string, from: number, until: number}

// The voice starts a little into the clip, over the anthem, so its first word is not on the first frame.
export const VOICE_AT_SECONDS = 0.4

const MOST_WORDS = 6

const LAST_HELD_SECONDS = 0.4

// The design's ceiling (docs/video-lab.md): 900 frames, 28 to 47 minutes on a runner, inside the render job's 60.
const LONGEST_VOICED_SECONDS = 30

function phrasesOf(text: string): string[][] {
    const phrases: string[][] = [[]]
    for (const word of text.split(/\s+/).filter(Boolean)) {
        phrases[phrases.length - 1].push(word)
        if (/[,;:]$/.test(word)) phrases.push([])
    }
    return phrases.filter((phrase) => phrase.length > 0)
}

// As even as they go, the longer first: 7 words are 4 and 3, 13 are 5, 4 and 4.
function piecesOf(words: readonly string[]): string[] {
    const count = Math.ceil(words.length / MOST_WORDS)
    const size = Math.floor(words.length / count)
    const longer = words.length % count
    const pieces: string[] = []
    let at = 0
    for (let i = 0; i < count; i++) {
        const next = at + size + (i < longer ? 1 : 0)
        pieces.push(words.slice(at, next).join(" "))
        at = next
    }
    return pieces
}

// Each line cut at its commas, semicolons and colons, then evenly into pieces of 6 words at most, each timed by its
// share of the line's characters. A piece shows until the next one starts, the last until its line ends and a little
// after.
export function subtitlesOf(lines: readonly Said[]): Subtitle[] {
    const timed = lines.flatMap(({text, from, to}) => {
        const pieces = phrasesOf(text).flatMap(piecesOf)
        const characters = pieces.reduce((sum, piece) => sum + piece.length, 0)
        let before = 0
        return pieces.map((piece) => {
            const start = from + (to - from) * before / characters
            before += piece.length
            return {text: piece, start, end: to}
        })
    })
    return timed.map(({text, start, end}, i) => ({
        text,
        from: VOICE_AT_SECONDS + start,
        until: VOICE_AT_SECONDS + (i + 1 < timed.length ? timed[i + 1].start : end + LAST_HELD_SECONDS),
    }))
}

export function subtitleAt(subtitles: readonly Subtitle[], seconds: number): Subtitle | undefined {
    return subtitles.find(({from, until}) => seconds >= from && seconds < until)
}

// A clip with a voice lasts as long as the voice, from when it starts to its last words read, then its call: the call
// never crosses the last words, and the voice is never cut, past the 22 s of a clip with none when it must.
export function voicedTimelineOf(lines: readonly Said[], ending = ENDING_SECONDS): Timeline {
    const said = lines.at(-1)?.to
    if (said === undefined) throw new Error("a voice says something: it has no line")
    const seconds = Math.round((VOICE_AT_SECONDS + said + LAST_HELD_SECONDS + ending) * 100) / 100
    if (seconds > LONGEST_VOICED_SECONDS) {
        throw new Error(`a voice of ${said.toFixed(1)} s makes a clip of ${seconds.toFixed(1)} s, past the ${LONGEST_VOICED_SECONDS} s `
            + "the render job has time for")
    }
    return {seconds, ending}
}
