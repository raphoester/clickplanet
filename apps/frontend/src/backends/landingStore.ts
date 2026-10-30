import {Landing, LandingStore} from "../domain/landing.ts"

export const LANDING_STORAGE_KEY = "clickplanet-landing"

export function localLandingStore(): LandingStore {
    return {
        read(): Landing | undefined {
            try {
                const kept = window.localStorage.getItem(LANDING_STORAGE_KEY)
                if (!kept) return undefined

                const landing = JSON.parse(kept) as Partial<Landing>
                if (typeof landing.flag !== "string" || typeof landing.tag !== "string") return undefined

                return {flag: landing.flag, tag: landing.tag}
            } catch {
                return undefined
            }
        },
        write(landing: Landing): void {
            try {
                window.localStorage.setItem(LANDING_STORAGE_KEY, JSON.stringify(landing))
            } catch {
                // storage unavailable
            }
        },
    }
}
