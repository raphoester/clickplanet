import {useEffect, useState} from "react"
import {Turn, TurnBackend} from "../../backends/turn.ts"
import {nextReadAt} from "../../domain/battle.ts"

export function useTurn(backend: TurnBackend | undefined): Turn | undefined {
    const [turn, setTurn] = useState<Turn>()
    const [reads, setReads] = useState(0)

    useEffect(() => {
        if (!backend) return

        let stale = false
        backend.turn().then(
            (read) => {
                if (!stale) setTurn(read)
            },
            (e) => console.error("Could not read what the schedule plays", e),
        )
        return () => {
            stale = true
        }
    }, [backend, reads])

    useEffect(() => backend?.changes(() => setReads((n) => n + 1)), [backend])

    useEffect(() => {
        if (!backend) return

        const now = Date.now()
        const timer = setTimeout(() => setReads((n) => n + 1), nextReadAt(turn, now) - now)
        return () => clearTimeout(timer)
    }, [backend, turn, reads])

    return turn
}
