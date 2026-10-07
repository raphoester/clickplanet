import {Countries} from "../domain/countries.ts"
import {Place, Story, THE_WORLD} from "../domain/clip/story.ts"

export type Words = {
    headline: string
    line: string
    call: string
    // The flags of what the call asks the viewer to fight for.
    callFlags: string[]
    link: string
    tags: string[]
}

export type Moment = {
    replayAt: number
    played: number
    held: ReadonlyMap<string, number>
    ending: number
}

export type Overlay = {
    show(moment: Moment): void
}

const NUMBER = new Intl.NumberFormat("en-US")

const LIST = new Intl.ListFormat("en", {style: "long", type: "conjunction"})

const TIME = new Intl.DateTimeFormat("en-GB", {
    weekday: "short", day: "numeric", month: "short", hour: "2-digit", minute: "2-digit", timeZone: "UTC",
})

const SITE = "https://clickplanet.lol"

// The only continent with a flag of its own in static/countries/svg.
const CONTINENT_FLAGS: ReadonlyMap<string, string[]> = new Map([["Europe", ["eu"]]])

export function nameOf(code: string): string {
    return Countries.get(code)?.name ?? code.toUpperCase()
}

export function placeName(place: Place): string {
    return "country" in place ? nameOf(place.country) : place.region
}

function spanOf(milliseconds: number): string {
    const hours = Math.round(milliseconds / 3_600_000)
    if (hours >= 2) return `${hours} hours`
    if (hours === 1) return "1 hour"
    return `${Math.max(1, Math.round(milliseconds / 60_000))} minutes`
}

function tagOf(name: string): string {
    return `#${name.replace(/[^\p{L}\p{N}]/gu, "")}`
}

export function wordsOf(story: Story, span: number, headline?: string): Words {
    const attacker = nameOf(story.attacker)
    const place = placeName(story.place)
    const during = spanOf(span)
    const taken = NUMBER.format(story.taken)
    const tags = [...new Set([tagOf("clickplanet"), tagOf(attacker), ...place === THE_WORLD ? [] : [tagOf(place)], "#geography", "#map"])]

    if (story.kind === "battle" && story.rival !== undefined) {
        return {
            headline: headline ?? `${attacker} VS ${nameOf(story.rival)}`.toUpperCase(),
            line: `The battle for ${place}, ${during} of it.`,
            call: "PICK A SIDE",
            callFlags: [story.attacker, story.rival],
            link: SITE,
            tags,
        }
    }

    const victims = story.victims.map(nameOf)
    if (story.kind === "comeback") {
        return {
            headline: headline ?? `${attacker} STRIKES BACK`.toUpperCase(),
            line: victims.length > 0
                ? `${taken} tiles taken back in ${during}, from ${LIST.format(victims)}.`
                : `${taken} tiles taken back in ${during}.`,
            call: `FIGHT FOR ${attacker}`.toUpperCase(),
            callFlags: [story.attacker],
            link: `${SITE}/?f=${story.attacker}`,
            tags,
        }
    }

    const defended = "country" in story.place ? story.place.country : story.victims[0]
    return {
        headline: headline ?? (story.kind === "invasion"
            ? `${attacker} IS INVADING ${place}`
            : `${attacker} IS ATTACKING ${place}`).toUpperCase(),
        line: victims.length > 0 && story.kind === "attack"
            ? `${taken} tiles in ${during}, from ${LIST.format(victims)}.`
            : `${taken} tiles in ${during}.`,
        call: `DEFEND ${place}`.toUpperCase(),
        callFlags: "country" in story.place ? [story.place.country] : CONTINENT_FLAGS.get(story.place.region) ?? [],
        link: defended === undefined ? SITE : `${SITE}/?f=${defended}`,
        tags,
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
    image.src = `/static/countries/svg/${code}.svg`
    image.alt = ""
    return image
}

export function createOverlay(root: HTMLElement, story: Story, words: Words, opening: ReadonlyMap<string, number>): Overlay {
    const sides = story.rival === undefined ? [story.attacker] : [story.attacker, story.rival]

    const top = element("header", "clip-top", root)
    const headline = element("h1", "clip-headline", top)
    for (const side of sides) flag(side, "clip-headline-flag", headline)
    element("span", "", headline).textContent = words.headline
    element("p", "clip-line", top).textContent = words.line

    const bottom = element("footer", "clip-bottom", root)
    const counters = element("div", "clip-counters", bottom)
    const shown = sides.map((side) => {
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

    const ending = element("section", "clip-ending", root)
    const endingFlags = element("div", "clip-ending-flags", ending)
    for (const called of words.callFlags) flag(called, "clip-ending-flag", endingFlags)
    element("p", "clip-call", ending).textContent = words.call
    element("p", "clip-url", ending).textContent = "clickplanet.lol"

    return {
        show({replayAt, played, held, ending: over}) {
            for (const {side, tiles, delta} of shown) {
                const count = held.get(side) ?? 0
                const moved = count - (opening.get(side) ?? 0)
                tiles.textContent = `${NUMBER.format(count)} ${count === 1 ? "tile" : "tiles"}`
                delta.textContent = moved === 0 ? "" : `${moved > 0 ? "+" : "−"}${NUMBER.format(Math.abs(moved))}`
                delta.classList.toggle("clip-counter-delta--gain", moved > 0)
                delta.classList.toggle("clip-counter-delta--loss", moved < 0)
            }
            time.textContent = `${TIME.format(new Date(replayAt))} UTC`
            progress.style.transform = `scaleX(${played})`
            ending.style.opacity = String(over)
            top.style.opacity = String(1 - over)
            bottom.style.opacity = String(1 - over)
        },
    }
}
