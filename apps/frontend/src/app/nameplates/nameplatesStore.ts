import {useEffect} from "react"
import {create} from "zustand"
import {AccountNameplate, MAX_NAMEPLATES_ASKED, Nameplate, NameplatesBackend} from "../../backends/nameplates.ts"
import {NameColor} from "../../backends/player.ts"

// A nameplate read this long ago is read again the next time a name shows it.
export const NAMEPLATE_FRESH_MS = 5 * 60_000
// A read that failed is tried again, for the same accounts, no sooner than this.
export const NAMEPLATE_RETRY_MS = 10_000
// The accounts asked within this wait, a frame, go out in one call.
export const NAMEPLATE_BATCH_MS = 16
// How long a new chat message waits for its names before it shows anyway.
export const NAMEPLATE_WAIT_MS = 2_000

// An account the other players cannot name: deleted, asked deleted, or never shown.
export type Unnamed = null

type NameplatesStore = {
    // By account id, what the last read or the page's own knowledge said. An account never read is absent.
    nameplates: Readonly<Record<string, Nameplate | Unnamed>>
    // Reads the account's nameplate, with every other account asked in the same frame, unless it is fresh or being read.
    ask: (accountId: string) => void
    // Asks each account, and settles once each is read, or after NAMEPLATE_WAIT_MS.
    whenRead: (accountIds: readonly string[]) => Promise<void>
    // Where the nameplates are read from. None stops reading and forgets every nameplate.
    serve: (backend: NameplatesBackend | undefined) => void
    // Whole nameplates known as they are now: the roster's lines.
    learn: (nameplates: readonly AccountNameplate[]) => void
    // The page's own username and color, from its profile, SetName and SetColor: they show at once.
    own: (accountId: string, name: string, color: NameColor) => void
}

export const useNameplates = create<NameplatesStore>()((set, get, store) => {
    let backend: NameplatesBackend | undefined
    // A read answers only for the backend it was sent to.
    let generation = 0
    // When each account read is to be read again.
    const due = new Map<string, number>()
    // When the page last learned an account itself: a read sent before says less.
    const learned = new Map<string, number>()
    const reading = new Set<string>()
    const asked = new Set<string>()
    let timer: ReturnType<typeof setTimeout> | undefined

    const schedule = () => {
        timer ??= setTimeout(flush, NAMEPLATE_BATCH_MS)
    }

    const flush = () => {
        timer = undefined
        const from = backend
        if (!from) return
        const ids = [...asked]
        asked.clear()
        for (let i = 0; i < ids.length; i += MAX_NAMEPLATES_ASKED) read(from, ids.slice(i, i + MAX_NAMEPLATES_ASKED))
    }

    const read = (from: NameplatesBackend, ids: string[]) => {
        const sentAt = Date.now()
        const sent = generation
        ids.forEach((id) => reading.add(id))
        from.nameplates(ids).then(
            (found) => {
                if (sent !== generation) return
                keep(ids, found, sentAt)
            },
            (e) => {
                if (sent !== generation) return
                console.error("Could not read the nameplates", e)
                const retryAt = Date.now() + NAMEPLATE_RETRY_MS
                ids.forEach((id) => {
                    reading.delete(id)
                    due.set(id, retryAt)
                })
            },
        )
    }

    const keep = (ids: readonly string[], found: readonly AccountNameplate[], sentAt: number) => {
        const freshUntil = Date.now() + NAMEPLATE_FRESH_MS
        const byId = new Map(found.map((nameplate) => [nameplate.accountId, nameplate]))
        const kept: Record<string, Nameplate | Unnamed> = {}
        for (const id of ids) {
            reading.delete(id)
            if ((learned.get(id) ?? -Infinity) >= sentAt) continue
            due.set(id, freshUntil)
            const read = byId.get(id)
            kept[id] = read ? plateOf(read) : null
        }
        update(kept)
    }

    // Only what changed: a nameplate read again the same keeps its object, so nothing that shows it renders.
    const update = (next: Record<string, Nameplate | Unnamed>) => set((store) => {
        const changed = Object.entries(next).filter(([id, plate]) => !samePlate(store.nameplates[id], plate))
        return changed.length === 0 ? store : {nameplates: {...store.nameplates, ...Object.fromEntries(changed)}}
    })

    const named = (ids: readonly string[]) => ids.every((id) => id === "" || id in get().nameplates)

    return {
        nameplates: {},
        ask: (accountId) => {
            if (accountId === "" || asked.has(accountId) || reading.has(accountId)) return
            if ((due.get(accountId) ?? 0) > Date.now()) return
            asked.add(accountId)
            if (backend) schedule()
        },
        whenRead: (accountIds) => new Promise((resolve) => {
            accountIds.forEach(get().ask)
            if (named(accountIds)) {
                resolve()
                return
            }
            const settle = () => {
                clearTimeout(waiting)
                unsubscribe()
                resolve()
            }
            const waiting = setTimeout(settle, NAMEPLATE_WAIT_MS)
            const unsubscribe = store.subscribe(() => {
                if (named(accountIds)) settle()
            })
        }),
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
                set({nameplates: {}})
                return
            }
            if (asked.size > 0) schedule()
        },
        learn: (nameplates) => {
            const now = Date.now()
            const kept: Record<string, Nameplate> = {}
            for (const nameplate of nameplates) {
                if (nameplate.accountId === "") continue
                learned.set(nameplate.accountId, now)
                due.set(nameplate.accountId, now + NAMEPLATE_FRESH_MS)
                kept[nameplate.accountId] = plateOf(nameplate)
            }
            update(kept)
        },
        own: (accountId, name, color) => {
            if (accountId === "") return
            const held = get().nameplates[accountId]
            learned.set(accountId, Date.now())
            update({[accountId]: {name, color, admin: held?.admin ?? false, guest: false}})
            // A profile does not say whether the account is an admin: a read does.
            if (!held) {
                due.delete(accountId)
                get().ask(accountId)
            }
        },
    }
})

// The nameplate drawn for an account's name: undefined until it is read, null for an account nobody can name, and read
// again once NAMEPLATE_FRESH_MS old. An empty id, a message kept before accounts, is nobody.
export function useNameplate(accountId: string): Nameplate | Unnamed | undefined {
    const nameplate = useNameplates((store) => accountId === "" ? null : store.nameplates[accountId])
    // Every render asks: a fresh nameplate costs nothing, and one gone stale is read again as the name shows it.
    useEffect(() => useNameplates.getState().ask(accountId))
    return nameplate
}

// The page's own username and color, at once: its profile, and each name or color it picks.
export function useOwnNameplate(accountId: string | undefined, name: string | undefined, color: NameColor | undefined): void {
    useEffect(() => {
        if (accountId !== undefined && name !== undefined) {
            useNameplates.getState().own(accountId, name, color ?? NameColor.UNSPECIFIED)
        }
    }, [accountId, name, color])
}

function plateOf({name, color, admin, guest}: Nameplate): Nameplate {
    return {name, color, admin, guest}
}

function samePlate(a: Nameplate | Unnamed | undefined, b: Nameplate | Unnamed): boolean {
    if (a === undefined || a === null || b === null) return a === b
    return a.name === b.name && a.color === b.color && a.admin === b.admin && a.guest === b.guest
}
