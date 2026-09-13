const SVG_NS = "http://www.w3.org/2000/svg"

/**
 * The points of a jagged star, alternating `outer` and `inner` radii, as an SVG
 * `points` list. `wobble` pushes every other spike out a little further, so the
 * burst looks thrown rather than stamped.
 */
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

/**
 * The explosion on the blast pointer, drawn rather than an emoji: an emoji is a
 * different picture on every platform and never matched the badge it sat in.
 *
 * An orange burst outlined in the badge's dark red, with a pale core, on the
 * same 100-unit square as the box's question mark.
 */
export function blastMarkSvg(className: string): SVGSVGElement {
    const svg = document.createElementNS(SVG_NS, "svg")
    svg.setAttribute("viewBox", "0 0 100 100")
    svg.setAttribute("class", className)
    svg.setAttribute("aria-hidden", "true")

    const outer = document.createElementNS(SVG_NS, "polygon")
    outer.setAttribute("points", burstPoints(10, 44, 25, 4))
    outer.setAttribute("fill", "#FFA928")
    outer.setAttribute("stroke", "#4A0A02")
    outer.setAttribute("stroke-width", "7")
    outer.setAttribute("stroke-linejoin", "round")

    const core = document.createElementNS(SVG_NS, "polygon")
    core.setAttribute("points", burstPoints(8, 24, 13, 2))
    core.setAttribute("fill", "#FFF1B8")

    svg.append(outer, core)

    return svg
}
