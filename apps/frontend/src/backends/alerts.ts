export type DiscordLink = {
    offered: boolean
    linked: boolean
    // The server's page in Discord, where the pings are. Empty when not offered.
    serverUrl: string
}

export const NOT_OFFERED: DiscordLink = {offered: false, linked: false, serverUrl: ""}

// A player connects Discord by signing in with it, which joins the server too: this only says whether it did.
export interface AlertsBackend {
    discordLink(): Promise<DiscordLink>
}
