import {useEffect, useRef, useState} from "react"
import {PlayerTitle, TitlesListener} from "../../backends/progression.ts"

export const TOKEN_CHECK_MS = 1_000

// The server reads the token as the stream opens: a new token opens it again.
export function useTitlesEarned(
    backend: TitlesListener | undefined,
    signedIn: boolean,
    onEarned: (title: PlayerTitle) => void,
): void {
    const [token, setToken] = useState(() => backend?.heldIdentity())
    const earned = useRef(onEarned)
    earned.current = onEarned

    useEffect(() => {
        if (!backend) return
        const check = () => setToken(backend.heldIdentity())
        check()
        const timer = window.setInterval(check, TOKEN_CHECK_MS)
        return () => window.clearInterval(timer)
    }, [backend])

    useEffect(() => {
        if (!backend || !signedIn || !token) return
        return backend.listenForTitles((title) => earned.current(title))
    }, [backend, signedIn, token])
}
