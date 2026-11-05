import {AlertsBackend, DiscordLink} from "../../backends/alerts.ts"

// A player connects Discord by signing in with it: the same trip joins the game's server, with its flag's role.
export class DiscordAlerts {
    constructor(
        private readonly backend: AlertsBackend,
        private readonly signIn: () => Promise<void>,
        private joinAsked: boolean,
    ) {
    }

    public read = (): Promise<DiscordLink> => this.backend.discordLink()

    public connect = (): Promise<void> => this.signIn()

    // The page was opened to join (?discord=join): the first offer to connect starts the sign-in, once.
    public joinIfAsked = (): void => {
        if (!this.joinAsked) return
        this.joinAsked = false
        this.signIn().catch((e) => console.error("Could not start the Discord sign-in", e))
    }
}
