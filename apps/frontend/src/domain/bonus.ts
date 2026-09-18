/**
 * What catching a bonus box is worth: a triple that runs for a time, or a
 * charge that is kept until it is spent.
 *
 * This is data, not drawing, so it sits in the domain rather than beside the
 * WebGL that spawns the box or the React that announces it — both read it, and
 * neither owns it. **The client never decides a reward**: the server grants it
 * and says what it granted, and this is the shape that answer arrives in.
 */
export type BonusReward =
    | TimedReward
    | {
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

/** The one reward that runs for a time: the clicks refill faster. */
export type TimedReward = {
    kind: "tripleClicks"
    seconds: number
}

export function isTimed(reward: BonusReward): reward is TimedReward {
    return reward.kind === "tripleClicks"
}

/**
 * A timed reward that is running, with the moment it lapses.
 *
 * `endsAt` is on the same monotonic clock as a `ClickBudget` reading
 * (`performance.now()`), and for the same reason: this counts down against how
 * long *this machine* has watched, not against a server timestamp from an
 * unrelated clock.
 */
export type ActiveBonus = {
    reward: TimedReward
    endsAt: number
}

/**
 * The use-once bonuses the player holds. At most one of each kind, each kept
 * until it is spent — there is no clock on any of them, so nothing here counts
 * down. The server says them once, at load and when the account changes; after
 * that the client follows its own calls (see PlanetBackend), so a charge spent
 * in another tab shows here until the next read.
 */
export type Charges = {
    bomb: boolean
    enclose: boolean
    /** Zero is no spread charge. */
    spreadClicksLeft: number
}

export const NO_CHARGES: Charges = {bomb: false, enclose: false, spreadClicksLeft: 0}

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
 * bomb first, since it is the one that waits on the player to use it.
 */
export function chargeLabels(charges: Charges): {kind: "bomb" | "encloseClicks" | "spreadClicks", label: string}[] {
    const labels: {kind: "bomb" | "encloseClicks" | "spreadClicks", label: string}[] = []
    if (charges.bomb) labels.push({kind: "bomb", label: "Bomb ready"})
    if (charges.enclose) labels.push({kind: "encloseClicks", label: "Enclose ready"})
    if (charges.spreadClicksLeft > 0) {
        const clicks = charges.spreadClicksLeft === 1 ? "1 click" : `${charges.spreadClicksLeft} clicks`
        labels.push({kind: "spreadClicks", label: `Spread: ${clicks} left`})
    }

    return labels
}

/** By how much a reward multiplies how fast clicks refill. */
export function multiplierOf(reward: BonusReward): number {
    switch (reward.kind) {
        case "tripleClicks":
            return 3
        case "spreadClicks":
        case "bomb":
        case "encloseClicks":
            return 1
    }
}

/**
 * The words for a reward, in the three lengths the screen needs them: shouted
 * in the middle of the screen, explained under it, and squeezed onto the meter.
 *
 * The explanation says what the bonus does and nothing about how long a triple
 * lasts: the meter counts that down, and it is read in the second the
 * announcement is up.
 *
 * Kept in one place so a second kind of reward is one case here rather than an
 * edit in every component that mentions it.
 */
export function describeReward(reward: BonusReward): {
    title: string
    detail: string
    badge: string
} {
    switch (reward.kind) {
        case "tripleClicks":
            return {
                title: "Triple clicks",
                detail: `Clicks refill ${multiplierOf(reward)}× faster`,
                badge: `${multiplierOf(reward)}×`,
            }
        case "spreadClicks":
            return {
                title: "Spread clicks",
                detail: `Your next ${reward.clicks} clicks also take the tiles around them`,
                badge: "+6",
            }
        case "bomb":
            return {
                title: "Bomb",
                detail: "Resets the tiles in an area. Kept until you drop it",
                badge: "💣",
            }
        case "encloseClicks":
            return {
                title: "Enclose",
                detail: `Close a shape of up to ${reward.maxTiles} tiles to take the tiles inside`,
                badge: "⬡",
            }
    }
}

/**
 * Whole seconds left on a bonus, rounded up so it reads "1s" through the last
 * second rather than sitting on "0s" while it is still running.
 */
export function secondsLeft(bonus: ActiveBonus, at: number): number {
    return Math.max(0, Math.ceil((bonus.endsAt - at) / 1000))
}

/** Whether a bonus has run out at `at`. */
export function hasLapsed(bonus: ActiveBonus, at: number): boolean {
    return at >= bonus.endsAt
}
