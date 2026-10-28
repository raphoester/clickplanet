import {Countries} from "../domain/countries.ts"
import {NamedSide} from "../domain/clip/cast.ts"
import {inPlace, Place, sidesOf, Story, storyCountriesOf, THE_WORLD} from "../domain/clip/story.ts"
import {flagOfContinent, regionOf} from "../domain/regions.ts"
import {callUpWordsOf} from "./callUp.ts"

export type Words = {
    headline: string
    // Only where it says something the map does not.
    line: string | undefined
    call: string
    // The flags of what the call asks the viewer to fight for.
    callFlags: string[]
    link: string
    // The headline said in other words, for the caption: the posting job picks one.
    teasers: string[]
    // The name of each teaser's wording, in the same order: the same for every flag and every language, so the posts
    // can be compared by it ("invades", "under-attack-from").
    teaserNames: string[]
    table?: Standing[]
    when?: string
    speech?: Speech
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
    replayAt: number
    played: number
    held: ReadonlyMap<string, number>
    // Seconds into the call to act, below 0 before it.
    ending: number
}

export type Overlay = {
    show(moment: Moment): void
    // Once the fonts are in: sets a name too long for its row smaller, never cut nor wrapped.
    fit(): void
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

export function said(name: string): string {
    return SAID_WITH_THE.test(name) ? `the ${name}` : name
}

// 1st, 2nd, 3rd, 4th, 11th, 21st.
function ordinalOf(rank: number): string {
    const teen = rank % 100 >= 11 && rank % 100 <= 13
    const suffix = teen ? "th" : ["th", "st", "nd", "rd"][rank % 10] ?? "th"
    return `${rank}${suffix}`
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

function tagOf(name: string): string {
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

const named = (teasers: [name: string, line: string][]) => ({
    teasers: teasers.map(([, line]) => line.toUpperCase()),
    teaserNames: teasers.map(([name]) => name),
})

// headline: the editor's, in place of the generator's own. The caption never repeats the clip: a teaser that says the
// headline is left out.
export function wordsOf(story: Story, headline?: string): Words {
    const words = wordedOf(story, headline)
    const kept = words.teasers.map((_, i) => i).filter((i) => words.teasers[i] !== words.headline)
    return {...words, teasers: kept.map((i) => words.teasers[i]), teaserNames: kept.map((i) => words.teaserNames[i])}
}

function wordedOf(story: Story, headline?: string): Words {
    if (story.kind === "callup") return callUpWordsOf(story, {nameOf, said}, headline)

    const attacker = said(nameOf(story.attacker))
    const place = placeSaid(story.place)
    const sides = sidesOf(story)
    const victim = story.victims[0] === undefined ? undefined : said(nameOf(story.victims[0]))

    if (story.kind === "holding") {
        const world = "region" in story.place && story.place.region === THE_WORLD
        const share = story.share ?? 0
        // Never more than it holds: 79.6% is 79%.
        const percent = `${Math.floor(share * 100)}%`
        const country = countriesOf(story.place)[0]
        return {
            headline: headline ?? `${attacker} HOLDS ${percent} OF ${place}`.toUpperCase(),
            line: undefined,
            call: world ? "FIGHT BACK" : `TAKE ${place} BACK`.toUpperCase(),
            callFlags: world ? [] : placeFlags(story.place),
            link: country === undefined ? SITE : `${SITE}/?f=${country}`,
            ...named([
                ["flies-the-flag-of", `${percent} of ${place} flies the flag of ${attacker}`],
                world ? ["biggest-empire-on-earth", `the biggest empire on earth: ${attacker}`]
                    : share >= 0.5 ? ["rules", `${attacker} rules ${place}`]
                        : ["biggest-power-in", `${attacker} is the biggest power in ${place}`],
            ]),
        }
    }

    if (story.kind === "round" && story.day !== undefined) {
        const day = story.day
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
            link: SITE,
            ...named([["takes-the-day", `${attacker} takes ${won}`], ["day-goes-to", `${won} goes to ${attacker}`]]),
        }
    }

    if (story.kind === "rout") {
        const loser = said(nameOf(sides[0]))
        const home = inPlace(story.place, sides[0], regionOf)
        const ownCountry = "country" in story.place && story.place.country === sides[0]
        const takerAtHome = "country" in story.place && story.place.country === story.attacker && story.team === undefined
        return {
            headline: headline ?? (ownCountry ? `${loser} IS FALLING`
                : home ? `${loser} IS LOSING ${place}` : `${loser} GETS KICKED OUT OF ${place}`).toUpperCase(),
            line: undefined,
            call: sides.length === 2 ? "PICK A SIDE" : `FIGHT FOR ${loser}`.toUpperCase(),
            callFlags: sides,
            link: sides.length === 2 ? SITE : `${SITE}/?f=${sides[0]}`,
            ...named(ownCountry ? [["the-fall-of", `the fall of ${loser}`], ["collapses", `${loser} collapses`]]
                : home ? [["loses-ground-in", `${loser} loses ground in ${place}`], ["in-trouble-in", `${loser} is in trouble in ${place}`]]
                    : takerAtHome ? [["retakes-its-land-from", `${attacker} retakes its land from ${loser}`],
                        ["generational-downfall", `generational downfall of ${loser}`]]
                        : [["generational-downfall", `generational downfall of ${loser}`], ["loses-place", `${loser} loses ${place}`]]),
        }
    }

    if (story.team !== undefined) {
        const team = said(story.team)
        return {
            headline: headline ?? `${team} STRIKES BACK`.toUpperCase(),
            line: undefined,
            call: `FIGHT FOR ${team}`.toUpperCase(),
            callFlags: sides,
            link: SITE,
            ...named(victim === undefined ? [["fights-back", `${team} fights back`], ["unites", `${team} unites`]]
                : [["unites-against", `${team} unites against ${victim}`], ["fights-back-against", `${team} fights back against ${victim}`]]),
        }
    }

    if (sides.length === 2) {
        const other = said(nameOf(sides[1]))
        return {
            headline: headline ?? (story.kind === "kickout"
                ? `${attacker} IS KICKING ${other} OUT OF ${place}`
                : `${attacker} VS ${other}`).toUpperCase(),
            line: story.kind === "battle" ? `The battle for ${place}` : undefined,
            call: "PICK A SIDE",
            callFlags: sides,
            link: SITE,
            ...named(story.kind === "kickout"
                ? [["takes-place-from", `${attacker} takes ${place} from ${other}`], ["loses-place-to", `${other} loses ${place} to ${attacker}`]]
                : [["war-for", `war for ${place}: ${attacker} vs ${other}`], ["fight-for", `${attacker} and ${other} fight for ${place}`]]),
        }
    }

    if (story.kind === "comeback") {
        const ownCountry = "country" in story.place && story.place.country === story.attacker
        return {
            headline: headline ?? `${attacker} STRIKES BACK`.toUpperCase(),
            line: undefined,
            call: `FIGHT FOR ${attacker}`.toUpperCase(),
            callFlags: [story.attacker],
            link: `${SITE}/?f=${story.attacker}`,
            ...named(ownCountry && victim !== undefined
                ? [["retakes-its-land-from", `${attacker} retakes its land from ${victim}`], ["fights-back-against", `${attacker} fights back against ${victim}`]]
                : [["fights-back-in", `${attacker} fights back in ${place}`], ["is-back-in", `${attacker} is back in ${place}`]]),
        }
    }

    const world = "region" in story.place && story.place.region === THE_WORLD
    const defended = world ? undefined : countriesOf(story.place)[0] ?? story.victims[0]
    const kicked = victim !== undefined && !countriesOf(story.place).includes(story.victims[0])
    return {
        headline: headline ?? (world ? `${attacker} IS TAKING OVER ${place}`
            : story.kind === "invasion" ? `${attacker} IS INVADING ${place}`
                : `${attacker} IS ATTACKING ${place}`).toUpperCase(),
        line: undefined,
        call: world ? "FIGHT BACK" : `DEFEND ${place}`.toUpperCase(),
        callFlags: placeFlags(story.place),
        link: defended === undefined ? SITE : `${SITE}/?f=${defended}`,
        ...named(world ? [["takes-over-the-world", `${attacker} takes over the world`], ["nobody-can-stop", `nobody can stop ${attacker}`]]
            : story.kind === "invasion"
                ? [kicked ? ["invades-and-kicks-out", `${attacker} invades ${place} and kicks out ${victim}`] : ["invades", `${attacker} invades ${place}`],
                    ["under-attack-from", `${place} is under attack from ${attacker}`]]
                : [["takes-over", `${attacker} takes over ${place}`], ["under-attack-from", `${place} is under attack from ${attacker}`]]),
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

// Luckiest Guy draws Latin-1 and little more: "ŁOTWA" in it would mix two fonts in one word.
const DISPLAY_FONT_DRAWS = /^[\u0020-\u00ff\u0131\u0152\u0153\u2000-\u206f\u20ac\u2122]*$/u

// A share of a place, to a tenth of a point, never more than it is: 79.96% is 79.9%.
function shareOf(count: number, ground: number): string {
    return `${(Math.floor(count / Math.max(1, ground) * 1000) / 10).toFixed(1)}%`
}

const TOP_FIGHTERS = "TOP FIGHTERS"

// The call fades in as the counter fades out, and the end card only once the counter is gone: it covers where the
// counter was.
const FADE_SECONDS = 0.5

const faded = (seconds: number) => Math.min(1, Math.max(0, seconds / FADE_SECONDS))

function tilesSaid(count: number): string {
    return `${NUMBER.format(count)} ${count === 1 ? "tile" : "tiles"}`
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
    const said = [
        words.headline, words.call, words.when ?? "", ...(words.table ?? []).map(({name, points}) => `${name} ${points}`),
        ...speech === undefined ? [] : [0, 1, 2, 5, 22, 1234].map(speech.tiles),
        ...named.length === 0 ? [] : [TOP_FIGHTERS],
        ...named.flatMap(({fighters}) => fighters.map(({name, took}) => `${name} ${tilesSaid(took)}`)),
    ]
    if (!said.every((text) => DISPLAY_FONT_DRAWS.test(text))) root.classList.add("clip--fallback-font")

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
        show({replayAt, played, held, ending: into}) {
            const over = faded(into)
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
            progress.style.transform = `scaleX(${played})`
            ending.style.opacity = String(over)
            top.style.opacity = String(1 - over)
            bottom.style.opacity = String(1 - over)
            if (card !== undefined) card.style.opacity = String(faded(into - FADE_SECONDS))
        },
        fit() {
            for (const name of names) {
                if (name.scrollWidth > name.clientWidth) name.style.fontSize = `${Math.floor(name.clientWidth / name.scrollWidth * 100) / 100}em`
            }
        },
    }
}
