import {ranked, tally, TileChange} from "./changes.ts"
import {Place, RegionOf, Story, THE_WORLD} from "./story.ts"

// Russia's ground is 71% of Natural Earth's Europe: a flag holding Siberia does not hold Europe. Russia is a place of
// its own, and Europe is held without it.
const OUTSIDE_EUROPE = "ru"

// The flag holding the most of the world is a story from a tenth of the map: 26,000 tiles, twenty Frances.
const WORLD_SHARE = 0.1

const CONTINENT_SHARE = 0.4

// A foreign flag on most of a country, not on part of it, which is a war told by the other stories.
const COUNTRY_SHARE = 0.75

// About a France: a smaller country held by a foreign flag is a corner of a war, and 49 countries are this big.
const BIG_COUNTRY = 1000

// A holding is news when its flag fought for it: this many tiles of the place changed hands to it or from it in the
// window, and the clip shows them.
const FOUGHT_LEAST = 50

// The flags holding the most after it, which the clip shows too.
const RIVALS = 2

// A flag holding its own country is no news, wherever it is counted: Australia holding 88% of Oceania on its own ground.
const OWN_GROUND_MOST = 0.5

// The top holder of a place, and why it is no story when it is not.
export type Holding = {story: Story, ground: number, short: string | undefined}

const LEAST: Record<"world" | "continent" | "country", number> = {
    world: WORLD_SHARE, continent: CONTINENT_SHARE, country: COUNTRY_SHARE,
}

// owners: the map as it stands. tiles: how many tiles the map has, ids from 1. fought: what changed hands in the window.
// The flag holding the most of each place, best first: the world, each continent and each big country.
export function holdingsOf(
    owners: ReadonlyMap<number, string>,
    tiles: number,
    groundOf: (tile: number) => string | undefined,
    fought: readonly TileChange[],
    regionOf: RegionOf,
): Holding[] {
    const sizes = tally(Array.from({length: tiles}, (_, i) => groundOf(i + 1)).flatMap((ground) => ground ?? []))
    const keysOf = memo((ground: string | undefined) => placeKeysOf(ground, sizes, regionOf))

    const grounds = new Map<string, number>()
    const held = new Map<string, Map<string, number>>()
    // What each flag holds of each place on its own country's ground.
    const home = new Map<string, Map<string, number>>()
    const add = (counts: Map<string, Map<string, number>>, key: string, flag: string) => {
        const there = counts.get(key) ?? new Map<string, number>()
        there.set(flag, (there.get(flag) ?? 0) + 1)
        counts.set(key, there)
    }
    for (let tile = 1; tile <= tiles; tile++) {
        const owner = owners.get(tile)
        const ground = groundOf(tile)
        for (const key of keysOf(ground)) {
            grounds.set(key, (grounds.get(key) ?? 0) + 1)
            if (owner === undefined) continue
            add(held, key, owner)
            if (owner === ground) add(home, key, owner)
        }
    }

    const tops = [...held].flatMap(([key, there]) => {
        const [first, ...rest] = ranked(there)
        return first === undefined ? [] : [{key, holder: first[0], tiles: first[1], rivals: rest.slice(0, RIVALS).map(([flag]) => flag)}]
    })
    const holderOf = new Map(tops.map(({key, holder}) => [key, holder]))
    const fights = tally(fought.flatMap(({tile, from, to}) => from === to ? [] : keysOf(groundOf(tile)).filter((key) => {
        const holder = holderOf.get(key)
        return holder !== undefined && (holder === from || holder === to)
    })))

    const holdings = tops.map(({key, holder, tiles: count, rivals}) => {
        const {place, scale} = placeOf(key)
        const ground = grounds.get(key) ?? 0
        const share = count / Math.max(1, ground)
        const story: Story = {kind: "holding", attacker: holder, rival: undefined, victims: rivals, place, taken: count, share}
        const short = (home.get(key)?.get(holder) ?? 0) >= count * OWN_GROUND_MOST ? "mostly its own country"
            : share < LEAST[scale] ? `too little, ${Math.round(LEAST[scale] * 100)}% at least`
                : (fights.get(key) ?? 0) < FOUGHT_LEAST ? "it fought for none of it in the window" : undefined
        return {story, ground, short, dominance: share / LEAST[scale]}
    })

    // One holding a flag, where it dominates the most for the place's size: 80% of Europe over 15% of the world, and
    // Antarctica's holder holds the continent rather than the country of the same ground.
    const told = new Set<string>()
    return holdings
        .sort((a, b) => Number(a.short !== undefined) - Number(b.short !== undefined) || b.dominance - a.dominance)
        .map(({story, ground, short}): Holding => {
            if (short !== undefined) return {story, ground, short}
            if (told.has(story.attacker)) return {story, ground, short: "told where it holds more"}
            told.add(story.attacker)
            return {story, ground, short}
        })
        .sort((a, b) => Number(a.short !== undefined) - Number(b.short !== undefined) || b.story.taken - a.story.taken)
}

// Whether a country's ground counts for a holding of a place: what the headline's share is of.
export function inHolding(place: Place, ground: string | undefined, regionOf: RegionOf): boolean {
    if ("region" in place && place.region === THE_WORLD) return true
    if (ground === undefined) return false
    if ("country" in place) return ground === place.country
    if ("countries" in place) return place.countries.includes(ground)
    return ground !== OUTSIDE_EUROPE && regionOf(ground) === place.region
}

function placeKeysOf(ground: string | undefined, sizes: ReadonlyMap<string, number>, regionOf: RegionOf): string[] {
    if (ground === undefined) return [THE_WORLD]
    const continent = ground === OUTSIDE_EUROPE ? undefined : regionOf(ground)
    return [
        THE_WORLD,
        ...continent === undefined ? [] : [`region:${continent}`],
        ...(sizes.get(ground) ?? 0) >= BIG_COUNTRY ? [`country:${ground}`] : [],
    ]
}

function placeOf(key: string): {place: Place, scale: keyof typeof LEAST} {
    if (key === THE_WORLD) return {place: {region: THE_WORLD}, scale: "world"}
    const [scale, name] = key.split(":")
    return scale === "country" ? {place: {country: name}, scale: "country"} : {place: {region: name}, scale: "continent"}
}

function memo<T>(compute: (ground: string | undefined) => T): (ground: string | undefined) => T {
    const known = new Map<string | undefined, T>()
    return (ground) => {
        if (!known.has(ground)) known.set(ground, compute(ground))
        return known.get(ground)!
    }
}
