const SVG_NS = "http://www.w3.org/2000/svg"

// A flag on a pole, the rally's mark: what the arrow to a ring of the page's flag carries.
export function rallyMarkSvg(className: string): SVGSVGElement {
    const svg = document.createElementNS(SVG_NS, "svg")
    svg.setAttribute("viewBox", "0 0 100 100")
    svg.setAttribute("class", className)
    svg.setAttribute("aria-hidden", "true")

    const pole = document.createElementNS(SVG_NS, "path")
    pole.setAttribute("d", "M34 84 V16")
    pole.style.setProperty("stroke", "var(--ink)")
    pole.setAttribute("stroke-width", "8")
    pole.setAttribute("stroke-linecap", "round")

    const flag = document.createElementNS(SVG_NS, "path")
    flag.setAttribute("d", "M34 18 L78 32 L34 46 Z")
    flag.style.setProperty("fill", "var(--cream)")
    flag.style.setProperty("stroke", "var(--ink)")
    flag.setAttribute("stroke-width", "6")
    flag.setAttribute("stroke-linejoin", "round")

    svg.append(pole, flag)

    return svg
}
