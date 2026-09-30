/**
 * Whether a press on the globe was a click or a drag of it.
 *
 * The browser sends `click` for every press let go where it began, however far
 * it went in between: turning the globe and letting go over the planet claimed
 * the tile under the cursor. So a press that wanders past its tolerance is a
 * drag, and so is one a second finger joins, which is a pinch. Mouse and touch
 * go through the same rule; a fingertip only gets more room, since it rolls as
 * it lifts.
 */
export type DragRules = {
    /** How far, in CSS pixels, a mouse or a pen may wander and still click. */
    mousePx: number
    /** The same for a finger. */
    touchPx: number
}

/** The parts of a `PointerEvent` the rule reads. */
export type Pointer = {
    pointerId: number
    clientX: number
    clientY: number
    isPrimary: boolean
    pointerType: string
}

type Press = {pointerId: number, x: number, y: number, tolerancePx: number}

export class ClickOrDrag {
    private press: Press | undefined
    private travelled = false

    constructor(private readonly rules: DragRules) {}

    /** Whether the last press turned the globe rather than clicked on it. */
    get dragged(): boolean {
        return this.travelled
    }

    begin(pointer: Pointer) {
        if (!pointer.isPrimary) {
            this.travelled = true
            return
        }
        this.press = {
            pointerId: pointer.pointerId,
            x: pointer.clientX,
            y: pointer.clientY,
            tolerancePx: pointer.pointerType === "touch" ? this.rules.touchPx : this.rules.mousePx,
        }
        this.travelled = false
    }

    move(pointer: Pick<Pointer, "pointerId" | "clientX" | "clientY">) {
        if (!this.press || pointer.pointerId !== this.press.pointerId) return
        const distance = Math.hypot(pointer.clientX - this.press.x, pointer.clientY - this.press.y)
        if (distance > this.press.tolerancePx) this.travelled = true
    }
}
