import {Countries} from "../countries.ts"
import {flagOfContinent} from "../regions.ts"
import {ranked, tally, TileChange} from "./changes.ts"

export type Place = {country: string} | {countries: [string, string]} | {region: string}

// A day of the season once it closed: the countries that scored the most in it, and the season's table after it.
export type Day = {
    season: number
    number: number
    finale: boolean
    endedAt: number
    podium: {country: string, rank: number, points: number}[]
    table: {country: string, rank: number, points: number}[]
}

// team: the continent whose flags strike back together, when no one flag of them leads. A holding is what the map is
// rather than what changed on it: its flag (attacker) holds share of the place, the flags holding the most after it
// are its victims. A round is a day of the season its flag won, the next places on its podium its victims. A callup is
// a country that can still win the season after a day, called to the Final Battle at finaleAt: the leader its victim.
export type Story = {
    kind: "attack" | "invasion" | "comeback" | "kickout" | "rout" | "battle" | "holding" | "round" | "callup"
    attacker: string
    rival: string | undefined
    victims: string[]
    place: Place
    taken: number
    team?: string
    share?: number
    day?: Day
    finaleAt?: number
}

// The continent, as Natural Earth names it.
export type RegionOf = (country: string) => string | undefined

const EVEN_FORCES = 0.6

// Two flags are at war when this much of what each of them took, it took from the other. Two flags taking as much
// from a third are allies, not a battle; nor is a flag nibbling at one busy taking a third.
const AT_WAR = 0.25

const ONE_COUNTRY = 0.9

const ONE_REGION = 0.7

const VICTIMS = 3

// The flags a clip is about make at least this much of what is taken around them, or they are not its main
// characters. A continent's flags striking back together need more between them.
const MAIN_SHARE = 0.35

const TEAM_SHARE = 0.5

// A continent striking back is told from the side of the flag it throws out, once that flag has lost half of what
// it held there, and held enough there to be thrown out of it.
const ROUT_SHARE = 0.5

const ROUT_LEAST = 300

export const THE_WORLD = "the world"

// The world is too big for a flag to take a third of all taken across it, as on one front: Christmas Island took 28%
// of it in a day, more than twice the next flag. A flag leads it when it took twice as much as any other.
const WORLD_LEAD = 2

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

// Two stories with one key are one story told twice, whatever their headlines. angle: what the story tells of its flag
// there, none for what it did; round: the day a round tells or a callup follows, "1.5".
export type StoryKey = {flag: string, continent: string, angle?: "holding" | "round" | "callup", round?: string}

export function storyKeyOf(story: Story, regionOf: RegionOf): StoryKey {
    const place = story.place
    const country = "country" in place ? place.country : "countries" in place ? place.countries[0] : undefined
    const key = {
        flag: sidesOf(story)[0],
        continent: "region" in place ? place.region : country === undefined ? THE_WORLD : regionOf(country) ?? country,
    }
    if (story.kind === "holding") return {...key, angle: "holding"}
    if (story.kind === "round" && story.day !== undefined) return {...key, angle: "round", round: `${story.day.season}.${story.day.number}`}
    if (story.kind === "callup" && story.day !== undefined) return {...key, angle: "callup", round: `${story.day.season}.${story.day.number}`}
    return key
}

// Every country a story shows: its place's, its flags' and the flags it takes from. A continent's own flag is not one.
export function storyCountriesOf(story: Story): string[] {
    const place = story.place
    const grounds = "country" in place ? [place.country] : "countries" in place ? place.countries : []
    return [...new Set([...grounds, story.attacker, story.rival, ...story.victims])]
        .filter((code): code is string => code !== undefined && Countries.has(code))
}

// heldAtStart: the flag that held most of a country's ground when the story starts.
export function storyOf(
    changes: readonly TileChange[],
    groundOf: (tile: number) => string | undefined,
    regionOf: RegionOf,
    attacker?: string,
    heldAtStart?: (country: string) => string | undefined,
): Story | undefined {
    const gains = ranked(tally(changes.flatMap(({to}) => to === undefined ? [] : [to])))
    const lead = attacker ?? gains[0]?.[0]
    if (lead === undefined) return undefined

    const taken = gains.find(([flag]) => flag === lead)?.[1] ?? 0
    const challenger = gains.find(([flag]) => flag !== lead)
    const rival = challenger !== undefined && taken > 0 && challenger[1] >= taken * EVEN_FORCES
        && atWar(changes, lead, challenger[0]) ? challenger[0] : undefined

    const takes = changes.filter(({to}) => to === lead || (rival !== undefined && to === rival))
    const victims = ranked(tally(takes.flatMap(({from}) => from === undefined || from === lead ? [] : [from])))
        .slice(0, VICTIMS)
        .map(([flag]) => flag)

    const place = placeOf(takes.map(({tile}) => groundOf(tile)), regionOf)
    const home = "country" in place ? place.country === lead
        : "countries" in place ? place.countries.includes(lead)
            : backHome(lead, place.region, takes, regionOf)
    const kind = rival !== undefined ? "battle"
        : home ? "comeback"
            : "region" in place ? "attack"
                : "country" in place && heldAtStart?.(place.country) === lead && victims.length > 0
                    && victims[0] !== place.country ? "kickout" : "invasion"

    return {kind, attacker: lead, rival, victims, place, taken}
}

// around: every change near the story, whoever made it. The story as it is when its flags lead what happens there;
// as its continent's when the flags of one continent strike back together and none of them leads; else nothing.
export function castOf(story: Story, around: readonly TileChange[], regionOf: RegionOf): Story | undefined {
    const takers = tally(around.flatMap(({from, to}) => to === undefined || to === from ? [] : [to]))
    const total = [...takers.values()].reduce((sum, count) => sum + count, 0)
    const shareOf = (among: (flag: string) => boolean) =>
        [...takers].reduce((sum, [flag, count]) => among(flag) ? sum + count : sum, 0) / Math.max(1, total)

    if (total === 0 || shareOf((flag) => flag === story.attacker || flag === story.rival) >= MAIN_SHARE) return story
    if (story.kind !== "comeback" || !("region" in story.place)) return undefined
    const team = story.place.region
    return shareOf((flag) => regionOf(flag) === team) >= TEAM_SHARE ? {...story, team} : undefined
}

export function leadsTheWorld(flag: string, changes: readonly TileChange[]): boolean {
    const takers = ranked(tally(changes.flatMap(({from, to}) => to === undefined || to === from ? [] : [to])))
    const own = takers.find(([taker]) => taker === flag)?.[1] ?? 0
    const next = takers.find(([taker]) => taker !== flag)?.[1] ?? 0
    return own > 0 && own >= next * WORLD_LEAD
}

function atWar(changes: readonly TileChange[], one: string, other: string): boolean {
    const between = (from: string, to: string) => changes.filter((change) => change.from === from && change.to === to).length
    const took = (flag: string) => changes.filter(({to}) => to === flag).length
    return between(other, one) >= took(one) * AT_WAR && between(one, other) >= took(other) * AT_WAR
}

// held: the tiles the flag the story takes most from held in the place, when the story starts and when it ends.
// Once it lost half of it, the story is it losing it: to a continent together, or to many flags none of which leads
// (led false), told from its side; to one attacker, "X IS KICKING Y OUT", unless the place is its own (home): nobody
// is kicked out of their own land. A flag taking its own ground back still strikes back.
export function routOf(story: Story, held: {before: number, after: number}, led = true, home = false): Story {
    if (held.before < ROUT_LEAST || held.after > held.before * (1 - ROUT_SHARE)) return story
    if (story.team !== undefined || !led) return {...story, kind: "rout"}
    return !home && (story.kind === "attack" || story.kind === "invasion") ? {...story, kind: "kickout"} : story
}

// A flag losing land around a front nobody leads, told from scratch: the flag that took the most of it, attacking
// where it lost it. Only a flag thrown out of there makes it a story (routOf); the front's own story, a kickout or a
// battle of other flags, says nothing of it.
export function lossOf(told: Story, loser: string, taker: string, place: Place): Story {
    return {
        kind: "region" in place ? "attack" : "invasion",
        attacker: taker,
        rival: undefined,
        victims: [loser, ...told.victims.filter((victim) => victim !== loser)],
        place,
        taken: told.taken,
    }
}

// Two stories of one flag in one place, or in a place and another inside it, are one story, whoever it beat: Portugal
// taking Germany is part of Portugal taking Europe. So are two about one flag thrown out of one place. Battles are told
// by both sides in their place.
export function sameStory(a: Story, b: Story, placeName: (place: Place) => string, regionOf: RegionOf): boolean {
    if (sameRout(a, b, regionOf)) return true
    // What a flag holds, and a day it won, are told beside what it did, once a flag.
    const told = (story: Story) => story.kind === "holding" || story.kind === "round" || story.kind === "callup"
    if (told(a) || told(b)) return a.kind === b.kind && a.attacker === b.attacker
    // A flag thrown out is its story, whoever took its land.
    if (a.kind === "rout" || b.kind === "rout") return false
    if (a.kind === "comeback" && b.kind === "comeback") return a.attacker === b.attacker
    if (a.kind === "battle" || b.kind === "battle") {
        return a.kind === b.kind && a.attacker === b.attacker && a.rival === b.rival && placeName(a.place) === placeName(b.place)
    }
    return a.attacker === b.attacker && overlaps(a.place, b.place, regionOf)
}

// Of two tellings of one story, the one over more of the map tells it: the continent before the country.
export function widerThan(a: Place, b: Place): boolean {
    const breadth = (place: Place) => "country" in place ? 0 : "countries" in place ? 1 : place.region === THE_WORLD ? 3 : 2
    return breadth(a) > breadth(b)
}

// A flag's tour of the world is told beside its stories on each continent, not instead of them.
function overlaps(a: Place, b: Place, regionOf: RegionOf): boolean {
    const world = (place: Place) => "region" in place && place.region === THE_WORLD
    if (world(a) || world(b)) return world(a) && world(b)
    const countries = (place: Place) => "country" in place ? [place.country] : "countries" in place ? place.countries : []
    if (!("region" in a) && !("region" in b)) return countries(a).some((country) => countries(b).includes(country))
    const regions = (place: Place) => "region" in place ? [place.region] : countries(place).map(regionOf)
    return regions(a).some((region) => regions(b).includes(region))
}

// Two stories about one flag thrown out of one place, or of a place and another inside it, are one story, whoever
// threw it out: Israel kicking Christmas Island out of Germany is part of Christmas Island kicked out of Europe.
export function sameRout(a: Story, b: Story, regionOf: RegionOf): boolean {
    const thrownOut = (story: Story) => story.kind === "rout" || story.kind === "kickout"
    return thrownOut(a) && thrownOut(b) && a.victims[0] === b.victims[0] && overlaps(a.place, b.place, regionOf)
}

// Whether a country's ground is part of a place: what a clip's headline names is where its counter counts, so a flag
// "kicked out of Pakistan" is not shown holding its land next door.
export function inPlace(place: Place, ground: string | undefined, regionOf: RegionOf): boolean {
    if ("region" in place && place.region === THE_WORLD) return true
    if (ground === undefined) return false
    if ("country" in place) return ground === place.country
    if ("countries" in place) return place.countries.includes(ground)
    return regionOf(ground) === place.region
}

// A flag of a continent taking it back from flags from elsewhere is not attacking it: when they lost it at least half
// of what it took there, not only when one of them lost it the most.
function backHome(lead: string, region: string, takes: readonly TileChange[], regionOf: RegionOf): boolean {
    const losers = takes.flatMap(({from}) => from === undefined || from === lead ? [] : [from])
    return regionOf(lead) === region && losers.length > 0
        && losers.filter((loser) => regionOf(loser) !== region).length * 2 >= losers.length
}

// One country when nearly all of it is there, else the continent most of it is in, else the two countries most of
// it is in.
export function placeOf(grounds: readonly (string | undefined)[], regionOf: RegionOf): Place {
    const known = grounds.filter((ground): ground is string => ground !== undefined)
    if (known.length === 0) return {region: THE_WORLD}

    const countries = ranked(tally(known))
    const [country, count] = countries[0]
    if (count >= known.length * ONE_COUNTRY) return {country}

    const [region, inside] = ranked(tally(known.flatMap((ground) => regionOf(ground) ?? [])))[0] ?? []
    if (region !== undefined && inside !== undefined && inside >= known.length * ONE_REGION) return {region}

    const second = countries[1]
    if (second !== undefined && count + second[1] >= known.length * ONE_REGION) return {countries: [country, second[0]]}

    return {region: THE_WORLD}
}
