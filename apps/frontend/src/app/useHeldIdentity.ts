import {useEffect, useState} from "react"

export const TOKEN_CHECK_MS = 1_000

// A stream's server reads the token as it opens: a stream keyed on this opens again under a new one.
export function useHeldIdentity(source: {heldIdentity(): string | undefined}): string | undefined {
    const [token, setToken] = useState(() => source.heldIdentity())

    useEffect(() => {
        const check = () => setToken(source.heldIdentity())
        check()
        const timer = window.setInterval(check, TOKEN_CHECK_MS)
        return () => window.clearInterval(timer)
    }, [source])

    return token
}
