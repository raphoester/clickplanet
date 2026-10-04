export const QUESTION_MARK = {
    size: 100,
    hook: "M 31 37 C 31 14, 69 14, 69 36 C 69 52, 50 51, 50 65",
    strokeWidth: 17,
    dot: {cx: 50, cy: 85, r: 9},
} as const

const SVG_NS = "http://www.w3.org/2000/svg"

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
    hook.style.setProperty("stroke", colour)
    hook.setAttribute("stroke-width", String(QUESTION_MARK.strokeWidth))
    hook.setAttribute("stroke-linecap", "round")
    hook.setAttribute("stroke-linejoin", "round")

    const dot = document.createElementNS(SVG_NS, "circle")
    dot.setAttribute("cx", String(QUESTION_MARK.dot.cx))
    dot.setAttribute("cy", String(QUESTION_MARK.dot.cy))
    dot.setAttribute("r", String(QUESTION_MARK.dot.r))
    dot.style.setProperty("fill", colour)

    group.append(hook, dot)

    return group
}
