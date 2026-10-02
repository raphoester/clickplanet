import type {Presence} from "../backends/player.ts"

export const ANNOUNCE_EVERY_MS = 30_000

export const SETTLE_MS = 1_000

export type Announcing = Presence & {
    username?: string
}

export class PresenceSchedule {
    private wanted: Announcing
    private changedAt = -Infinity
    private last?: {key: string, session: string, at: number}
    private sending = false

    constructor(wanted: Announcing) {
        this.wanted = wanted
    }

    public want(next: Announcing, now: number): void {
        if (keyOf(next) === keyOf(this.wanted)) return
        this.wanted = next
        this.changedAt = now
    }

    public claim(now: number, session: string | undefined): Presence | undefined {
        if (session === undefined || this.sending) return undefined
        if (!this.isDue(now, session)) return undefined

        this.last = {key: keyOf(this.wanted), session, at: now}
        this.sending = true
        return {countryCode: this.wanted.countryCode}
    }

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
