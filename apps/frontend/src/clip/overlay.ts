import {Countries} from "../domain/countries.ts"
import {inPlace, Place, Story, THE_WORLD} from "../domain/clip/story.ts"
import {flagOfContinent, regionOf} from "../domain/regions.ts"

export type Words = {
    headline: string
    // Only where it says something the map does not.
    line: string | undefined
    call: string
    // The flags of what the call asks the viewer to fight for.
    callFlags: string[]
    link: string
    // What to post with the clip: links in a caption cannot be clicked, so the site is plain text.
    caption: string
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

const TIME = new Intl.DateTimeFormat("en-GB", {
    weekday: "short", day: "numeric", month: "short", hour: "2-digit", minute: "2-digit", timeZone: "UTC",
})

const SITE = "https://clickplanet.lol"

const ENGLISH = new Intl.DisplayNames(["en"], {type: "region", style: "short"})

// The game's names are cut to fit its board ("Christmas", "N.Zealand", "Czech Rep."): a clip says the English name they
// were cut from, from the CLDR data the browser carries. A name the game chose over it stays: "Turkey", "Ivory Coast",
// "UAE", and its own flags'.
export function nameOf(code: string): string {
    const game = Countries.get(code)?.name ?? code.toUpperCase()
    const english = englishNameOf(code)
    return english !== undefined && cutFrom(game, english) ? english : game
}

// ISO's x codes are for private use: CLDR has a test name where the game has Brittany.
function englishNameOf(code: string): string | undefined {
    if (!/^[a-wyz][a-z]$/.test(code)) return undefined
    const name = ENGLISH.of(code.toUpperCase())
    // "Congo - Kinshasa" names two places to tell them apart; the game's "DR Congo" says it better.
    return name === undefined || name.includes(" - ") ? undefined : name.replace(/ \(.*\)$/, "")
}

function cutFrom(game: string, english: string): boolean {
    if (/[.,]/.test(game)) return true
    const wordsOf = (name: string) => name.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase().split(/[^a-z]+/).filter(Boolean)
    const full = wordsOf(english)
    return english.length > game.length && wordsOf(game).every((word) => full.some((whole) => whole.startsWith(word)))
}

// The countries a place is made of; none for a continent.
function countriesOf(place: Place): string[] {
    return "country" in place ? [place.country] : "countries" in place ? place.countries : []
}

// The UK, the Netherlands: a name said with its article.
const SAID_WITH_THE = /^(UK|USA|UAE|Gambia|Middle East)$|(lands|ines|amas|ives|oros|elles|Republic|Territories)$/

function said(name: string): string {
    return SAID_WITH_THE.test(name) ? `the ${name}` : name
}

export function placeName(place: Place): string {
    return "region" in place ? place.region : countriesOf(place).map(nameOf).join(" and ")
}

function placeSaid(place: Place): string {
    return "region" in place ? said(place.region) : countriesOf(place).map((country) => said(nameOf(country))).join(" and ")
}

// The flags of a place: its countries', or its continent's own when it has one.
function placeFlags(place: Place): string[] {
    if (!("region" in place)) return countriesOf(place)
    const flag = flagOfContinent(place.region)
    return flag === undefined ? [] : [flag]
}

// The account's own tags, and the place's. Never the attacker's: a flag's tag can be a political feed.
const TAGS = ["#clickplanet", "#pixelwars", "#rplace", "#wplace", "#map"]

function tagOf(name: string): string {
    return `#${name.toLowerCase().replace(/[^\p{L}\p{N}]/gu, "")}`
}

// The flag a continent striking back together is counted and shown under.
export function teamFlagOf(story: Story): string | undefined {
    return story.team === undefined ? undefined : flagOfContinent(story.team) ?? story.attacker
}

// The flags a story is between: the attacker, and whoever it fights or kicks out. A continent striking back
// together is its own flag, or its leading one's when it has none.
export function sidesOf(story: Story): string[] {
    const team = teamFlagOf(story)
    if (story.kind === "rout") return team === undefined ? [story.victims[0]] : [story.victims[0], team]
    if (team !== undefined) return [team]
    if (story.kind === "battle" && story.rival !== undefined) return [story.attacker, story.rival]
    if (story.kind === "kickout" && story.victims.length > 0) return [story.attacker, story.victims[0]]
    return [story.attacker]
}

export function wordsOf(story: Story, headline?: string): Words {
    const attacker = said(nameOf(story.attacker))
    const place = placeSaid(story.place)
    const placeTags = "region" in story.place
        ? story.place.region === THE_WORLD ? [] : [tagOf(story.place.region)]
        : countriesOf(story.place).map((country) => tagOf(nameOf(country)))
    const tags = [...TAGS, ...placeTags]
    const sides = sidesOf(story)
    // The caption asks what the call at the end asks.
    const finish = (words: Omit<Words, "caption">, question: string): Words => ({
        ...words,
        caption: `${words.headline}. ${question} 👇\nclickplanet.lol\n${tags.join(" ")}`,
    })

    if (story.kind === "rout") {
        const loser = said(nameOf(sides[0]))
        const home = inPlace(story.place, sides[0], regionOf)
        const ownCountry = "country" in story.place && story.place.country === sides[0]
        return finish({
            headline: headline ?? (ownCountry ? `${loser} IS FALLING`
                : home ? `${loser} IS LOSING ${place}` : `${loser} GETS KICKED OUT OF ${place}`).toUpperCase(),
            line: undefined,
            call: sides.length === 2 ? "PICK A SIDE" : `FIGHT FOR ${loser}`.toUpperCase(),
            callFlags: sides,
            link: sides.length === 2 ? SITE : `${SITE}/?f=${sides[0]}`,
        }, sides.length === 2 ? "Pick a side" : "Who saves them?")
    }

    if (story.team !== undefined) {
        return finish({
            headline: headline ?? `${said(story.team)} STRIKES BACK`.toUpperCase(),
            line: undefined,
            call: `FIGHT FOR ${said(story.team)}`.toUpperCase(),
            callFlags: sides,
            link: SITE,
        }, "Who joins them?")
    }

    if (sides.length === 2) {
        return finish({
            headline: headline ?? (story.kind === "kickout"
                ? `${attacker} IS KICKING ${said(nameOf(sides[1]))} OUT OF ${place}`
                : `${attacker} VS ${said(nameOf(sides[1]))}`).toUpperCase(),
            line: story.kind === "battle" ? `The battle for ${place}` : undefined,
            call: "PICK A SIDE",
            callFlags: sides,
            link: SITE,
        }, "Pick a side")
    }

    if (story.kind === "comeback") {
        return finish({
            headline: headline ?? `${attacker} STRIKES BACK`.toUpperCase(),
            line: undefined,
            call: `FIGHT FOR ${attacker}`.toUpperCase(),
            callFlags: [story.attacker],
            link: `${SITE}/?f=${story.attacker}`,
        }, "Who joins them?")
    }

    const world = "region" in story.place && story.place.region === THE_WORLD
    const defended = world ? undefined : countriesOf(story.place)[0] ?? story.victims[0]
    return finish({
        headline: headline ?? (world ? `${attacker} IS TAKING OVER ${place}`
            : story.kind === "invasion" ? `${attacker} IS INVADING ${place}`
                : `${attacker} IS ATTACKING ${place}`).toUpperCase(),
        line: undefined,
        call: world ? "FIGHT BACK" : `DEFEND ${place}`.toUpperCase(),
        callFlags: placeFlags(story.place),
        link: defended === undefined ? SITE : `${SITE}/?f=${defended}`,
    }, "Who stops them?")
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
    const sides = sidesOf(story)

    const top = element("header", "clip-top", root)
    const headline = element("h1", "clip-headline", top)
    for (const side of sides) flag(side, "clip-headline-flag", headline)
    element("span", "", headline).textContent = words.headline
    if (words.line !== undefined) element("p", "clip-line", top).textContent = words.line

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
