import {Leaver, LeaverStore} from "../domain/crazyGames.ts"

export const LEAVER_STORAGE_KEY = "clickplanet-crazygames-leaver"

export function localLeaverStore(): LeaverStore {
    return {
        read(): Leaver | undefined {
            try {
                const kept = window.localStorage.getItem(LEAVER_STORAGE_KEY)
                if (!kept) return undefined

                const leaver = JSON.parse(kept) as Partial<Leaver>
                if (typeof leaver.user !== "string" || typeof leaver.at !== "number") return undefined

                return {user: leaver.user, at: leaver.at}
            } catch {
                return undefined
            }
        },
        write(leaver: Leaver | undefined): void {
            try {
                if (leaver === undefined) window.localStorage.removeItem(LEAVER_STORAGE_KEY)
                else window.localStorage.setItem(LEAVER_STORAGE_KEY, JSON.stringify(leaver))
            } catch {
                // storage unavailable
            }
        },
    }
}
