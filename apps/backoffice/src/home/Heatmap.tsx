import {PointerEvent, useEffect, useMemo, useRef, useState} from "react"
import {wholeOf} from "./activity.ts"
import {colorAt, Grid, Heat, heatOf, placeOf, Rgb, rgbOf} from "./heat.ts"

const HEAT_STOPS = ["--heat-0", "--heat-1", "--heat-2", "--heat-3"]
// The calmest change is still drawn above the bare land.
const FLOOR = 0.12

export function Heatmap({grid}: {grid: Grid}) {
    const canvas = useRef<HTMLCanvasElement>(null)
    const heat = useMemo(() => heatOf(grid), [grid])
    const [hovered, setHovered] = useState<number | undefined>(undefined)
    const dark = useDarkScheme()

    useEffect(() => {
        if (canvas.current !== null) draw(canvas.current, grid, heat)
    }, [grid, heat, dark])

    const hover = (event: PointerEvent<HTMLCanvasElement>) => {
        const box = event.currentTarget.getBoundingClientRect()
        const column = Math.floor(((event.clientX - box.left) / box.width) * grid.columns)
        const row = Math.floor(((event.clientY - box.top) / box.height) * grid.rows)
        const inside = column >= 0 && column < grid.columns && row >= 0 && row < grid.rows
        setHovered(inside ? row * grid.columns + column : undefined)
    }

    return (
        <section className="panel" aria-label="Where tiles changed hands">
            <h2>Where tiles changed hands</h2>
            <canvas
                ref={canvas}
                className="heatmap"
                width={grid.columns}
                height={grid.rows}
                role="img"
                aria-label={heat.hottest === undefined ? "No tile changed hands" : `Busiest around ${placeOf(heat.hottest, grid.columns)}`}
                onPointerMove={hover}
                onPointerLeave={() => setHovered(undefined)}
            />
            <p className="readout" aria-live="polite">{readoutOf(grid, heat, hovered)}</p>
            {heat.busiest > 0 && (
                <div className="heat-scale">
                    <span>{rateOf(heat.calmest)}</span>
                    <span className="heat-ramp" aria-hidden="true"/>
                    <span>{rateOf(heat.busiest)} changes a tile</span>
                </div>
            )}
        </section>
    )
}

function readoutOf(grid: Grid, heat: Heat, hovered: number | undefined): string {
    const cell = hovered ?? heat.hottest
    if (cell === undefined) return "No tile changed hands."

    const land = grid.land[cell] ?? 0
    if (land === 0) return `${placeOf(cell, grid.columns)}: sea`

    const line = `${placeOf(cell, grid.columns)}: ${wholeOf(grid.changes[cell] ?? 0)} changes on ${wholeOf(land)} tiles`
    return hovered === undefined ? `Busiest: ${line}` : line
}

function rateOf(rate: number): string {
    return rate >= 10 ? wholeOf(Math.round(rate)) : rate.toFixed(rate >= 1 ? 1 : 2)
}

function draw(canvas: HTMLCanvasElement, grid: Grid, heat: Heat) {
    const context = canvas.getContext("2d")
    if (context === null) return

    const style = getComputedStyle(canvas)
    const sea = rgbOf(style.getPropertyValue("--map-sea"))
    const land = rgbOf(style.getPropertyValue("--map-land"))
    const stops = HEAT_STOPS.map(name => rgbOf(style.getPropertyValue(name)))

    const image = context.createImageData(grid.columns, grid.rows)
    for (let cell = 0; cell < grid.columns * grid.rows; cell++) {
        const level = heat.levels[cell]
        const color: Rgb = (grid.changes[cell] ?? 0) > 0
            ? colorAt(FLOOR + (1 - FLOOR) * level, stops)
            : (grid.land[cell] ?? 0) > 0 ? land : sea
        image.data.set([...color, 255], cell * 4)
    }
    context.putImageData(image, 0, 0)
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
