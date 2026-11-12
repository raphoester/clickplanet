import {useCallback, useState} from "react"
import {Teased} from "../../domain/battleTeaser.ts"
import {Ask, askClosed, askDue, Asked, askedOf, askOpened, AskScene, NOTHING_ASKED, NUDGES_STORAGE_KEY} from "../../domain/nudges.ts"

function readAsked(): Asked {
    try {
        return askedOf(window.localStorage.getItem(NUDGES_STORAGE_KEY))
    } catch {
        return NOTHING_ASKED
    }
}

function writeAsked(asked: Asked) {
    try {
        window.localStorage.setItem(NUDGES_STORAGE_KEY, JSON.stringify(asked))
    } catch (e) {
        console.error("Could not remember the asks", e)
    }
}

export type AskState = {
    ask?: Ask
    close: (ask: Ask) => void
    open: (ask: Ask) => void
}

export function useAsk(scene: AskScene, teased: Teased | null, now: number): AskState {
    const [asked, setAsked] = useState(readAsked)

    const keep = useCallback((next: (asked: Asked) => Asked) => {
        const changed = next(readAsked())
        writeAsked(changed)
        setAsked(changed)
    }, [])
    const close = useCallback((ask: Ask) => keep((kept) => askClosed(kept, ask, Date.now())), [keep])
    const open = useCallback((ask: Ask) => keep((kept) => askOpened(kept, ask, Date.now())), [keep])

    return {ask: askDue(asked, teased, scene, now), close, open}
}
