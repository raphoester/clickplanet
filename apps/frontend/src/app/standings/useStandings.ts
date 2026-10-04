import {useEffect, useState} from "react"
import {Standing, StandingsBackend} from "../../backends/standings.ts"

export function useStandings(backend: StandingsBackend, countryCode: string): readonly Standing[] | undefined {
    const [read, setRead] = useState<{countryCode: string, standings: Standing[]}>()

    useEffect(() => {
        let stale = false
        const stop = backend.listenForStandings(countryCode, (standings) => {
            if (!stale) setRead({countryCode, standings})
        })
        return () => {
            stale = true
            stop()
        }
    }, [backend, countryCode])

    return read?.countryCode === countryCode ? read.standings : undefined
}
