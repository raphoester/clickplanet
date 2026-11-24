import {PointerEvent, ReactNode, useEffect, useLayoutEffect, useRef, useState} from "react"
import {Places} from "../boards/places.ts"
import {kmWords, Place} from "../flags/pin.ts"
import {rgbOf} from "../home/heat.ts"
import {useWidth} from "../home/useWidth.ts"
import {useDarkScheme} from "../useDarkScheme.ts"
import {Box, landOf, WORLD_BOX, worldLandOf} from "./mapLand.ts"
import {
    boxOf, kmAcross, onGlobe, pannedBy, placeAt, pointOf, regionOf, resizedTo, roundedOf, View, worldOf, zoomedAt,
} from "./mapView.ts"

export type MarkKind = "flag" | "submission" | "country"

// Another capital on the map: a mark always, its flag and name when there is room.
export type Mark = {key: string, place: Place, kind: MarkKind, name: string, icon: ReactNode}

type Props = {
    places: Places
    capital: Place
    name: string
    icon: ReactNode
    marks: Mark[]
    // Unset, the capital stays where it is.
    onMove?: (capital: Place) => void
    label: string
}

// A press that moves less than this is a tap.
const TAP_PX = 6
const WHEEL_STEP = 0.0015
const BUTTON_STEP = 2
// The land past the frame drawn with it, on each side, so a drag shows land before the next drawing.
const DETAIL_MARGIN = 0.25
const DETAIL_EVERY_MS = 120
const MAX_PIXEL_RATIO = 2
// Past this zoom the whole world's picture is too coarse: the land in the frame is drawn at the frame's own scale.
const DETAIL_FROM = WORLD_BOX.scale * 1.25
// Names show on a frame narrower than this, never over one another; the capital's own always.
const NAMES_DEGREES = 90
const LABEL_HEIGHT = 18
// The zoom buttons' corner, where no name goes.
const BUTTONS_WIDTH = 210
const BUTTONS_HEIGHT = 40

type Layer = {box: Box, canvas: HTMLCanvasElement}
type Colors = {land: number, sea: number}
type Gesture =
    | {kind: "pan", start: [number, number], moved: boolean}
    | {kind: "drag", offset: [number, number]}
    | {kind: "pinch", distance: number, middle: [number, number]}

// The capital on a planisphere of the board's land, longitude across and latitude up, framed on its region at first;
// the other capitals around it. Drag to pan, the wheel or two fingers to zoom. When it may move, the capital's mark
// drags, and a tap elsewhere puts it there.
export function CapitalMap({places, capital, name, icon, marks, onMove, label}: Props) {
    const [frame, width] = useWidth<HTMLDivElement>()
    const height = Math.round(width < 600 ? width * 2 / 3 : width / 2)
    const [view, setView] = useState<View>(() => regionOf(capital, width, height))
    const canvas = useRef<HTMLCanvasElement>(null)
    const context = useRef<CanvasRenderingContext2D | null>()
    const world = useRef<Layer>()
    const detail = useRef<Layer>()
    const drawnAt = useRef(0)
    const colors = useRef<Colors>()
    const latest = useRef(view)
    const pointers = useRef(new Map<number, [number, number]>())
    const gesture = useRef<Gesture>()
    const dark = useDarkScheme()
    const ratio = Math.min(MAX_PIXEL_RATIO, window.devicePixelRatio || 1)
    latest.current = view

    useLayoutEffect(() => {
        setView(was => was.width === width && was.height === height ? was : resizedTo(was, width, height))
    }, [width, height])

    const draw = () => {
        const target = context.current
        if (target === null || target === undefined || canvas.current === null) return
        const style = getComputedStyle(canvas.current)
        paint(target, latest.current, ratio, style.getPropertyValue("--surface"), world.current, detail.current)
    }

    useEffect(() => {
        if (canvas.current === null) return
        if (context.current === undefined) context.current = canvas.current.getContext("2d")
        if (context.current === null) return
        const style = getComputedStyle(canvas.current)
        colors.current = {land: pixelOf(style.getPropertyValue("--map-land")), sea: pixelOf(style.getPropertyValue("--map-sea"))}
        world.current = layerOf(worldLandOf(places), WORLD_BOX, colors.current)
        detail.current = undefined
        draw()
        // draw reads the refs it needs: only a new board or scheme makes the layers again.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [places, dark])

    useEffect(() => {
        if (view.scale * ratio <= DETAIL_FROM) detail.current = undefined
        draw()
        if (context.current === null || context.current === undefined || colors.current === undefined) return
        if (view.scale * ratio <= DETAIL_FROM || (detail.current !== undefined && fresh(detail.current, view, ratio))) return
        const wait = Math.max(0, drawnAt.current + DETAIL_EVERY_MS - performance.now())
        const timer = setTimeout(() => {
            if (colors.current === undefined) return
            const box = boxOf(latest.current, ratio, DETAIL_MARGIN)
            detail.current = layerOf(landOf(places, box), box, colors.current)
            drawnAt.current = performance.now()
            draw()
        }, wait)
        return () => clearTimeout(timer)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [view, places, dark, ratio])

    useEffect(() => {
        const element = frame.current
        if (element === null) return
        const wheel = (event: WheelEvent) => {
            event.preventDefault()
            const [x, y] = localOf(element, event)
            const step = event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? element.clientHeight : 1
            setView(was => zoomedAt(was, Math.exp(-event.deltaY * step * WHEEL_STEP), x, y))
        }
        element.addEventListener("wheel", wheel, {passive: false})
        return () => element.removeEventListener("wheel", wheel)
    }, [frame])

    const down = (event: PointerEvent<HTMLDivElement>) => {
        if ((event.target as Element).closest(".map-buttons") !== null) return
        if (event.pointerType === "mouse" && event.button !== 0) return
        const at = localOf(event.currentTarget, event)
        capture(event)
        pointers.current.set(event.pointerId, at)
        if (pointers.current.size >= 2) {
            gesture.current = pinchOf(pointers.current)
            return
        }
        if (onMove !== undefined && (event.target as Element).closest(".mark-own") !== null) {
            const [x, y] = pointOf(latest.current, capital)
            gesture.current = {kind: "drag", offset: [at[0] - x, at[1] - y]}
            return
        }
        gesture.current = {kind: "pan", start: at, moved: false}
    }

    const move = (event: PointerEvent<HTMLDivElement>) => {
        const before = pointers.current.get(event.pointerId)
        const now = gesture.current
        if (before === undefined || now === undefined) return
        const at = localOf(event.currentTarget, event)
        pointers.current.set(event.pointerId, at)
        switch (now.kind) {
            case "pan":
                setView(was => pannedBy(was, at[0] - before[0], at[1] - before[1]))
                if (Math.hypot(at[0] - now.start[0], at[1] - now.start[1]) > TAP_PX) now.moved = true
                return
            case "drag":
                onMove?.(roundedOf(placeAt(latest.current, at[0] - now.offset[0], at[1] - now.offset[1])))
                return
            case "pinch": {
                const next = pinchOf(pointers.current)
                if (next.kind !== "pinch") return
                setView(was => zoomedAt(
                    pannedBy(was, next.middle[0] - now.middle[0], next.middle[1] - now.middle[1]),
                    next.distance / now.distance, next.middle[0], next.middle[1],
                ))
                gesture.current = next
                return
            }
        }
    }

    const up = (event: PointerEvent<HTMLDivElement>, tapped: boolean) => {
        if (!pointers.current.delete(event.pointerId)) return
        const was = gesture.current
        if (tapped && was?.kind === "pan" && !was.moved && onMove !== undefined) {
            const [x, y] = localOf(event.currentTarget, event)
            if (onGlobe(latest.current, y)) onMove(roundedOf(placeAt(latest.current, x, y)))
        }
        const [left] = pointers.current.values()
        gesture.current = left === undefined ? undefined
            : pointers.current.size >= 2 ? pinchOf(pointers.current) : {kind: "pan", start: left, moved: true}
    }

    const zoom = (factor: number) => setView(was => zoomedAt(was, factor, width / 2, height / 2))
    const km = kmAcross(view)

    return (
        <figure className="capital-figure">
            <div
                ref={frame}
                className={`capital-map${onMove === undefined ? "" : " movable"}`}
                style={{height}}
                onPointerDown={down}
                onPointerMove={move}
                onPointerUp={event => up(event, true)}
                onPointerCancel={event => up(event, false)}
            >
                <canvas
                    ref={canvas}
                    className="capital-land"
                    role="img"
                    aria-label={label}
                    width={Math.round(width * ratio)}
                    height={Math.round(height * ratio)}
                    style={{width, height}}
                />
                <Marks view={view} capital={capital} name={name} icon={icon} marks={marks} movable={onMove !== undefined}/>
                <div className="map-buttons">
                    <button type="button" className="button-small" aria-label="Zoom in" onClick={() => zoom(BUTTON_STEP)}>+</button>
                    <button type="button" className="button-small" aria-label="Zoom out" onClick={() => zoom(1 / BUTTON_STEP)}>−</button>
                    <button type="button" className="button-small" onClick={() => setView(worldOf(width, height))}>World</button>
                    <button type="button" className="button-small" onClick={() => setView(regionOf(capital, width, height))}>Capital</button>
                </div>
            </div>
            <figcaption className="hint">
                {places.board.name} · {km === undefined ? "the whole world" : `${kmWords(km)} across`}
            </figcaption>
        </figure>
    )
}

type Placed = {mark: Mark, x: number, y: number}
type Side = "right" | "left"

// Every mark in the frame; the names that fit, beside their mark or else before it: the capital's own first, then the
// flags' and the submissions', then the countries' nearest the middle.
function Marks({view, capital, name, icon, marks, movable}: {
    view: View, capital: Place, name: string, icon: ReactNode, marks: Mark[], movable: boolean,
}) {
    const [x, y] = pointOf(view, capital).map(tenthOf)
    const inFrame: Placed[] = []
    for (const mark of marks) {
        const [mx, my] = pointOf(view, mark.place)
        if (mx >= -8 && mx <= view.width + 8 && my >= -8 && my <= view.height + 8) inFrame.push({mark, x: tenthOf(mx), y: tenthOf(my)})
    }
    const ownSide: Side = labelRect(x, y, name, "right").right > view.width ? "left" : "right"
    const taken: Rect[] = [labelRect(x, y, name, ownSide), {left: view.width - BUTTONS_WIDTH, top: 0, right: view.width, bottom: BUTTONS_HEIGHT}]
    const named = new Map<string, Side>()
    if (view.width / view.scale <= NAMES_DEGREES) {
        const middle = (each: Placed) => Math.hypot(each.x - view.width / 2, each.y - view.height / 2)
        const order = [...inFrame].sort((a, b) =>
            Number(a.mark.kind === "country") - Number(b.mark.kind === "country") || middle(a) - middle(b))
        for (const each of order) {
            const side = (["right", "left"] as const).find(side => {
                const rect = labelRect(each.x, each.y, each.mark.name, side)
                return rect.left >= 0 && rect.right <= view.width && rect.top >= 0 && rect.bottom <= view.height
                    && !taken.some(other => overlap(rect, other))
            })
            if (side === undefined) continue
            taken.push(labelRect(each.x, each.y, each.mark.name, side))
            named.set(each.mark.key, side)
        }
    }

    return (
        <div className="map-marks" aria-hidden="true">
            {inFrame.map(({mark, x, y}) => (
                <span key={mark.key} className={`map-mark mark-${mark.kind}`} style={{left: x, top: y}} title={mark.name}/>
            ))}
            {inFrame.filter(each => named.has(each.mark.key)).map(({mark, x, y}) => (
                <span key={`label-${mark.key}`} className={`map-label label-${named.get(mark.key)}`} style={{left: x, top: y}}>
                    {mark.icon} {mark.name}
                </span>
            ))}
            <span className={`map-mark mark-own${movable ? " mark-movable" : ""}`} style={{left: x, top: y}} title={name}/>
            <span className={`map-label label-own label-${ownSide}`} style={{left: x, top: y}}>{icon} {name}</span>
        </div>
    )
}

const tenthOf = (pixels: number) => Math.round(pixels * 10) / 10

type Rect = {left: number, top: number, right: number, bottom: number}

// A label's room, beside its mark or before it: a flag and about 6.5 px a letter.
function labelRect(x: number, y: number, name: string, side: Side): Rect {
    const width = 40 + name.length * 6.5
    const top = y - LABEL_HEIGHT / 2
    const bottom = y + LABEL_HEIGHT / 2
    return side === "right" ? {left: x - 6, top, right: x + width, bottom} : {left: x - width, top, right: x + 6, bottom}
}

function overlap(a: Rect, b: Rect): boolean {
    return a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom
}

function pinchOf(pointers: Map<number, [number, number]>): Gesture {
    const [a, b] = pointers.values()
    if (b === undefined) return {kind: "pan", start: a, moved: true}
    return {kind: "pinch", distance: Math.max(1, Math.hypot(a[0] - b[0], a[1] - b[1])), middle: [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2]}
}

function localOf(element: Element, event: {clientX: number, clientY: number}): [number, number] {
    const box = element.getBoundingClientRect()
    return [event.clientX - box.left, event.clientY - box.top]
}

function capture(event: PointerEvent<HTMLDivElement>) {
    try {
        event.currentTarget.setPointerCapture(event.pointerId)
    } catch {
        // A pointer the browser no longer tracks: its moves still reach the frame while over it.
    }
}

// A layer drawn at its box's scale still covers the frame, neither much coarser nor much finer.
function fresh(layer: Layer, view: View, ratio: number): boolean {
    const {box} = layer
    const scale = view.scale * ratio
    if (box.scale < scale / 1.1 || box.scale > scale * 1.6) return false
    const west = view.longitude - view.width / 2 / view.scale
    const east = view.longitude + view.width / 2 / view.scale
    const north = Math.min(90, view.latitude + view.height / 2 / view.scale)
    const south = Math.max(-90, view.latitude - view.height / 2 / view.scale)
    return west >= box.west && east <= box.west + box.width / box.scale
        && north <= box.north + 1e-9 && south >= box.north - box.height / box.scale - 1e-9
}

// An opaque pixel of a token's color, as an ImageData's 32-bit word reads it on a little-endian machine.
function pixelOf(token: string): number {
    const [r, g, b] = rgbOf(token)
    return ((255 << 24) | (b << 16) | (g << 8) | r) >>> 0
}

function layerOf(land: Uint8Array, box: Box, colors: Colors): Layer | undefined {
    const canvas = document.createElement("canvas")
    canvas.width = box.width
    canvas.height = box.height
    const context = canvas.getContext("2d")
    if (context === null) return undefined
    const image = context.createImageData(box.width, box.height)
    const pixels = new Uint32Array(image.data.buffer)
    for (let i = 0; i < land.length; i++) pixels[i] = land[i] === 1 ? colors.land : colors.sea
    context.putImageData(image, 0, 0)
    return {box, canvas}
}

// The world's picture, once a turn of the globe where the frame is wider than one, then the frame's own land over it.
function paint(context: CanvasRenderingContext2D, view: View, ratio: number, off: string, world?: Layer, detail?: Layer) {
    context.setTransform(ratio, 0, 0, ratio, 0, 0)
    context.fillStyle = off
    context.fillRect(0, 0, view.width, view.height)
    const placeOf = ({box, canvas}: Layer, turn: number) => context.drawImage(
        canvas,
        view.width / 2 + (box.west + turn - view.longitude) * view.scale,
        view.height / 2 - (box.north - view.latitude) * view.scale,
        box.width / box.scale * view.scale,
        box.height / box.scale * view.scale,
    )
    if (world !== undefined) {
        const west = view.longitude - view.width / 2 / view.scale
        const first = Math.floor((west + 180) / 360) * 360
        for (let turn = first; turn - 180 < west + view.width / view.scale; turn += 360) placeOf(world, turn)
    }
    if (detail !== undefined) placeOf(detail, 0)
}
