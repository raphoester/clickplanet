import {useEffect, useRef, useState} from "react"

// Planet grants it from auth's event, a second or two after the sign-in; a refill held after the wait is the player's.
const READS_MS = [0, 2_000, 5_000]
export const SIGN_IN_REFILL_WAIT_MS = 10_000

export function useSignInRefill(due: boolean, held: boolean, refresh: () => void, spend: () => void): void {
    const [since, setSince] = useState<number>()
    const [over, setOver] = useState(false)
    const spent = useRef(false)
    const calls = useRef({refresh, spend})
    useEffect(() => {
        calls.current = {refresh, spend}
    })

    useEffect(() => {
        if (due && since === undefined) setSince(Date.now())
    }, [due, since])

    useEffect(() => {
        if (since === undefined) return

        const left = (ms: number) => Math.max(0, since + ms - Date.now())
        const reads = READS_MS.map((ms) => setTimeout(() => calls.current.refresh(), left(ms)))
        const end = setTimeout(() => setOver(true), left(SIGN_IN_REFILL_WAIT_MS))
        return () => {
            reads.forEach(clearTimeout)
            clearTimeout(end)
        }
    }, [since])

    useEffect(() => {
        if (since === undefined || over || !held || spent.current) return
        spent.current = true
        calls.current.spend()
    }, [since, over, held])
}
