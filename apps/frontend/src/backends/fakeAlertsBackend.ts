import {DISCORD_INVITE} from "../links.ts"
import {AlertsBackend, DiscordLink} from "./alerts.ts"
import {CALLBACK_PATH} from "../domain/signInCallback.ts"

// Offered and not linked. Start comes straight back to the callback page, as Discord would after the player agreed.
export class FakeAlertsBackend implements AlertsBackend {
    private linked = false

    public async discordLink(): Promise<DiscordLink> {
        return {offered: true, linked: this.linked}
    }

    public async startDiscordLink(): Promise<string> {
        return `${CALLBACK_PATH}?code=fake-discord&state=fake`
    }

    public async completeDiscordLink(): Promise<string> {
        this.linked = true
        return DISCORD_INVITE
    }
}
