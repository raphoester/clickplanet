import {useState} from "react"
import {AuthFailure, failureOf} from "../../backends/account.ts"

export type Ping = {
    busy: boolean
    failure?: AuthFailure
    press: () => void
}

// The press leaves for Discord: busy until the page goes, or the failure that kept it.
export function usePing(connect: () => Promise<void>): Ping {
    const [busy, setBusy] = useState(false)
    const [failure, setFailure] = useState<AuthFailure>()

    const press = () => {
        if (busy) return
        setBusy(true)
        setFailure(undefined)
        connect().catch((e) => {
            setFailure(failureOf(e))
            setBusy(false)
        })
    }

    return {busy, failure, press}
}
