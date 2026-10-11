import {useCallback, useState} from "react"

export const DISCORD_BAR_STORAGE_KEY = "clickplanet-discord-bar"

function readHidden(): boolean {
    try {
        return window.localStorage.getItem(DISCORD_BAR_STORAGE_KEY) === "hidden"
    } catch {
        return false
    }
}

export function useDiscordBar() {
    const [hidden, setHidden] = useState(readHidden)

    const hide = useCallback(() => {
        setHidden(true)
        try {
            window.localStorage.setItem(DISCORD_BAR_STORAGE_KEY, "hidden")
        } catch (e) {
            console.error("Could not persist the Discord bar", e)
        }
    }, [])

    return {hidden, hide}
}
