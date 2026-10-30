import {useEffect, useRef, useState} from "react"
import {CrazyGamesPage, CrazyGamesSettings, DEFAULT_CRAZYGAMES_SETTINGS} from "../../backends/crazyGames.ts"

export type CrazyGamesView = {
    chat: boolean
    // CrazyGames' mute outranks the player's sound settings.
    silenced: boolean
}

export function useCrazyGames(page: CrazyGamesPage | undefined): CrazyGamesView {
    const [settings, setSettings] = useState<CrazyGamesSettings>(() => page?.settings() ?? DEFAULT_CRAZYGAMES_SETTINGS)

    useEffect(() => page?.listenForSettings(setSettings), [page])

    return {chat: settings.chat, silenced: !settings.audio}
}

// The globe on screen is the gameplay: CrazyGames measures the load up to it.
export function useGameplay(page: CrazyGamesPage | undefined, ready: boolean) {
    const told = useRef(false)

    useEffect(() => {
        if (!page || !ready || told.current) return
        told.current = true
        page.started()
    }, [page, ready])
}
