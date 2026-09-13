/**
 * The tiles this client clicked a moment ago, so a broadcast can be told apart
 * as the answer to one of them.
 *
 * A spread click and a boosted click reach every client alike, with a country
 * and a tile but no word of who made them. The client that made one clicked
 * that tile, for that country, a round trip earlier — which is what this
 * remembers. Another player of the same country clicking the same tile inside
 * the window is taken for this one, and that is rare enough to live with.
 */
export class OwnClicks {
    private readonly clicks = new Map<number, {country: string, at: number}>()

    constructor(
        /** How long after a click its broadcast is still taken to be it, in seconds. */
        private readonly windowSeconds: number,
        /** Past this many tiles remembered, the stale ones are dropped. */
        private readonly limit = 256,
    ) {
    }

    record(tile: number, country: string, seconds: number) {
        this.clicks.set(tile, {country, at: seconds})
        if (this.clicks.size <= this.limit) return
        for (const [id, click] of this.clicks) {
            if (seconds - click.at > this.windowSeconds) this.clicks.delete(id)
        }
    }

    /** Whether this client clicked `tile` for `country` within the window. */
    has(tile: number, country: string, seconds: number): boolean {
        const click = this.clicks.get(tile)
        return click !== undefined && click.country === country && seconds - click.at <= this.windowSeconds
    }
}
