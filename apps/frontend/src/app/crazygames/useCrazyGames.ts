import {useCallback, useEffect, useRef, useState} from "react"
import {CrazyGamesPage, CrazyGamesSettings, DEFAULT_CRAZYGAMES_SETTINGS} from "../../backends/crazyGames.ts"
import {Allegiance} from "../../domain/allegiance.ts"
import {Country} from "../../domain/countries.ts"

export type CrazyGamesView = {
    chat: boolean
    // CrazyGames' mute outranks the player's sound settings.
    silenced: boolean
}

// CrazyGames refuses a second room report within 250 ms: a flag set back at load is told once, settled.
export const ROOM_SETTLE_MS = 1_000

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

export function useRoom(page: CrazyGamesPage | undefined, ready: boolean, flag: string) {
    useEffect(() => {
        if (!page || !ready) return
        const timer = setTimeout(() => page.room.report(flag), ROOM_SETTLE_MS)
        return () => clearTimeout(timer)
    }, [page, ready, flag])
}

export type Invitation = {
    // A friend's flag while the player holds another: the player chooses.
    asked?: Country
    answer: () => void
}

export function useInvites(page: CrazyGamesPage | undefined, held: Allegiance | undefined, play: (country: Country) => void): Invitation {
    const [invited, setInvited] = useState<Country>()

    useEffect(() => page?.room.listenForInvites(setInvited), [page])

    useEffect(() => {
        if (!invited) return
        if (held === undefined) play(invited)
        else if (held.countryId === invited.code) setInvited(undefined)
    }, [invited, held, play])

    const answer = useCallback(() => setInvited(undefined), [])

    return {asked: invited && held && held.countryId !== invited.code ? invited : undefined, answer}
}
