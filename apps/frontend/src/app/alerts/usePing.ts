import {useState} from "react"
import {AlertsFailure, alertsFailureOf} from "../../backends/alerts.ts"
import {PING_NAME, pingLabelOf} from "./alertsMessages.ts"

export type Ping = {
    label: string
    name: string
    busy: boolean
    press: () => void
}

export function usePing(leave: () => Promise<void>): Ping {
    const [busy, setBusy] = useState(false)
    const [failure, setFailure] = useState<AlertsFailure>()

    const press = () => {
        if (busy) return
        setBusy(true)
        setFailure(undefined)
        leave().catch((e) => {
            setFailure(alertsFailureOf(e))
            setBusy(false)
        })
    }

    const label = pingLabelOf(failure)
    return {label, name: failure ? label : PING_NAME, busy, press}
}
