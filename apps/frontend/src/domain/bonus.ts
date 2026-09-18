/**
 * What catching a bonus box is worth: a charge, kept until it is spent, one of
 * each kind at most.
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
     * A charge: the next `clicks` clicks also take the tiles touching the one
     * clicked. The server picks those tiles and sends them down the stream, so
     * nothing here knows which they are.
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
     * A charge: the next click that closes a shape of the player's own tiles
     * also takes the tiles inside it. The server finds the shape; this only
     * says how big it may be.
     */
    kind: "encloseClicks"
    /** The most tiles the shape may hold. */
    maxTiles: number
}

/**
 * The use-once bonuses the player holds. At most one of each kind, each kept
 * until it is spent — there is no clock on any of them, so nothing here counts
 * down. The server says them once, at load and when the account changes; after
 * that the client follows its own calls (see PlanetBackend), so a charge spent
 * in another tab shows here until the next read.
 */
export type Charges = {
    refill: boolean
    bomb: boolean
    enclose: boolean
    /** Zero is no spread charge. */
    spreadClicksLeft: number
}

export const NO_CHARGES: Charges = {refill: false, bomb: false, enclose: false, spreadClicksLeft: 0}

/** The kinds a charge pill can be. */
export type ChargeKind = BonusReward["kind"]

/**
 * How big each charge is: the same for every player, read once at load. Game
 * configuration rather than state, so it is not asked again; a page open
 * across a change of rules shows the old sizes until it is reloaded.
 */
export type BonusRules = {
    /** Radians of arc: the aiming ring is drawn at the size of what it will clear. */
    blastRadius: number
    enclosureMaxTiles: number
    spreadClicks: number
}

/**
 * What the meter says about each charge held, in the order it shows them: the
 * two the player uses by pressing first, the refill and the bomb.
 */
export function chargeLabels(charges: Charges): {kind: ChargeKind, label: string}[] {
    const labels: {kind: ChargeKind, label: string}[] = []
    if (charges.refill) labels.push({kind: "refill", label: "Refill ready"})
    if (charges.bomb) labels.push({kind: "bomb", label: "Bomb ready"})
    if (charges.enclose) labels.push({kind: "encloseClicks", label: "Enclose ready"})
    if (charges.spreadClicksLeft > 0) {
        const clicks = charges.spreadClicksLeft === 1 ? "1 click" : `${charges.spreadClicksLeft} clicks`
        labels.push({kind: "spreadClicks", label: `Spread: ${clicks} left`})
    }

    return labels
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
                title: "Spread clicks",
                detail: `Your next ${reward.clicks} clicks also take the tiles around them`,
            }
        case "bomb":
            return {
                title: "Bomb",
                detail: "Resets the tiles in an area. Kept until you drop it",
            }
        case "encloseClicks":
            return {
                title: "Enclose",
                detail: `Close a shape of up to ${reward.maxTiles} tiles to take the tiles inside`,
            }
    }
}
