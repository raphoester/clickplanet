import {useEffect} from "react"
import {WarningsListener} from "../../backends/antibot.ts"
import {useHeldIdentity} from "../useHeldIdentity.ts"
import {useWarning} from "./warningStore.ts"

export function useWarningFeed(backend: WarningsListener): void {
    const token = useHeldIdentity(backend)

    useEffect(() => {
        if (!token) return
        const stop = backend.listenForWarnings(
            () => useWarning.getState().warn(),
            () => useWarning.getState().notice(),
        )
        return () => {
            stop()
            useWarning.getState().dismiss()
            useWarning.getState().dismissNotice()
        }
    }, [backend, token])
}
