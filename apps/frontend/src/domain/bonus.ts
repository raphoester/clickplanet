import {TollStep} from "./toll.ts"

export type BonusReward =
    | {
    kind: "refill"
    refills: number
} | {
    kind: "spreadClicks"
    clicks: number
} | {
    kind: "bomb"
    radius: number
    bombs: number
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
    | {
    kind: "virus"
    viruses: number
}
    | {
    kind: "mine"
    mines: number
}
    | {
    kind: "rally"
    rallies: number
}
    | {
    kind: "trap"
    traps: number
}

// Every kind is a count, up to what a hand holds of it (BonusRules).
export type Charges = {
    refills: number
    bombs: number
    enclosures: number
    spreadClicksLeft: number
    shields: number
    viruses: number
    mines: number
    rallies: number
    traps: number
}

export const NO_CHARGES: Charges = {
    refills: 0, bombs: 0, enclosures: 0, spreadClicksLeft: 0, shields: 0, viruses: 0, mines: 0, rallies: 0, traps: 0,
}

export type ChargeKind = BonusReward["kind"]

export const CHARGE_KINDS: readonly ChargeKind[] = [
    "refill", "bomb", "spreadClicks", "encloseClicks", "shields", "virus", "mine", "rally", "trap",
]

// A guest's level, and an account's before it has one: it holds only the open kinds.
export const NO_LEVEL = 0

export type UnlockLevels = Readonly<Partial<Record<ChargeKind, number>>>

export type Gates = {
    on: boolean
    open: readonly ChargeKind[]
    unlocks: UnlockLevels
}

export const NO_GATES: Gates = {on: false, open: [], unlocks: {}}

// Progression's half of the gates: the player's level and the level each gated kind opens at.
export type Ladder = {
    level: number
    unlocks: UnlockLevels
}

export const NO_LADDER: Ladder = {level: NO_LEVEL, unlocks: {}}

export function gatesOf(rules: Pick<BonusRules, "gatesOn" | "openKinds"> | undefined, ladder: Ladder): Gates {
    return {on: rules?.gatesOn ?? false, open: rules?.openKinds ?? [], unlocks: ladder.unlocks}
}

// The level of a gated kind the ladder names none for: it opens at no level.
export const NEVER = Number.POSITIVE_INFINITY

function openingLevel(kind: ChargeKind, gates: Gates): number {
    return gates.open.includes(kind) ? NO_LEVEL : gates.unlocks[kind] ?? NEVER
}

// The level a gated kind opens at, NEVER for one the ladder omits, while the player is below it.
export function lockedAt(kind: ChargeKind, gates: Gates, level: number): number | undefined {
    if (!gates.on) return undefined
    const at = openingLevel(kind, gates)
    return at > level ? at : undefined
}

// The order given while no open kind is told: the ladder alone would put the gated kinds first.
export function inUnlockOrder(kinds: readonly ChargeKind[], gates: Gates): ChargeKind[] {
    if (gates.open.length === 0) return [...kinds]
    return [...kinds].sort((a, b) => {
        const [x, y] = [openingLevel(a, gates), openingLevel(b, gates)]
        return Number(x > y) - Number(x < y)
    })
}

export type Switches = {
    spread: boolean
    enclose: boolean
    shield: boolean
}

export const ALL_OFF: Switches = {spread: false, enclose: false, shield: false}

export function switched(switches: Switches, name: keyof Switches, on: boolean): Switches {
    return on ? {...ALL_OFF, [name]: true} : {...switches, [name]: false}
}

export type SwitchRules = Pick<BonusRules, "spreadUnlimited" | "enclosuresUnlimited" | "enclosuresOff" | "spreadOff">

export function switchesHeld(switches: Switches, charges: Charges, rules: SwitchRules | undefined): Switches {
    const spread = switches.spread && !rules?.spreadOff && (!!rules?.spreadUnlimited || charges.spreadClicksLeft > 0)
    const enclose = switches.enclose && !rules?.enclosuresOff && (!!rules?.enclosuresUnlimited || charges.enclosures > 0)
    const shield = switches.shield && charges.shields > 0
    if (spread === switches.spread && enclose === switches.enclose && shield === switches.shield) return switches
    return {spread, enclose, shield}
}

// The switch on while no other is: an unlimited one, spread before enclose, never one while it is off.
export function restingSwitch(rules: SwitchRules | undefined): "spread" | "enclose" | undefined {
    if (rules?.spreadUnlimited && !rules.spreadOff) return "spread"
    if (rules?.enclosuresUnlimited && !rules.enclosuresOff) return "enclose"
    return undefined
}

export function atRest(switches: Switches, resting: keyof Switches | undefined): Switches {
    if (!resting || switches.spread || switches.enclose || switches.shield) return switches
    return {...switches, [resting]: true}
}

export function sameSwitches(a: Switches, b: Switches): boolean {
    return a.spread === b.spread && a.enclose === b.enclose && a.shield === b.shield
}

export type BonusNotice =
    "shieldFull" | "shieldCapital" | "shieldTaken" | "shieldNotYours" | "nothingEnclosed" | "virusNoEnemy" | "mineTaken" | "mineNotYours"
    | "rallyOn" | "trapTaken" | "trapNotYours"

export type BonusNoticeEvent = {
    notice: BonusNotice
    seq: number
}

export function slotOf(notice: BonusNotice): ChargeKind {
    switch (notice) {
        case "nothingEnclosed":
            return "encloseClicks"
        case "virusNoEnemy":
            return "virus"
        case "mineTaken":
        case "mineNotYours":
            return "mine"
        case "rallyOn":
            return "rally"
        case "trapTaken":
        case "trapNotYours":
            return "trap"
        case "shieldFull":
        case "shieldCapital":
        case "shieldTaken":
        case "shieldNotYours":
            return "shields"
    }
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
    spreadUnlimited: boolean
    // No enclosure closes a shape or is offered: the page hides the enclose slot.
    enclosuresOff: boolean
    // No click spreads, and no spread click is offered: the page hides the spread slot.
    spreadOff: boolean
    // No rally is called or offered: the page hides the rally slot and puts its aim away.
    ralliesOff: boolean
    // The most a virus infects, a tile taken or a shield struck each.
    virusInfections: number
    // How many refills, bombs and viruses a hand holds.
    refills: number
    bombs: number
    viruses: number
    // Bonuses' gates: while on, a kind not open is locked under the level progression's ladder opens it at.
    gatesOn: boolean
    openKinds: readonly ChargeKind[]
    // How many mines a hand holds, and the rings of tiles around its own a mine's blast takes.
    mines: number
    mineRings: number
    // How many rallies a hand holds; a ring's radius in radians and how long it lasts; what a click inside one of the
    // flag's takes from the bank.
    rallies: number
    rallyRadius: number
    rallyLastsMs: number
    rallyCost: number
    // How many traps a hand holds, and the clicks a trap takes from the bank of the player whose take sprang it.
    traps: number
    trapClicks: number
}
