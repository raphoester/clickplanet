export type AcceptedClick = {
    country: string
    took: boolean
}

export type ListenForClicks = (onAccepted: (click: AcceptedClick) => void) => () => void

export type AcceptedClicks = {
    record: (click: AcceptedClick) => void
    listenForClicks: ListenForClicks
}

export function acceptedClicks(): AcceptedClicks {
    const listeners = new Set<(click: AcceptedClick) => void>()
    return {
        record: (click) => listeners.forEach((listener) => listener(click)),
        listenForClicks: (onAccepted) => {
            listeners.add(onAccepted)
            return () => listeners.delete(onAccepted)
        },
    }
}
