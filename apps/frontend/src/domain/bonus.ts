/**
 * What catching a bonus box is worth: a charge, kept until it is spent.
 *
 * This is data, not drawing, so it sits in the domain rather than beside the
 * WebGL that spawns the box or the React that announces it — both read it, and
 * neither owns it. **The client never decides a reward**: the server grants it
 * and says what it granted, and this is the shape that answer arrives in.
 */
export type BonusReward =
    | {
    /**
     * A charge: fills the click bank to full, when the player chooses. The
     * bank's size and the fill are the server's.
     */
    kind: "refill"
} | {
    /**
     * Spread clicks added to the pool: while spread is switched on, each click
     * also takes the tiles touching the one clicked. The server picks those
     * tiles and sends them down the stream, so nothing here knows which they are.
     */
    kind: "spreadClicks"
    clicks: number
} | {
    /**
     * A charge: one bomb, kept until it is dropped. It clears every tile within
     * `radius` radians of arc of where it lands. `radius` is only what the
     * aiming ring is drawn at: the server picks the tiles.
     */
    kind: "bomb"
    radius: number
}
    | {
    /**
     * Enclosures added to the stack: while enclose is switched on, a click that
     * closes a shape of the player's own tiles also takes the tiles inside it,
     * and spends one. The server finds the shape; this only says how big it may be.
     */
    kind: "encloseClicks"
    shapes: number
    /** The most tiles one shape may hold. */
    maxTiles: number
}

/**
 * The bonuses the player holds, none of them on a clock. The server says them
 * once, at load and when the account changes; after that the client follows
 * its own calls (see PlanetBackend), so a charge spent in another tab shows
 * here until the next read.
 */
export type Charges = {
    refill: boolean
    bomb: boolean
    /** Shapes that can still be closed, up to `BonusRules.enclosures`. */
    enclosures: number
    /** The spread pool, up to `BonusRules.spreadClicks`. */
    spreadClicksLeft: number
}

export const NO_CHARGES: Charges = {refill: false, bomb: false, enclosures: 0, spreadClicksLeft: 0}

/** The kinds a charge can be. */
export type ChargeKind = BonusReward["kind"]

/**
 * The bonuses the player has switched on. Off by default: a charge is spent
 * only when the player asks. A click carries them, and the server spends one
 * only when its switch is on. One at most: the server refuses a click with both.
 */
export type Switches = {
    spread: boolean
    enclose: boolean
}

export const ALL_OFF: Switches = {spread: false, enclose: false}

/**
 * The switches with `name` turned on or off. One bonus at a time: turning one
 * on turns the other off, since a click spreads or encloses, never both.
 */
export function switched(switches: Switches, name: keyof Switches, on: boolean): Switches {
    return on ? {...ALL_OFF, [name]: true} : {...switches, [name]: false}
}

/**
 * The switches, with any whose pool is empty turned off: a switch on over
 * nothing would say the next click does something it will not. Hands back the
 * same object when nothing changes.
 */
export function switchesHeld(switches: Switches, charges: Charges): Switches {
    const spread = switches.spread && charges.spreadClicksLeft > 0
    const enclose = switches.enclose && charges.enclosures > 0
    if (spread === switches.spread && enclose === switches.enclose) return switches
    return {spread, enclose}
}

/**
 * How big each charge is and how many can be held: the same for every player,
 * read once at load. Game configuration rather than state, so it is not asked
 * again; a page open across a change of rules shows the old sizes until it is
 * reloaded.
 */
export type BonusRules = {
    /** Radians of arc: the aiming ring is drawn at the size of what it will clear. */
    blastRadius: number
    enclosureMaxTiles: number
    /** The most spread clicks held. */
    spreadClicks: number
    /** The most enclosures held. */
    enclosures: number
}

/**
 * The words for a reward, in the two lengths the screen needs them: shouted in
 * the middle of the screen, and explained under it.
 *
 * Kept in one place so a second kind of reward is one case here rather than an
 * edit in every component that mentions it.
 */
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
