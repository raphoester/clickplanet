import {useEffect, useState} from "react"
import {ShownNamesBackend} from "../../backends/clips.ts"

// A name is judged within seconds of being set: a name read as not shown yet is asked once more after this.
const ASK_AGAIN_MS = 30_000

// The player's own name as the clips let a video show it, which the address carries; none until it is read.
export function useShownName(names: ShownNamesBackend, accountId: string | undefined, username: string | undefined): string | undefined {
    const [read, setRead] = useState<{accountId: string, username: string, shown: string | undefined}>()

    useEffect(() => {
        if (accountId === undefined || username === undefined) return
        let live = true
        let again: number | undefined
        const ask = (retry: boolean) => names.shownNames([accountId]).then(
            (shown) => {
                if (!live) return
                const name = shown.get(accountId)
                setRead({accountId, username, shown: name})
                if (name !== username && retry) again = window.setTimeout(() => void ask(false), ASK_AGAIN_MS)
            },
            (e) => console.error("could not read which name a video may show", e),
        )
        void ask(true)
        return () => {
            live = false
            window.clearTimeout(again)
        }
    }, [names, accountId, username])

    return read !== undefined && read.accountId === accountId && read.username === username ? read.shown : undefined
}
