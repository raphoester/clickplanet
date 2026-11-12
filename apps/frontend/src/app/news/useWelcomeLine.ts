import {useEffect} from "react"
import {ShownNamesBackend} from "../../backends/clips.ts"
import {LandingStore} from "../../domain/landing.ts"
import {welcomeLineOf} from "../../domain/video/welcome.ts"
import {useNews} from "./newsStore.ts"

const WELCOMED_KEY = "clickplanet-recruit-welcomed"

function welcomed(): boolean {
    try {
        return window.localStorage.getItem(WELCOMED_KEY) !== null
    } catch {
        return true
    }
}

function keepWelcomed() {
    try {
        window.localStorage.setItem(WELCOMED_KEY, "1")
    } catch {
        // storage unavailable: welcomed() said so already
    }
}

// A newcomer who came by a player's address is told who called it, once, when the map is up.
export function useWelcomeLine(names: ShownNamesBackend, landings: LandingStore, ready: boolean, flag: string) {
    const tellWelcome = useNews((news) => news.tellWelcome)

    useEffect(() => {
        if (!ready || welcomed()) return
        const recruiter = landings.read()?.recruiterAccountId
        if (recruiter === undefined) return
        let live = true
        names.shownNames([recruiter]).then(
            (shown) => {
                if (!live) return
                keepWelcomed()
                const line = welcomeLineOf(recruiter, shown, flag)
                if (line !== undefined) tellWelcome(line)
            },
            (e) => console.error("could not read the name of who called you", e),
        )
        return () => {
            live = false
        }
    }, [names, landings, ready, flag, tellWelcome])
}
