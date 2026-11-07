import {Fingers} from "../../domain/fingers.ts"

// OrbitControls follows each finger on the canvas from its pointerdown to its pointerup or pointercancel. A swipe from
// the screen's left edge that iOS takes for its swipe back, then lets go of, sends the pointerdown and never its end
// (WebKit bug 239014). The controls then follow a finger that is gone, for good: the next finger makes two, a drag
// zooms and slides the globe as a pinch would, and only a reload brought it back. The touch events still count the
// fingers right, so the fingers the canvas follows past that count are ended here with the pointercancel iOS owed,
// which every listener of the canvas takes as it would the browser's. See docs/stray-fingers.md.
export function endStrayFingers(canvas: HTMLCanvasElement, signal: AbortSignal) {
    const fingers = new Fingers()
    const listening = {signal, passive: true}

    // OrbitControls releases the capture of the last finger it lets go of. Releasing a pointer the browser has already
    // ended throws, and releasing one this canvas does not hold does nothing anyway.
    const release = canvas.releasePointerCapture.bind(canvas)
    canvas.releasePointerCapture = (pointerId: number) => {
        if (canvas.hasPointerCapture(pointerId)) release(pointerId)
    }

    const end = (stray: number[]) => {
        for (const pointerId of stray) {
            canvas.dispatchEvent(new PointerEvent("pointercancel", {pointerId, pointerType: "touch", bubbles: true}))
        }
    }

    canvas.addEventListener("pointerdown", (event) => {
        if (event.pointerType === "touch") fingers.press(event.pointerId)
    }, listening)
    for (const ending of ["pointerup", "pointercancel"] as const) {
        canvas.addEventListener(ending, (event) => fingers.lift(event.pointerId), listening)
    }

    // A finger's pointerdown comes before its touchstart, so the new finger is counted on both sides here, before it
    // has moved. Were it the other way round, the count would only lag, and end nothing that is still down.
    canvas.addEventListener("touchstart", (event) => end(fingers.strayBeyond(event.targetTouches.length)), listening)

    // Whichever of a lift's pointerup and touchend comes first, no finger on the screen leaves none to follow.
    for (const ending of ["touchend", "touchcancel"] as const) {
        window.addEventListener(ending, (event) => {
            if (event.touches.length === 0) end(fingers.strayBeyond(0))
        }, {...listening, capture: true})
    }

    // No finger stays down on a page put away.
    window.addEventListener("pagehide", () => end(fingers.strayBeyond(0)), listening)
    document.addEventListener("visibilitychange", () => {
        if (document.hidden) end(fingers.strayBeyond(0))
    }, listening)
}
