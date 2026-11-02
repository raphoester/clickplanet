import {AlertsBackend, DiscordLink} from "../../backends/alerts.ts"

// A player connects Discord by signing in with it: the same trip joins the game's server, with its flag's role.
export class DiscordAlerts {
    constructor(
        private readonly backend: AlertsBackend,
        private readonly signIn: () => Promise<void>,
    ) {
    }

    public read = (): Promise<DiscordLink> => this.backend.discordLink()

    public connect = (): Promise<void> => this.signIn()
}
