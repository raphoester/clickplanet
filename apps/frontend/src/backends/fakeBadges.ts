import {AccountBadge, Badge, BadgesBackend, MAX_BADGES_ASKED} from "./badges.ts"

// A fake that knows some accounts' badges: undefined for an account it does not know.
export type FakeBadgeSource = {
    badgeOf(accountId: string): Badge | undefined
}

// GetBadges over the fakes: the first that knows an account answers for it, so the roster, the chat and the boards
// draw one badge for one account. As the server, it refuses too many ids, and leaves out an account unknown or with
// nothing to show.
export class FakeBadges implements BadgesBackend {
    // How many GetBadges calls the page made: `fakeBadges.calls` in the console.
    public calls = 0

    constructor(private readonly sources: readonly FakeBadgeSource[]) {
    }

    public async badges(accountIds: readonly string[]): Promise<AccountBadge[]> {
        this.calls++
        if (accountIds.length > MAX_BADGES_ASKED) throw new Error(`GetBadges asked ${accountIds.length} accounts, at most ${MAX_BADGES_ASKED}`)

        return [...new Set(accountIds)].flatMap((accountId) => {
            const badge = this.sources.map((source) => source.badgeOf(accountId)).find((known) => known !== undefined)
            return badge && (badge.streak > 0 || badge.wornTitle) ? [{accountId, ...badge}] : []
        })
    }
}
