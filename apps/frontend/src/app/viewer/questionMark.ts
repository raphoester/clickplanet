/**
 * The bonus box's question mark, drawn as a shape rather than typed.
 *
 * A "?" set in a font is a different glyph on every platform — a thin serif on
 * one machine, a narrow sans on the next — and never the chunky mark a game
 * box wants. This is one stroke and one dot on a 100-unit square, so the box's
 * canvas face and the pointer's SVG badge draw exactly the same mark.
 */
export const QUESTION_MARK = {
    size: 100,
    /** The hook: over the top from the left, round, and down to the middle. */
    hook: "M 31 37 C 31 14, 69 14, 69 36 C 69 52, 50 51, 50 65",
    strokeWidth: 17,
    dot: {cx: 50, cy: 85, r: 9},
} as const

const SVG_NS = "http://www.w3.org/2000/svg"

/**
 * Draws the mark filling a `size` square at (`x`, `y`), in `ink` over a drop
 * shadow in `shadow`. The shadow is what keeps it legible over a bright face.
 */
export function drawQuestionMark(
    context: CanvasRenderingContext2D,
    x: number,
    y: number,
    size: number,
    ink: string,
    shadow: string,
): void {
    const hook = new Path2D(QUESTION_MARK.hook)
    const {cx, cy, r} = QUESTION_MARK.dot

    const pass = (colour: string, offsetX: number, offsetY: number) => {
        context.save()
        context.translate(x + offsetX, y + offsetY)
        context.scale(size / QUESTION_MARK.size, size / QUESTION_MARK.size)

        context.strokeStyle = colour
        context.lineWidth = QUESTION_MARK.strokeWidth
        context.lineCap = "round"
        context.lineJoin = "round"
        context.stroke(hook)

        context.fillStyle = colour
        context.beginPath()
        context.arc(cx, cy, r, 0, 2 * Math.PI)
        context.fill()

        context.restore()
    }

    pass(shadow, -size * 0.03, size * 0.04)
    pass(ink, 0, 0)
}

/**
 * The same mark as an inline SVG element, in `ink` over a drop shadow drawn in
 * `currentColor` — so the CSS around it decides the shadow, and can animate it.
 */
export function questionMarkSvg(className: string, ink: string): SVGSVGElement {
    const svg = document.createElementNS(SVG_NS, "svg")
    svg.setAttribute("viewBox", `0 0 ${QUESTION_MARK.size} ${QUESTION_MARK.size}`)
    svg.setAttribute("class", className)
    svg.setAttribute("aria-hidden", "true")

    const shadow = markGroup("currentColor")
    shadow.setAttribute("transform", `translate(${-QUESTION_MARK.size * 0.03} ${QUESTION_MARK.size * 0.04})`)

    svg.append(shadow, markGroup(ink))

    return svg
}

function markGroup(colour: string): SVGGElement {
    const group = document.createElementNS(SVG_NS, "g")

    const hook = document.createElementNS(SVG_NS, "path")
    hook.setAttribute("d", QUESTION_MARK.hook)
    hook.setAttribute("fill", "none")
    hook.setAttribute("stroke", colour)
    hook.setAttribute("stroke-width", String(QUESTION_MARK.strokeWidth))
    hook.setAttribute("stroke-linecap", "round")
    hook.setAttribute("stroke-linejoin", "round")

    const dot = document.createElementNS(SVG_NS, "circle")
    dot.setAttribute("cx", String(QUESTION_MARK.dot.cx))
    dot.setAttribute("cy", String(QUESTION_MARK.dot.cy))
    dot.setAttribute("r", String(QUESTION_MARK.dot.r))
    dot.setAttribute("fill", colour)

    group.append(hook, dot)

    return group
}
