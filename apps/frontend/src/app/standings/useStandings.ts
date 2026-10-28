import {useEffect, useState} from "react"
import {Boards, Standing} from "../../backends/standings.ts"

export function useStandings(backend: Boards, countryCode: string): readonly Standing[] | undefined {
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
