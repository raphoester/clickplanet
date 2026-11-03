import {DISCORD_INVITE} from "../links.ts"
import {AlertsBackend, DiscordLink} from "./alerts.ts"

// Offered, and linked once the fake account signed in with a provider: in fake mode every sign-in is the Discord one.
// As the backend's, the join lands joinLandsMs after the first read that finds the account signed in.
export class FakeAlertsBackend implements AlertsBackend {
    private signedInAt?: number

    constructor(
        private readonly signedIn: () => Promise<boolean>,
        private readonly joinLandsMs = 0,
        private readonly now: () => number = Date.now,
    ) {
    }

    public async discordLink(): Promise<DiscordLink> {
        const signedIn = await this.signedIn()
        this.signedInAt = signedIn ? this.signedInAt ?? this.now() : undefined
        const linked = this.signedInAt !== undefined && this.now() - this.signedInAt >= this.joinLandsMs
        return {offered: true, linked, serverUrl: DISCORD_INVITE}
    }
}
