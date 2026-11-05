export type DragRules = {
    mousePx: number
    touchPx: number
}

export type Pointer = {
    pointerId: number
    clientX: number
    clientY: number
    isPrimary: boolean
    pointerType: string
}

export type Spot = {clientX: number, clientY: number}

type Press = {pointerId: number, x: number, y: number, tolerancePx: number, lift?: Spot}

export class ClickOrDrag {
    private press: Press | undefined
    private travelled = false

    constructor(private readonly rules: DragRules) {}

    // iOS puts a quick second tap's click where the first tap was (WebKit bug 216681); the pointer events are right.
    spotOf(click: Spot): Spot | undefined {
        if (this.travelled) return undefined
        return this.press?.lift ?? click
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

    end(pointer: Pick<Pointer, "pointerId" | "clientX" | "clientY">) {
        if (!this.press || pointer.pointerId !== this.press.pointerId) return
        this.press.lift = {clientX: pointer.clientX, clientY: pointer.clientY}
    }
}
