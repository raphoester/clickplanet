import {ranked, tally, TileChange} from "./changes.ts"

export type Place = {country: string} | {region: string}

export type Story = {
    kind: "attack" | "invasion" | "comeback" | "battle"
    attacker: string
    rival: string | undefined
    victims: string[]
    place: Place
    taken: number
}

// The continent, as Natural Earth names it.
export type RegionOf = (country: string) => string | undefined

const EVEN_FORCES = 0.6

const ONE_COUNTRY = 0.9

const ONE_REGION = 0.7

const VICTIMS = 3

export const THE_WORLD = "the world"

export function storyOf(
    changes: readonly TileChange[],
    groundOf: (tile: number) => string | undefined,
    regionOf: RegionOf,
    attacker?: string,
): Story | undefined {
    const gains = ranked(tally(changes.flatMap(({to}) => to === undefined ? [] : [to])))
    const lead = attacker ?? gains[0]?.[0]
    if (lead === undefined) return undefined

    const taken = gains.find(([flag]) => flag === lead)?.[1] ?? 0
    const challenger = gains.find(([flag]) => flag !== lead)
    const rival = challenger !== undefined && taken > 0 && challenger[1] >= taken * EVEN_FORCES ? challenger[0] : undefined

    const takes = changes.filter(({to}) => to === lead || (rival !== undefined && to === rival))
    const victims = ranked(tally(takes.flatMap(({from}) => from === undefined || from === lead ? [] : [from])))
        .slice(0, VICTIMS)
        .map(([flag]) => flag)

    const place = placeOf(takes.map(({tile}) => groundOf(tile)), regionOf)
    const kind = rival !== undefined ? "battle"
        : !("country" in place) ? "attack"
            : place.country === lead ? "comeback" : "invasion"

    return {kind, attacker: lead, rival, victims, place, taken}
}

// One country when nearly all of it is there, else the continent most of it is in.
export function placeOf(grounds: readonly (string | undefined)[], regionOf: RegionOf): Place {
    const known = grounds.filter((ground): ground is string => ground !== undefined)
    if (known.length === 0) return {region: THE_WORLD}

    const [country, count] = ranked(tally(known))[0]
    if (count >= known.length * ONE_COUNTRY) return {country}

    const [region, inside] = ranked(tally(known.flatMap((ground) => regionOf(ground) ?? [])))[0] ?? []
    if (region !== undefined && inside !== undefined && inside >= known.length * ONE_REGION) return {region}

    return {region: THE_WORLD}
}
