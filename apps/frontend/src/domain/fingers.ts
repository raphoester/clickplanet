// The fingers pressed on the globe and not lifted yet, by pointer id, oldest first.
//
// iOS can take a touch from the page for its edge swipe and never end it there: the page gets the pointerdown, then
// neither a pointerup nor a pointercancel. The touch events still count the fingers on the screen right, so a finger
// kept past that count is one iOS took. The oldest go first: the finger iOS took was pressed before the ones that
// came after its swipe.
export class Fingers {
    private pressed: number[] = []

    press(pointerId: number) {
        if (!this.pressed.includes(pointerId)) this.pressed.push(pointerId)
    }

    lift(pointerId: number) {
        this.pressed = this.pressed.filter((pressed) => pressed !== pointerId)
    }

    // `touching` is the browser's count of the fingers on the screen. The fingers past it are lifted here and given
    // back, oldest first, for their end to be sent.
    strayBeyond(touching: number): number[] {
        const stray = this.pressed.slice(0, Math.max(0, this.pressed.length - touching))
        this.pressed = this.pressed.slice(stray.length)
        return stray
    }
}
