const PARAM = "discord"
const JOIN = "join"

// A Discord link outside the game opens it with this: the game starts Discord's sign-in, which joins the server.
export const DISCORD_JOIN_PATH = `/play?${PARAM}=${JOIN}`

export function discordJoinAsked(url: URL): boolean {
    return url.searchParams.get(PARAM) === JOIN
}

export function withoutDiscordJoin(url: URL): URL {
    const stripped = new URL(url)
    stripped.searchParams.delete(PARAM)
    return stripped
}
