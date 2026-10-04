import {useEffect, useState} from "react"
import {Standing, StandingsBackend} from "../../backends/standings.ts"

export const STANDINGS_EVERY_MS = 15_000

export function useStandings(backend: StandingsBackend, countryCode: string): readonly Standing[] | undefined {
    const [read, setRead] = useState<{countryCode: string, standings: Standing[]}>()

    useEffect(() => {
        let stale = false
        const readStandings = () => backend.standings(countryCode).then(
            (standings) => {
                if (!stale) setRead({countryCode, standings})
            },
            (e) => console.error("Could not read the standings", e),
        )

        void readStandings()
        const timer = setInterval(readStandings, STANDINGS_EVERY_MS)
        return () => {
            stale = true
            clearInterval(timer)
        }
    }, [backend, countryCode])

    return read?.countryCode === countryCode ? read.standings : undefined
}
