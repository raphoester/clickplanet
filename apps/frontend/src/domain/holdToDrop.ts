export type HoldRules = {
    holdSeconds: number
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

    tick(at: number): {progress: number, drop?: T} {
        if (!this.press) return {progress: 0}

        const held = at - this.press.from
        if (held < this.rules.holdSeconds - 1e-9) return {progress: Math.max(0, held / this.rules.holdSeconds)}

        const drop = this.press.target
        this.press = undefined
        return {progress: 0, drop}
    }
}
