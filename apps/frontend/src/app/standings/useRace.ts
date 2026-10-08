import {useEffect, useState} from "react"
import {Race, StandingsBackend} from "../../backends/standings.ts"

export function useRace(backend: StandingsBackend): Race | undefined {
    const [race, setRace] = useState<Race>()

    useEffect(() => {
        let stale = false
        const stop = backend.listenForRace((read) => {
            if (!stale) setRace(read)
        })
        return () => {
            stale = true
            stop()
        }
    }, [backend])

    return race
}
