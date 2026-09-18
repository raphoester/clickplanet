import type {Presence} from "../backends/player.ts"

/** The server drops a player 90s after its last announce, so three chances to be heard. */
export const ANNOUNCE_EVERY_MS = 30_000

/**
 * How long the flag and the username must hold still before a change is announced.
 * Picking a country is a few clicks through a list; one announce says where the
 * player landed, not each step on the way.
 */
export const SETTLE_MS = 1_000

/** What an announce says, and the username it says it under. */
export type Announcing = Presence & {
    /** Not sent — the server reads it off the token — but a new one is worth an announce. */
    username?: string
}

/**
 * When to announce, and nothing else: no clock, no timer, no network, like
 * `SessionClient`. The hook asks `claim` on a short tick and sends whatever it
 * hands back.
 *
 * An announce is due:
 * - **as soon as a token is held that the last one did not go out under** —
 *   the first click's mint, the mint after a refused token, and the one after a
 *   sign-in, which names another account;
 * - once the flag or the username has held still `SETTLE_MS`
 *   after changing;
 * - `ANNOUNCE_EVERY_MS` after the last one.
 *
 * A failed announce counts as one: it is tried again on the next of those, not
 * on the next tick, which would hammer a server that is down.
 */
export class PresenceSchedule {
    private wanted: Announcing
    private changedAt = -Infinity
    private last?: {key: string, session: string, at: number}
    private sending = false

    constructor(wanted: Announcing) {
        this.wanted = wanted
    }

    /** The flag and the username, as they are now. */
    public want(next: Announcing, now: number): void {
        if (keyOf(next) === keyOf(this.wanted)) return
        this.wanted = next
        this.changedAt = now
    }

    /**
     * What to announce now, marked as sent, or undefined when nothing is due.
     * `session` is the token held right now, never a minted one: without one
     * nothing is due. One announce at a time; `settle` ends it.
     */
    public claim(now: number, session: string | undefined): Presence | undefined {
        if (session === undefined || this.sending) return undefined
        if (!this.isDue(now, session)) return undefined

        this.last = {key: keyOf(this.wanted), session, at: now}
        this.sending = true
        return {countryCode: this.wanted.countryCode}
    }

    /** The announce `claim` handed out has landed or failed. */
    public settle(): void {
        this.sending = false
    }

    private isDue(now: number, session: string): boolean {
        const last = this.last
        if (!last || last.session !== session) return true
        if (last.key !== keyOf(this.wanted)) return now - this.changedAt >= SETTLE_MS
        return now - last.at >= ANNOUNCE_EVERY_MS
    }
}

function keyOf(announcing: Announcing): string {
    return JSON.stringify([announcing.countryCode, announcing.username ?? null])
}
