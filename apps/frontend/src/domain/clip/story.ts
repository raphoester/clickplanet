import {Countries} from "../countries.ts"
import {flagOfContinent} from "../regions.ts"

export type Place = {country: string} | {countries: [string, string]} | {region: string}

// A round of the season once it closed, a day or a battle's round of its own (own, numbered apart): the countries
// that scored the most in it, and the season's table after it.
export type Day = {
    season: number
    number: number
    finale: boolean
    own: boolean
    endedAt: number
    podium: {country: string, rank: number, points: number}[]
    table: {country: string, rank: number, points: number}[]
}

export type Kind = "attack" | "invasion" | "comeback" | "kickout" | "rout" | "battle" | "holding" | "round" | "callup"

// What a clip tells, as the editor wrote it from the findings, or as the data of the season gives it. team: the continent
// whose flags strike back together. A holding is what the map is rather than what changed on it: its flag (attacker)
// holds share of the place, the flags holding the most after it are its victims. A round is a day of the season its flag
// won, the next places on its podium its victims. A callup is a country that can still win the season after a day,
// called to the Final Battle at finaleAt: the leader its victim. downfall: a rout of a flag from the top of the world to
// far down it, as the replay measured it (downfallOf).
export type Story = {
    kind: Kind
    attacker: string
    rival: string | undefined
    victims: string[]
    place: Place
    team?: string
    share?: number
    day?: Day
    finaleAt?: number
    downfall?: boolean
}

export const KINDS: readonly Kind[] = ["attack", "invasion", "comeback", "kickout", "rout", "battle", "holding", "round", "callup"]

// The continent, as Natural Earth names it.
export type RegionOf = (country: string) => string | undefined

export const THE_WORLD = "the world"

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

// Whether a flag fights on the story's side of the front: what the camera follows and the counter counts. A rout is
// the loser's land going, to whoever takes it; a continent striking back together is every flag of it.
export function fightersOf(story: Story, regionOf: RegionOf): (flag: string | undefined) => boolean {
    if (story.team !== undefined) return (flag) => flag !== undefined && regionOf(flag) === story.team
    if (story.kind === "rout") return (flag) => flag === story.victims[0]
    return (flag) => flag === story.attacker || (story.rival !== undefined && flag === story.rival)
}

// Two stories with one key are one story told twice, whatever their headlines. angle: what the story tells of its flag
// there, none for what it did; round: the day a round tells or a callup follows, "1.5", or the battle, "1.battle.2".
export type StoryKey = {flag: string, continent: string, angle?: "holding" | "round" | "callup", round?: string}

export function storyKeyOf(story: Story, regionOf: RegionOf): StoryKey {
    const place = story.place
    const country = "country" in place ? place.country : "countries" in place ? place.countries[0] : undefined
    const key = {
        flag: sidesOf(story)[0],
        continent: "region" in place ? place.region : country === undefined ? THE_WORLD : regionOf(country) ?? country,
    }
    if (story.kind === "holding") return {...key, angle: "holding"}
    if (story.kind === "round" && story.day !== undefined) return {...key, angle: "round", round: roundKeyOf(story.day)}
    if (story.kind === "callup" && story.day !== undefined) return {...key, angle: "callup", round: roundKeyOf(story.day)}
    return key
}

function roundKeyOf(day: Day): string {
    return day.own ? `${day.season}.battle.${day.number}` : `${day.season}.${day.number}`
}

// Every country a story shows: its place's, its flags' and the flags it takes from. A continent's own flag is not one.
export function storyCountriesOf(story: Story): string[] {
    const place = story.place
    const grounds = "country" in place ? [place.country] : "countries" in place ? place.countries : []
    return [...new Set([...grounds, story.attacker, story.rival, ...story.victims])]
        .filter((code): code is string => code !== undefined && Countries.has(code))
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
