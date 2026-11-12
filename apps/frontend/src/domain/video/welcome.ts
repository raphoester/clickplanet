// One line for a newcomer who came by a player's address: the recruiter's name only as the clips let a video show
// it, none otherwise.
export function welcomeLineOf(recruiter: string | undefined, shown: ReadonlyMap<string, string>, flag: string): string | undefined {
    if (recruiter === undefined) return undefined
    const name = shown.get(recruiter)
    return name === undefined ? undefined : `${name} called you to fight for ${flag}`
}
