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

export type Unlimited = Pick<BonusRules, "spreadUnlimited" | "enclosuresUnlimited">

export function switchesHeld(switches: Switches, charges: Charges, unlimited: Unlimited | undefined): Switches {
    const spread = switches.spread && (!!unlimited?.spreadUnlimited || charges.spreadClicksLeft > 0)
    const enclose = switches.enclose && (!!unlimited?.enclosuresUnlimited || charges.enclosures > 0)
    const shield = switches.shield && charges.shields > 0
    if (spread === switches.spread && enclose === switches.enclose && shield === switches.shield) return switches
    return {spread, enclose, shield}
}

// The switch on while no other is: an unlimited one, spread before enclose.
export function restingSwitch(unlimited: Unlimited | undefined): "spread" | "enclose" | undefined {
    if (unlimited?.spreadUnlimited) return "spread"
    if (unlimited?.enclosuresUnlimited) return "enclose"
    return undefined
}

export function atRest(switches: Switches, resting: keyof Switches | undefined): Switches {
    if (!resting || switches.spread || switches.enclose || switches.shield) return switches
    return {...switches, [resting]: true}
}

export function ringsOf(switches: Switches, rules: BonusRules | undefined): number {
    return switches.spread ? rules?.spreadRings ?? 1 : 0
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
    // A take lands only next to the flag's land or across a water where it holds a coast.
    reach: boolean
    // A capital taken by another flag knocks its flag out.
    knockOut: boolean
    // No shield goes on a capital, and a fortify skips it.
    capitalShieldsFixed: boolean
    // No bomb is offered or dropped: the page hides the bomb.
    bombsOff: boolean
    // A spread takes every tile within this many steps of the tile clicked, and widens the reach as far.
    spreadRings: number
    spreadUnlimited: boolean
}
