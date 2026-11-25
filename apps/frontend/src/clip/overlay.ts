import {NamedSide} from "../domain/clip/cast.ts"
import {Place, sidesOf, Story, storyCountriesOf, THE_WORLD} from "../domain/clip/story.ts"
import {Subtitle, subtitleAt} from "../domain/clip/voice.ts"
import {ordinalOf} from "./ordinal.ts"
import {regionOf} from "../domain/regions.ts"
import {callUpWordsOf} from "./callUp.ts"
import {DISPLAY_FONT_DRAWS, nameOf, said} from "./names.ts"
import {flagPictureUrl} from "../app/flags/flagPicture.ts"

export type Words = {
    headline: string
    // Big words over the opening, before the headline: a lab story's alone (docs/video-lab.md).
    hook?: string
    // Only where it says something the map does not.
    line: string | undefined
    call: string
    // The flags of what the call asks the viewer to fight for.
    callFlags: string[]
    // The headline said in other words, for the caption: the posting job picks one.
    teasers: string[]
    // The name of each teaser's wording, in the same order, so the posts can be compared by it: "editor" for the
    // editor's, the same for every flag and every language for the generator's own ("takes-the-day").
    teaserNames: string[]
    table?: Standing[]
    when?: string
    speech?: Speech
    // A war report's voice, said: drawn where the headline was, which then stays the post's title alone.
    subtitles?: Subtitle[]
}

export type Standing = {rank: number, country: string, name: string, points: string}

export type Speech = {
    tiles(count: number): string
    number(count: number): string
    clock(at: number): string
}

// The words of a hashtag, with no #.
export type Hashtags = {countries: {code: string, tag: string}[], continent: string | undefined}

export type Moment = {
    // Seconds into the clip.
    seconds: number
    replayAt: number
    played: number
    held: ReadonlyMap<string, number>
    // Seconds into the call to act, below 0 before it.
    ending: number
}

export type Overlay = {
    show(moment: Moment): void
    // Once the fonts and the pictures are in: sets a name too long for its row smaller, never cut nor wrapped, and the
    // call's flags lower together where their row is wider than the card.
    fit(): void
}

const NUMBER = new Intl.NumberFormat("en-US")

const TIME = new Intl.DateTimeFormat("en-GB", {
    weekday: "short", day: "numeric", month: "short", hour: "2-digit", minute: "2-digit", timeZone: "UTC",
})

const DAY = new Intl.DateTimeFormat("en-GB", {weekday: "short", day: "numeric", month: "short", timeZone: "UTC"})

const LIST = new Intl.ListFormat("en-GB", {type: "conjunction"})

// The countries a place is made of; none for a continent.
function countriesOf(place: Place): string[] {
    return "country" in place ? [place.country] : "countries" in place ? place.countries : []
}

export function placeName(place: Place): string {
    return "region" in place ? place.region : LIST.format(countriesOf(place).map(nameOf))
}

export function tagOf(name: string): string {
    return name.toLowerCase().replace(/[^\p{L}\p{N}]/gu, "")
}

// Every flag of the story that is a country of the game, and the countries of its place; a continent's own flag is not one.
export function hashtagsOf(story: Story): Hashtags {
    const place = story.place
    const country = "country" in place ? place.country : "countries" in place ? place.countries[0] : undefined
    const continent = "region" in place ? place.region : country === undefined ? undefined : regionOf(country)
    return {
        countries: storyCountriesOf(story).map((code) => ({code, tag: tagOf(nameOf(code))})),
        continent: continent === undefined || continent === THE_WORLD ? undefined : tagOf(continent),
    }
}

// Every word the editor wrote of a story, for its clip and its caption, and the code whose anthem plays under it.
export type Written = {
    headline: string
    teasers: string[]
    line: string | undefined
    call: string
    callFlags: string[]
    anthem: string | undefined
    // A lab story's: its words over the opening, and the language they are all in; none is English.
    hook?: string
    language?: string
    // A war report's: the post's title, its writer's, which the caption opens with in place of the teasers.
    title?: string
}

// The name of a war report's title among the teasers: its writer's.
const WRITER = "writer"

const named = (teasers: [name: string, line: string][]) => ({
    teasers: teasers.map(([, line]) => line.toUpperCase()),
    teaserNames: teasers.map(([name]) => name),
})

// A story the editor tells is told in its words alone; a day won and a call-up, which no editor writes, in their own.
// headline: one given by hand, over either. The caption never repeats the clip: a teaser that says the headline is
// left out, but a war report's title, as its voice is drawn where the headline was.
export function wordsOf(story: Story, written?: Written, headline?: string): Words {
    const words = story.kind === "callup" ? callUpWordsOf(story, {nameOf, said}, headline)
        : story.kind === "round" ? roundWordsOf(story, headline)
            : editedOf(story, written, headline)
    const kept = words.teasers.map((_, i) => i).filter((i) => words.teaserNames[i] === WRITER || words.teasers[i] !== words.headline)
    return {...words, teasers: kept.map((i) => words.teasers[i]), teaserNames: kept.map((i) => words.teaserNames[i])}
}

function editedOf(story: Story, written: Written | undefined, headline: string | undefined): Words {
    if (written === undefined) throw new Error(`a ${story.kind} is told in the editor's words, and none were given`)
    return {
        headline: headline ?? written.headline,
        ...written.hook === undefined ? {} : {hook: written.hook},
        line: written.line,
        call: written.call,
        callFlags: written.callFlags,
        ...named(written.title !== undefined ? [[WRITER, written.title]]
            : written.teasers.map((teaser): [string, string] => ["editor", teaser])),
    }
}

function roundWordsOf(story: Story, headline: string | undefined): Words {
    const day = story.day
    if (day === undefined) throw new Error("a day won needs its day")
    const attacker = said(nameOf(story.attacker))
    const won = day.finale ? "the final battle" : day.own ? "the battle" : `day ${day.number}`
    const after = day.table.find(({country}) => country === story.attacker)
    const points = after === undefined ? "" : `${NUMBER.format(after.points)} ${after.points === 1 ? "point" : "points"}`
    return {
        headline: headline ?? `${attacker} WINS ${won}`.toUpperCase(),
        line: after === undefined ? undefined
            : after.rank > 1 ? `${ordinalOf(after.rank)} in the season with ${points}`
                : day.finale ? `Wins the season with ${points}` : `Leads the season with ${points}`,
        call: day.finale || day.own ? "PLAY FOR YOUR COUNTRY" : `WHO WINS DAY ${day.number + 1}?`,
        callFlags: [],
        ...named([["takes-the-day", `${attacker} takes ${won}`], ["day-goes-to", `${won} goes to ${attacker}`]]),
    }
}

function element<K extends keyof HTMLElementTagNameMap>(tag: K, className: string, parent: HTMLElement) {
    const created = document.createElement(tag)
    created.className = className
    parent.append(created)
    return created
}

function flag(code: string, className: string, parent: HTMLElement) {
    const image = element("img", className, parent)
    image.src = flagPictureUrl(code)
    image.alt = ""
    return image
}


// A share of a place, to a tenth of a point, never more than it is: 79.96% is 79.9%.
function shareOf(count: number, ground: number): string {
    return `${(Math.floor(count / Math.max(1, ground) * 1000) / 10).toFixed(1)}%`
}

const TOP_FIGHTERS = "TOP FIGHTERS"

// The height the call's flags share, each as wide as its own shape: the one they are drawn at, or lower where their row
// is wider than the room. drawn: the flags' own widths at that height, which scale with it; the rest of the row, their
// frames and the gaps between them, does not.
export function flagHeightOf(height: number, row: number, drawn: number, room: number): number {
    if (row <= room || drawn <= 0) return height
    return Math.floor(height * (room - (row - drawn)) / drawn)
}

function fitFlags(row: HTMLElement, flags: HTMLImageElement[]) {
    const card = row.parentElement
    if (flags.length === 0 || card === null) return
    const style = getComputedStyle(card)
    const room = card.clientWidth - parseFloat(style.paddingLeft || "0") - parseFloat(style.paddingRight || "0")
    const drawn = flags.reduce((sum, image) => sum + image.clientWidth, 0)
    const height = flags[0].clientHeight
    const fitted = flagHeightOf(height, row.scrollWidth, drawn, room)
    if (fitted < height) row.style.setProperty("--clip-flag-height", `${fitted}px`)
}

// The call fades in as the counter fades out, and the end card only once the counter is gone: it covers where the
// counter was.
const FADE_SECONDS = 0.5

const faded = (seconds: number) => Math.min(1, Math.max(0, seconds / FADE_SECONDS))

// The hook is read alone first, then fades as the headline comes in.
export const HOOK_SECONDS = 1.5

function tilesSaid(count: number): string {
    return `${NUMBER.format(count)} ${count === 1 ? "tile" : "tiles"}`
}

const MOST_SUBTITLE_LINES = 2

const subtitleText = ({text}: Subtitle) => text.toUpperCase()

function linesOf(box: HTMLElement): number {
    const style = getComputedStyle(box)
    const height = box.clientHeight - parseFloat(style.paddingTop || "0") - parseFloat(style.paddingBottom || "0")
    return Math.round(height / (parseFloat(style.lineHeight) || 1))
}

// The size each piece too long for two lines is set at, a pixel smaller at a time: a long word can make a third.
function subtitleSizesOf(box: HTMLElement, pieces: readonly Subtitle[]): Map<Subtitle, string> {
    const base = parseFloat(getComputedStyle(box).fontSize)
    const sizes = new Map<Subtitle, string>()
    for (const piece of pieces) {
        box.textContent = subtitleText(piece)
        let size = base
        while (linesOf(box) > MOST_SUBTITLE_LINES && size > base / 2) {
            size -= 1
            box.style.fontSize = `${size}px`
        }
        if (size < base) sizes.set(piece, `${size}px`)
        box.style.fontSize = ""
    }
    box.textContent = ""
    return sizes
}

// named: the players the end card names, by side. ground: the tiles of the place, when the counter says what share of
// it a side holds rather than how many tiles.
export function createOverlay(
    root: HTMLElement,
    story: Story,
    words: Words,
    opening: ReadonlyMap<string, number>,
    named: readonly NamedSide[],
    ground?: number,
): Overlay {
    const sides = sidesOf(story)
    const speech = words.speech
    const subtitles = words.subtitles ?? []
    const voiced = subtitles.length > 0
    const said = [
        ...voiced ? subtitles.map(subtitleText) : [words.headline, words.hook ?? ""],
        words.call, words.when ?? "", ...(words.table ?? []).map(({name, points}) => `${name} ${points}`),
        ...speech === undefined ? [] : [0, 1, 2, 5, 22, 1234].map(speech.tiles),
        ...named.length === 0 ? [] : [TOP_FIGHTERS],
        ...named.flatMap(({fighters}) => fighters.map(({name, took}) => `${name} ${tilesSaid(took)}`)),
    ]
    if (!said.every((text) => DISPLAY_FONT_DRAWS.test(text))) root.classList.add("clip--fallback-font")

    const top = element("header", "clip-top", root)
    // A voice's words are read where the headline was, and nothing else is: no headline, no line, no hook.
    const subtitle = voiced ? element("p", "clip-subtitle", element("div", "clip-subtitles", top)) : undefined
    if (!voiced) {
        const headline = element("h1", "clip-headline", top)
        for (const side of sides) flag(side, "clip-headline-flag", headline)
        element("span", "", headline).textContent = words.headline
        if (words.line !== undefined) element("p", "clip-line", top).textContent = words.line
    }
    let subtitleSizes = new Map<Subtitle, string>()
    let saying: Subtitle | undefined

    const bottom = element("footer", "clip-bottom", root)
    // A saga is days long: the day it is at reads as big as its counters.
    const day = story.kind === "saga" ? element("p", "clip-day", bottom) : undefined
    const counters = element("div", "clip-counters", bottom)
    // A call-up is about the season's points, which its end shows: the tiles of a few hours say nothing of them.
    const shown = (story.kind === "callup" ? [] : sides).map((side) => {
        const counter = element("div", "clip-counter", counters)
        flag(side, "clip-counter-flag", counter)
        return {side, tiles: element("span", "clip-counter-tiles", counter), delta: element("span", "clip-counter-delta", counter)}
    })
    const mark = element("p", "clip-mark", bottom)
    const logo = element("img", "clip-logo", mark)
    logo.src = "/static/logo.svg"
    logo.alt = ""
    element("span", "clip-site", mark).textContent = "clickplanet.lol"
    const time = element("span", "clip-time", mark)
    const track = element("div", "clip-track", bottom)
    const progress = element("div", "clip-progress", track)

    const hook = words.hook === undefined || voiced ? undefined : element("p", "clip-hook", root)
    if (hook !== undefined) hook.textContent = words.hook ?? ""

    const ending = element("section", "clip-ending", root)
    const endingFlags = element("div", "clip-ending-flags", ending)
    const callFlags = words.callFlags.map((called) => flag(called, "clip-ending-flag", endingFlags))
    if (words.table !== undefined) {
        const table = element("ol", "clip-podium", ending)
        for (const {rank, country, name, points} of words.table) {
            const place = element("li", `clip-podium-place clip-podium-place--${rank === 1 ? 1 : "next"}`, table)
            element("span", "clip-podium-rank", place).textContent = `#${rank}`
            flag(country, "clip-podium-flag", place)
            element("span", "clip-podium-name", place).textContent = name
            element("span", "clip-podium-points", place).textContent = points
        }
    } else if (story.day !== undefined) {
        const podium = element("ol", "clip-podium", ending)
        for (const {country, rank, points} of story.day.podium) {
            const place = element("li", `clip-podium-place clip-podium-place--${rank}`, podium)
            element("span", "clip-podium-rank", place).textContent = ordinalOf(rank).toUpperCase()
            flag(country, "clip-podium-flag", place)
            element("span", "clip-podium-name", place).textContent = nameOf(country).toUpperCase()
            element("span", "clip-podium-points", place).textContent = `+${NUMBER.format(points)}`
        }
    }
    element("p", "clip-call", ending).textContent = words.call
    if (words.when !== undefined) element("p", "clip-when", ending).textContent = words.when
    element("p", "clip-url", ending).textContent = "clickplanet.lol"
    const card = named.length === 0 ? undefined : element("section", "clip-fighters", ending)
    const names: HTMLElement[] = []
    if (card !== undefined) {
        element("h2", "clip-fighters-title", card).textContent = TOP_FIGHTERS
        for (const side of named) {
            const rows = element("ol", "clip-fighters-side", card)
            for (const {name, took} of side.fighters) {
                const row = element("li", "clip-fighter", rows)
                flag(side.flag, "clip-fighter-flag", row)
                const shown = element("span", "clip-fighter-name", row)
                shown.textContent = name
                names.push(shown)
                element("span", "clip-fighter-took", row).textContent = tilesSaid(took)
            }
        }
    }

    return {
        show({seconds, replayAt, played, held, ending: into}) {
            const over = faded(into)
            const opened = hook === undefined ? 1 : faded(seconds - HOOK_SECONDS)
            if (hook !== undefined) hook.style.opacity = String(1 - opened)
            const now = subtitleAt(subtitles, seconds)
            if (subtitle !== undefined && now !== saying) {
                subtitle.textContent = now === undefined ? "" : subtitleText(now)
                subtitle.style.fontSize = now === undefined ? "" : subtitleSizes.get(now) ?? ""
                saying = now
            }
            for (const {side, tiles, delta} of shown) {
                const count = held.get(side) ?? 0
                const moved = count - (opening.get(side) ?? 0)
                tiles.textContent = ground !== undefined ? shareOf(count, ground) : speech?.tiles(count) ?? tilesSaid(count)
                const by = ground !== undefined ? shareOf(Math.abs(moved), ground)
                    : speech?.number(Math.abs(moved)) ?? NUMBER.format(Math.abs(moved))
                delta.textContent = moved === 0 || by === "0.0%" ? "" : `${moved > 0 ? "+" : "−"}${by}`
                delta.classList.toggle("clip-counter-delta--gain", moved > 0)
                delta.classList.toggle("clip-counter-delta--loss", moved < 0)
            }
            time.textContent = speech?.clock(replayAt) ?? `${TIME.format(new Date(replayAt))} UTC`
            if (day !== undefined) day.textContent = DAY.format(new Date(replayAt)).toUpperCase()
            progress.style.transform = `scaleX(${played})`
            ending.style.opacity = String(over)
            top.style.opacity = String(Math.min(opened, 1 - over))
            bottom.style.opacity = String(1 - over)
            if (card !== undefined) card.style.opacity = String(faded(into - FADE_SECONDS))
        },
        fit() {
            fitFlags(endingFlags, callFlags)
            if (subtitle !== undefined) {
                subtitleSizes = subtitleSizesOf(subtitle, subtitles)
                saying = undefined
            }
            for (const name of names) {
                if (name.scrollWidth > name.clientWidth) name.style.fontSize = `${Math.floor(name.clientWidth / name.scrollWidth * 100) / 100}em`
            }
        },
    }
}
