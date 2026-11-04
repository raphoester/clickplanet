import {useEffect} from "react"
import {create} from "zustand"
import {AccountBadge, Badge, BadgesBackend, MAX_BADGES_ASKED} from "../../backends/badges.ts"
import {PlayerTitle} from "../../backends/title.ts"

// A badge read this long ago is read again the next time a name shows it.
export const BADGE_FRESH_MS = 5 * 60_000
// A read that failed is tried again, for the same accounts, no sooner than this.
export const BADGE_RETRY_MS = 60_000
// The accounts asked within this wait, a frame, go out in one call.
export const BADGE_BATCH_MS = 16

// An account read with nothing to show: unknown, a guest, deleted, or with neither a streak nor a title.
export const NO_BADGE: Badge = {streak: 0}

type BadgesStore = {
    // By account id, what the last read or the page's own change said. An account never read is absent.
    badges: Readonly<Record<string, Badge>>
    // Reads the account's badge, with every other account asked in the same frame, unless it is fresh or being read.
    ask: (accountId: string) => void
    // Where the badges are read from. None stops reading and forgets every badge.
    serve: (backend: BadgesBackend | undefined) => void
    // What the page learns of an account as it happens: it changed the title worn, or it read the streak.
    wear: (accountId: string, title: PlayerTitle | undefined) => void
    setStreak: (accountId: string, streak: number) => void
    // The account changed in a way the page cannot tell: it is read again, now if a name shows it.
    expire: (accountId: string) => void
}

export const useBadges = create<BadgesStore>()((set, get) => {
    let backend: BadgesBackend | undefined
    // A read answers only for the backend it was sent to.
    let generation = 0
    // When each account read is to be read again.
    const due = new Map<string, number>()
    // When the page last learned something of an account itself: a read sent before says less.
    const learned = new Map<string, number>()
    const reading = new Set<string>()
    const asked = new Set<string>()
    let timer: ReturnType<typeof setTimeout> | undefined

    const schedule = () => {
        timer ??= setTimeout(flush, BADGE_BATCH_MS)
    }

    const flush = () => {
        timer = undefined
        const from = backend
        if (!from) return
        const ids = [...asked]
        asked.clear()
        for (let i = 0; i < ids.length; i += MAX_BADGES_ASKED) read(from, ids.slice(i, i + MAX_BADGES_ASKED))
    }

    const read = (from: BadgesBackend, ids: string[]) => {
        const sentAt = Date.now()
        const sent = generation
        ids.forEach((id) => reading.add(id))
        from.badges(ids).then(
            (found) => {
                if (sent !== generation) return
                keep(ids, found, sentAt)
            },
            (e) => {
                if (sent !== generation) return
                console.error("Could not read the badges", e)
                const retryAt = Date.now() + BADGE_RETRY_MS
                ids.forEach((id) => {
                    reading.delete(id)
                    due.set(id, retryAt)
                })
            },
        )
    }

    const keep = (ids: readonly string[], found: readonly AccountBadge[], sentAt: number) => {
        const freshUntil = Date.now() + BADGE_FRESH_MS
        const byId = new Map(found.map((badge) => [badge.accountId, badge]))
        const kept: Record<string, Badge> = {}
        for (const id of ids) {
            reading.delete(id)
            if ((learned.get(id) ?? -Infinity) >= sentAt) continue
            due.set(id, freshUntil)
            const read = byId.get(id)
            kept[id] = read ? {streak: read.streak, wornTitle: read.wornTitle} : NO_BADGE
        }
        update(kept)
    }

    // Only what changed: a badge read again the same keeps its object, so nothing that shows it renders.
    const update = (next: Record<string, Badge>) => set((store) => {
        const changed = Object.entries(next).filter(([id, badge]) => !sameBadge(store.badges[id], badge))
        return changed.length === 0 ? store : {badges: {...store.badges, ...Object.fromEntries(changed)}}
    })

    const learn = (accountId: string, change: (badge: Badge) => Badge) => {
        if (accountId === "") return
        learned.set(accountId, Date.now())
        update({[accountId]: change(get().badges[accountId] ?? NO_BADGE)})
    }

    return {
        badges: {},
        ask: (accountId) => {
            if (accountId === "" || asked.has(accountId) || reading.has(accountId)) return
            if ((due.get(accountId) ?? 0) > Date.now()) return
            asked.add(accountId)
            if (backend) schedule()
        },
        serve: (next) => {
            if (next !== undefined && next === backend) return
            backend = next
            generation++
            clearTimeout(timer)
            timer = undefined
            due.clear()
            learned.clear()
            reading.clear()
            if (!next) {
                asked.clear()
                set({badges: {}})
                return
            }
            if (asked.size > 0) schedule()
        },
        wear: (accountId, title) => learn(accountId, (badge) => ({...badge, wornTitle: title})),
        setStreak: (accountId, streak) => learn(accountId, (badge) => ({...badge, streak})),
        expire: (accountId) => {
            if (accountId === "") return
            due.delete(accountId)
            learned.set(accountId, Date.now())
            if (get().badges[accountId]) get().ask(accountId)
        },
    }
})

// The badge drawn beside an account's name: undefined until it is read, and read again once BADGE_FRESH_MS old. An
// empty id, a message's author kept before accounts or deleted since, has none.
export function useBadge(accountId: string): Badge | undefined {
    const badge = useBadges((store) => accountId === "" ? undefined : store.badges[accountId])
    // Every render asks: a fresh badge costs nothing, and one gone stale is read again as the name shows it.
    useEffect(() => useBadges.getState().ask(accountId))
    return badge
}

// The streak the page read of its own account, beside its name everywhere.
export function useOwnStreak(accountId: string, streak: number | undefined): void {
    useEffect(() => {
        if (streak !== undefined) useBadges.getState().setStreak(accountId, streak)
    }, [accountId, streak])
}

function sameBadge(a: Badge | undefined, b: Badge): boolean {
    return a !== undefined && a.streak === b.streak && sameTitle(a.wornTitle, b.wornTitle)
}

function sameTitle(a: PlayerTitle | undefined, b: PlayerTitle | undefined): boolean {
    if (a === undefined || b === undefined) return a === b
    return a.id === b.id && a.name === b.name && a.feat === b.feat
        && a.rank?.trackId === b.rank?.trackId && a.rank?.trackName === b.rank?.trackName
        && a.rank?.number === b.rank?.number && a.rank?.count === b.rank?.count
}
