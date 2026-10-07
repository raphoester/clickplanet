export type VirtualClock = {
    advance(milliseconds: number): void
}

// A frame is drawn when the recorder asks, and every effect is timed in video seconds however slow the capture.
export function installVirtualClock(): VirtualClock {
    let now = 0
    let nextId = 1
    let waiting = new Map<number, FrameRequestCallback>()

    performance.now = () => now
    window.requestAnimationFrame = (callback) => {
        const id = nextId++
        waiting.set(id, callback)
        return id
    }
    window.cancelAnimationFrame = (id) => {
        waiting.delete(id)
    }

    return {
        advance(milliseconds) {
            now += milliseconds
            const due = waiting
            waiting = new Map()
            for (const callback of due.values()) callback(now)
        },
    }
}
