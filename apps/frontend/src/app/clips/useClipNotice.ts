import {useCallback, useEffect, useRef, useState} from "react"
import {Appearance, ClipsBackend} from "../../backends/clips.ts"
import {appearanceToShow, CLIPS_SHOWN_STORAGE_KEY, parseShownClips, readDue, shownAfter} from "../../domain/clipAppearances.ts"
import {usePageVisible} from "../usePageVisible.ts"

const NONE: readonly Appearance[] = []

function readShown(): string[] {
    try {
        return parseShownClips(window.localStorage.getItem(CLIPS_SHOWN_STORAGE_KEY))
    } catch {
        return []
    }
}

function writeShown(shown: readonly string[]) {
    try {
        window.localStorage.setItem(CLIPS_SHOWN_STORAGE_KEY, JSON.stringify(shown))
    } catch (e) {
        console.error("Could not remember the clips shown", e)
    }
}

export type ClipNotice = {
    appearance?: Appearance
    dismiss: (appearance: Appearance) => void
}

export function useClipNotice(backend: ClipsBackend | undefined, account: string | undefined): ClipNotice {
    const [read, setRead] = useState<{account: string, appearances: readonly Appearance[]}>()
    const [shown, setShown] = useState(readShown)
    const lastRead = useRef<{account: string, at: number}>()
    const visible = usePageVisible()

    useEffect(() => {
        if (account === undefined) {
            lastRead.current = undefined
            return
        }
        if (!backend || !visible) return
        const now = Date.now()
        if (lastRead.current?.account === account && !readDue(lastRead.current.at, now)) return

        lastRead.current = {account, at: now}
        backend.appearances().then(
            (appearances) => {
                if (lastRead.current?.account === account) setRead({account, appearances})
            },
            (e) => console.error("Could not read the clips you are in", e),
        )
    }, [backend, account, visible])

    const appearances = read && read.account === account ? read.appearances : NONE

    const dismiss = useCallback((appearance: Appearance) => {
        const next = shownAfter(shown, [appearance, ...appearances])
        writeShown(next)
        setShown(next)
    }, [shown, appearances])

    return {appearance: appearanceToShow(appearances, shown), dismiss}
}
