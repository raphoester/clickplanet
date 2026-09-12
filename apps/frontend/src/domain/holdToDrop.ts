/**
 * The press that drops a bomb: held still on one spot for `holdSeconds`.
 *
 * A bomb is too precious to go on a click, and a click is also what ends every
 * drag of the globe — so a press that moves is a drag, a press let go early is
 * a change of mind, and only a press held still to the end is a drop. Mouse and
 * touch go through the same rule.
 */
export type HoldRules = {
    holdSeconds: number
    /** How far, in CSS pixels, a press may wander and still be held still. */
    tolerancePx: number
}

type Press<T> = {pointerId: number, x: number, y: number, from: number, target: T}

export class HoldToDrop<T> {
    private press: Press<T> | undefined

    constructor(private readonly rules: HoldRules) {}

    get holding(): boolean {
        return this.press !== undefined
    }

    get target(): T | undefined {
        return this.press?.target
    }

    begin(pointerId: number, x: number, y: number, at: number, target: T) {
        this.press = {pointerId, x, y, from: at, target}
    }

    move(pointerId: number, x: number, y: number) {
        if (!this.press || pointerId !== this.press.pointerId) return
        if (Math.hypot(x - this.press.x, y - this.press.y) > this.rules.tolerancePx) this.press = undefined
    }

    end(pointerId: number) {
        if (this.press && pointerId === this.press.pointerId) this.press = undefined
    }

    cancel() {
        this.press = undefined
    }

    /**
     * How far the press has got, 0 to 1, or the target once it is held to the
     * end — at which point the press is over and will not drop twice.
     */
    tick(at: number): {progress: number, drop?: T} {
        if (!this.press) return {progress: 0}

        const held = at - this.press.from
        if (held < this.rules.holdSeconds - 1e-9) return {progress: Math.max(0, held / this.rules.holdSeconds)}

        const drop = this.press.target
        this.press = undefined
        return {progress: 0, drop}
    }
}
