const SVG_NS = "http://www.w3.org/2000/svg"

export function burstPoints(
    spikes: number,
    outer: number,
    inner: number,
    wobble = 0,
    centre = 50,
): string {
    return Array.from({length: spikes * 2}, (_, i) => {
        const spike = i % 2 === 0
        const radius = spike ? outer + (i % 4 === 0 ? wobble : -wobble) : inner
        const angle = (i / (spikes * 2)) * 2 * Math.PI - Math.PI / 2
        return `${(centre + radius * Math.cos(angle)).toFixed(2)},${(centre + radius * Math.sin(angle)).toFixed(2)}`
    }).join(" ")
}

export function blastMarkSvg(className: string): SVGSVGElement {
    const svg = document.createElementNS(SVG_NS, "svg")
    svg.setAttribute("viewBox", "0 0 100 100")
    svg.setAttribute("class", className)
    svg.setAttribute("aria-hidden", "true")

    const outer = document.createElementNS(SVG_NS, "polygon")
    outer.setAttribute("points", burstPoints(10, 44, 25, 4))
    outer.style.setProperty("fill", "var(--gold)")
    outer.style.setProperty("stroke", "var(--ink)")
    outer.setAttribute("stroke-width", "7")
    outer.setAttribute("stroke-linejoin", "round")

    const core = document.createElementNS(SVG_NS, "polygon")
    core.setAttribute("points", burstPoints(8, 24, 13, 2))
    core.style.setProperty("fill", "var(--cream)")

    svg.append(outer, core)

    return svg
}
