import {AlertsBackend, DiscordLink} from "../../backends/alerts.ts"

export type DiscordAlertsOptions = {
    navigate: (url: string) => void
    remember: () => void
}

export class DiscordAlerts {
    constructor(
        private readonly backend: AlertsBackend,
        private readonly options: DiscordAlertsOptions,
    ) {
    }

    public read = (): Promise<DiscordLink> => this.backend.discordLink()

    public leave = async (): Promise<void> => {
        const url = await this.backend.startDiscordLink()
        this.options.remember()
        this.options.navigate(url)
    }

    public complete = (code: string, state: string): Promise<string> => this.backend.completeDiscordLink(code, state)
}
