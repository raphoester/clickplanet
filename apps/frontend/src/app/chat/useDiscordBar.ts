import {useCallback, useState} from "react"
import {DISCORD_BAR_STORAGE_KEY, isStillHidden} from "./discordBar.ts"

function readHiddenAt(): string | null {
    try {
        return window.localStorage.getItem(DISCORD_BAR_STORAGE_KEY)
    } catch {
        return null
    }
}

export function useDiscordBar() {
    const [hidden, setHidden] = useState(() => isStillHidden(readHiddenAt(), Date.now()))

    const hide = useCallback(() => {
        setHidden(true)
        try {
            window.localStorage.setItem(DISCORD_BAR_STORAGE_KEY, String(Date.now()))
        } catch (e) {
            console.error("Could not persist the Discord bar", e)
        }
    }, [])

    return {hidden, hide}
}
