import {PointerEvent, useEffect, useMemo, useRef, useState} from "react"
import {wholeOf} from "./activity.ts"
import {busiestOf, cellOf, Cells, cellsOf, colorAt, levelsOf, Picture, pictureOf, placeOf, Rgb, rgbOf} from "./heat.ts"
import {Places} from "./places.ts"

const WIDTH = 1440
const HEAT_STOPS = ["--heat-0", "--heat-1", "--heat-2", "--heat-3"]
// A faint glow is a tint of the land, and the land is all glow from here up.
const FULL = 0.5

type Props = {places: Places, changes: Uint32Array}

export function Heatmap({places, changes}: Props) {
    const canvas = useRef<HTMLCanvasElement>(null)
    const picture = useMemo(() => pictureOf(places, changes, WIDTH), [places, changes])
    const levels = useMemo(() => levelsOf(picture), [picture])
    const cells = useMemo(() => cellsOf(places, changes), [places, changes])
    const [hovered, setHovered] = useState<number | undefined>(undefined)
    const dark = useDarkScheme()

    useEffect(() => {
        if (canvas.current !== null) draw(canvas.current, picture, levels)
    }, [picture, levels, dark])

    const hover = (event: PointerEvent<HTMLCanvasElement>) => {
        const box = event.currentTarget.getBoundingClientRect()
        const longitude = ((event.clientX - box.left) / box.width) * 360 - 180
        const latitude = 90 - ((event.clientY - box.top) / box.height) * 180
        setHovered(Math.abs(latitude) <= 90 && Math.abs(longitude) <= 180 ? cellOf(latitude, longitude) : undefined)
    }

    const busiest = busiestOf(cells)

    return (
        <section className="panel" aria-label="Where tiles changed hands">
            <h2>Where tiles changed hands</h2>
            <canvas
                ref={canvas}
                className="heatmap"
                width={picture.width}
                height={picture.height}
                role="img"
                aria-label={busiest === undefined ? "No tile changed hands" : `Busiest around ${placeOf(busiest)}`}
                onPointerMove={hover}
                onPointerLeave={() => setHovered(undefined)}
            />
            <p className="readout" aria-live="polite">{readoutOf(cells, hovered ?? busiest, hovered === undefined)}</p>
            {busiest !== undefined && (
                <div className="heat-scale" aria-hidden="true">
                    <span>Fewer changes</span>
                    <span className="heat-ramp"/>
                    <span>More</span>
                </div>
            )}
        </section>
    )
}

function readoutOf(cells: Cells, cell: number | undefined, busiest: boolean): string {
    if (cell === undefined) return "No tile changed hands."
    if (cells.tiles[cell] === 0) return `${placeOf(cell)}: sea`

    const line = `${placeOf(cell)}: ${wholeOf(cells.changes[cell])} changes on ${wholeOf(cells.tiles[cell])} tiles`
    return busiest ? `Busiest: ${line}` : line
}

function draw(canvas: HTMLCanvasElement, picture: Picture, levels: Float32Array) {
    const context = canvas.getContext("2d")
    if (context === null) return

    const style = getComputedStyle(canvas)
    const sea = rgbOf(style.getPropertyValue("--map-sea"))
    const land = rgbOf(style.getPropertyValue("--map-land"))
    const stops = HEAT_STOPS.map(name => rgbOf(style.getPropertyValue(name)))

    const image = context.createImageData(picture.width, picture.height)
    for (let pixel = 0; pixel < picture.land.length; pixel++) {
        const level = levels[pixel]
        const color: Rgb = picture.land[pixel] === 0 ? sea
            : level > 0 ? mix(land, colorAt(level, stops), Math.min(1, level / FULL)) : land
        image.data.set([...color, 255], pixel * 4)
    }
    context.putImageData(image, 0, 0)
}

function mix(from: Rgb, to: Rgb, share: number): Rgb {
    return [0, 1, 2].map(c => Math.round(from[c] + (to[c] - from[c]) * share)) as Rgb
}

// The canvas reads its colors when it draws, so it draws again when the scheme changes.
function useDarkScheme(): boolean {
    const [dark, setDark] = useState(() => schemeQuery()?.matches ?? false)

    useEffect(() => {
        const query = schemeQuery()
        if (query === undefined) return
        const follow = (event: MediaQueryListEvent) => setDark(event.matches)
        query.addEventListener("change", follow)
        return () => query.removeEventListener("change", follow)
    }, [])

    return dark
}

function schemeQuery(): MediaQueryList | undefined {
    return typeof window.matchMedia === "function" ? window.matchMedia("(prefers-color-scheme: dark)") : undefined
}
