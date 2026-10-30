import {useCallback, useEffect, useRef, useState} from "react"
import {CrazyGamesPage, CrazyGamesSettings, DEFAULT_CRAZYGAMES_SETTINGS} from "../../backends/crazyGames.ts"

export type CrazyGamesView = {
    chat: boolean
    // CrazyGames' mute, or an ad playing: CrazyGames' rule outranks the player's sound settings.
    silenced: boolean
    onAdPlaying: (playing: boolean) => void
}

export function useCrazyGames(page: CrazyGamesPage | undefined): CrazyGamesView {
    const [settings, setSettings] = useState<CrazyGamesSettings>(() => page?.settings() ?? DEFAULT_CRAZYGAMES_SETTINGS)
    const [adPlaying, setAdPlaying] = useState(false)

    useEffect(() => page?.listenForSettings(setSettings), [page])

    const onAdPlaying = useCallback((playing: boolean) => {
        setAdPlaying(playing)
        page?.playing(!playing)
    }, [page])

    return {chat: settings.chat, silenced: !settings.audio || adPlaying, onAdPlaying}
}

// The globe on screen is the gameplay: CrazyGames measures the load up to it.
export function useGameplay(page: CrazyGamesPage | undefined, ready: boolean) {
    const told = useRef(false)

    useEffect(() => {
        if (!page || !ready || told.current) return
        told.current = true
        page.loaded()
        page.playing(true)
    }, [page, ready])
}
