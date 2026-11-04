import {PlayerTitle} from "./title.ts"

// The most accounts one GetBadges call may name: the server refuses more.
export const MAX_BADGES_ASKED = 100

// What is drawn beside a player's name, wherever the name comes from.
export type Badge = {
    // As of today; 0 when it shows none.
    streak: number
    wornTitle?: PlayerTitle
}

export type AccountBadge = Badge & {
    accountId: string
}

export interface BadgesBackend {
    // Only the accounts with a badge to show: an unknown one, a guest, one deleted, or one with nothing to show is left
    // out. At most MAX_BADGES_ASKED ids.
    badges(accountIds: readonly string[]): Promise<AccountBadge[]>
}
