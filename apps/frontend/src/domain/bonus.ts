/**
 * What catching a bonus box is worth, and how long it lasts.
 *
 * This is data, not drawing, so it sits in the domain rather than beside the
 * WebGL that spawns the box or the React that announces it — both read it, and
 * neither owns it. **The client never decides a reward**: the server grants it
 * and says what it granted, and this is the shape that answer arrives in.
 */
export type BonusReward = {
    /**
     * - `tripleClicks`: the click allowance is multiplied.
     * - `spreadClicks`: every click also takes the tiles touching the one
     *   clicked. The server picks those tiles and sends them down the stream,
     *   so nothing here knows which they are.
     */
    kind: "tripleClicks" | "spreadClicks"
    seconds: number
}

/**
 * A reward that is currently running, with the moment it lapses.
 *
 * `endsAt` is on the same monotonic clock as a `ClickBudget` reading
 * (`performance.now()`), and for the same reason: this counts down against how
 * long *this machine* has watched, not against a server timestamp from an
 * unrelated clock.
 */
export type ActiveBonus = {
    reward: BonusReward
    endsAt: number
}

/** By how much a reward multiplies the click allowance. */
export function multiplierOf(reward: BonusReward): number {
    switch (reward.kind) {
        case "tripleClicks":
            return 3
        case "spreadClicks":
            return 1
    }
}

/**
 * The words for a reward, in the three lengths the screen needs them: shouted
 * in the middle of the screen, explained under it, and squeezed onto the meter.
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
                detail: `${multiplierOf(reward)}× your click rate for ${reward.seconds} seconds`,
                badge: `${multiplierOf(reward)}×`,
            }
        case "spreadClicks":
            return {
                title: "Spread clicks",
                detail: `Every click also takes the tiles around it for ${reward.seconds} seconds`,
                badge: "+6",
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
