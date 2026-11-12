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
    | {
    kind: "shields"
    shields: number
}

export type Charges = {
    refill: boolean
    bomb: boolean
    enclosures: number
    spreadClicksLeft: number
    shields: number
}

export const NO_CHARGES: Charges = {refill: false, bomb: false, enclosures: 0, spreadClicksLeft: 0, shields: 0}

export type ChargeKind = BonusReward["kind"]

export type Switches = {
    spread: boolean
    enclose: boolean
    shield: boolean
}

export const ALL_OFF: Switches = {spread: false, enclose: false, shield: false}

export function switched(switches: Switches, name: keyof Switches, on: boolean): Switches {
    return on ? {...ALL_OFF, [name]: true} : {...switches, [name]: false}
}

export function switchesHeld(switches: Switches, charges: Charges, enclosuresUnlimited: boolean): Switches {
    const spread = switches.spread && charges.spreadClicksLeft > 0
    const enclose = switches.enclose && (enclosuresUnlimited || charges.enclosures > 0)
    const shield = switches.shield && charges.shields > 0
    if (spread === switches.spread && enclose === switches.enclose && shield === switches.shield) return switches
    return {spread, enclose, shield}
}

export function atRest(switches: Switches, encloseAtRest: boolean): Switches {
    if (!encloseAtRest || switches.spread || switches.enclose || switches.shield) return switches
    return {...switches, enclose: true}
}

export function sameSwitches(a: Switches, b: Switches): boolean {
    return a.spread === b.spread && a.enclose === b.enclose && a.shield === b.shield
}

export type BonusNotice = "shieldFull" | "shieldCapital" | "shieldTaken" | "shieldNotYours" | "nothingEnclosed"

export type BonusNoticeEvent = {
    notice: BonusNotice
    seq: number
}

export function slotOf(notice: BonusNotice): ChargeKind {
    return notice === "nothingEnclosed" ? "encloseClicks" : "shields"
}

export type BonusRules = {
    blastRadius: number
    enclosureMaxTiles: number
    spreadClicks: number
    enclosures: number
    shields: number
    tileShields: number
    toll: readonly TollStep[]
    signInRefill: boolean
    enclosuresUnlimited: boolean
    // A take lands only next to the flag's land or at the far end of a bridge from it.
    reach: boolean
    // A capital taken by another flag knocks its flag out.
    knockOut: boolean
    // No shield goes on a capital, and a fortify skips it.
    capitalShieldsFixed: boolean
    // No bomb is offered or dropped: the page hides the bomb.
    bombsOff: boolean
    // The shields each capital starts an arena with, 0 when the set lays no capital.
    capitalShields: number
}

export function describeReward(reward: BonusReward): {
    title: string
    detail?: string
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
        case "shields":
            return {
                title: reward.shields === 1 ? "+1 shield" : `+${reward.shields} shields`,
                detail: "Switch shield on, then tap your tiles. Each shield stops one enemy click",
            }
    }
}
