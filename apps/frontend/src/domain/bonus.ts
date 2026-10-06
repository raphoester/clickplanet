import {TollStep} from "./toll.ts"

export type BonusReward =
    | {
    kind: "refill"
} | {
    kind: "spreadClicks"
    clicks: number
} | {
    kind: "bomb"
    radius: number
}
    | {
    kind: "encloseClicks"
    shapes: number
    maxTiles: number
}

export type Charges = {
    refill: boolean
    bomb: boolean
    enclosures: number
    spreadClicksLeft: number
}

export const NO_CHARGES: Charges = {refill: false, bomb: false, enclosures: 0, spreadClicksLeft: 0}

export type ChargeKind = BonusReward["kind"]

export type Switches = {
    spread: boolean
    enclose: boolean
}

export const ALL_OFF: Switches = {spread: false, enclose: false}

export function switched(switches: Switches, name: keyof Switches, on: boolean): Switches {
    return on ? {...ALL_OFF, [name]: true} : {...switches, [name]: false}
}

export function switchesHeld(switches: Switches, charges: Charges): Switches {
    const spread = switches.spread && charges.spreadClicksLeft > 0
    const enclose = switches.enclose && charges.enclosures > 0
    if (spread === switches.spread && enclose === switches.enclose) return switches
    return {spread, enclose}
}

export type BonusRules = {
    blastRadius: number
    enclosureMaxTiles: number
    spreadClicks: number
    enclosures: number
    toll: readonly TollStep[]
}

export function describeReward(reward: BonusReward): {
    title: string
    detail: string
} {
    switch (reward.kind) {
        case "refill":
            return {
                title: "Refill",
                detail: "Fills your clicks to full, when you choose",
            }
        case "spreadClicks":
            return {
                title: reward.clicks === 1 ? "+1 spread click" : `+${reward.clicks} spread clicks`,
                detail: "Switch spread on: each click also takes the tiles around it",
            }
        case "bomb":
            return {
                title: "Bomb",
                detail: "Resets the tiles in an area. Aim it from your inventory",
            }
        case "encloseClicks":
            return {
                title: reward.shapes === 1 ? "+1 enclosure" : `+${reward.shapes} enclosures`,
                detail: `Switch enclose on, then close a shape of up to ${reward.maxTiles} tiles to take the tiles inside`,
            }
    }
}
