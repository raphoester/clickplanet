import {DISCORD_INVITE} from "../links.ts"
import {AlertsBackend, DiscordLink} from "./alerts.ts"

// Offered, and linked once the fake account signed in with a provider: in fake mode every sign-in is the Discord one.
export class FakeAlertsBackend implements AlertsBackend {
    constructor(private readonly signedIn: () => Promise<boolean>) {
    }

    public async discordLink(): Promise<DiscordLink> {
        return {offered: true, linked: await this.signedIn(), serverUrl: DISCORD_INVITE}
    }
}
