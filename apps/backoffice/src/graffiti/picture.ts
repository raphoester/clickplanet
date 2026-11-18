import {Places} from "../boards/places.ts"
import {scaleOf} from "../boards/scale.ts"

// A point of the picture: east to the right and south down, the view from -1 to 1 both ways.
export type Point = [number, number]

// One tile as the globe shows it: its cell, a hexagon (a pentagon at the lattice's twelve corners).
export type Cell = {tile: number, corners: Point[], drawn: boolean}

// A drawing and the ground around it, as seen from straight above its middle, north up.
export type Picture = {cells: Cell[], latitude: number, longitude: number}

type Vector = [number, number, number]

// How far a tile's nearest neighbour is looked for, and the step a tile with none within it is drawn at, in radians on
// earth-1 (about 12 steps, and about 2): a board reads them at its own spacing.
const NEIGHBOUR_REACH = 0.05
const LONE_STEP = 0.0075

// The map's convention, as its generator reads a vertex (boards/places.ts reads it the same way).
function vectorOf(places: Places, index: number): Vector {
    const lat = places.latitudes[index] * Math.PI / 180
    const lon = places.longitudes[index] * Math.PI / 180
    return [-Math.cos(lat) * Math.cos(lon), Math.sin(lat), Math.cos(lat) * Math.sin(lon)]
}

const dot = (a: Vector, b: Vector) => a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
const angle = (a: Vector, b: Vector) => Math.acos(Math.min(1, Math.max(-1, dot(a, b))))

// The tiles by one-degree band of latitude and longitude, built once a page load: a picture reads the bands around
// its drawing, not the whole map.
type Bands = Map<number, number[]>

const bandsOfPlaces = new WeakMap<Places, Bands>()

function bandOf(latitude: number, longitude: number): number {
    const lat = Math.min(179, Math.floor(latitude + 90))
    const lon = ((Math.floor(longitude + 180) % 360) + 360) % 360
    return lat * 360 + lon
}

function bandsOf(places: Places): Bands {
    let bands = bandsOfPlaces.get(places)
    if (bands !== undefined) return bands
    bands = new Map()
    for (let i = 0; i < places.latitudes.length; i++) {
        const band = bandOf(places.latitudes[i], places.longitudes[i])
        const tiles = bands.get(band)
        if (tiles === undefined) bands.set(band, [i])
        else tiles.push(i)
    }
    bandsOfPlaces.set(places, bands)
    return bands
}

// Every tile within `radius` (radians) of `centre`, by its index.
function tilesNear(places: Places, centre: Vector, radius: number): number[] {
    const bands = bandsOf(places)
    const latitude = Math.asin(centre[1]) * 180 / Math.PI
    const longitude = Math.atan2(centre[2], -centre[0]) * 180 / Math.PI
    const reach = radius * 180 / Math.PI + 1
    const south = Math.max(-90, latitude - reach)
    const north = Math.min(90, latitude + reach)
    const widest = Math.cos(Math.max(Math.abs(south), Math.abs(north)) * Math.PI / 180)
    const across = widest < 1e-3 ? 180 : Math.min(180, reach / widest)
    const lowest = Math.cos(radius)
    const found: number[] = []
    for (let lat = Math.floor(south); lat <= Math.min(89, Math.floor(north)); lat++) {
        const bandsAcross = across >= 180 ? 360 : Math.ceil(2 * across) + 1
        const west = across >= 180 ? -180 : Math.floor(longitude - across)
        for (let step = 0; step < bandsAcross; step++) {
            for (const index of bands.get(bandOf(lat, west + step)) ?? []) {
                if (dot(vectorOf(places, index), centre) >= lowest) found.push(index)
            }
        }
    }
    return found
}

function circumcentre(a: Point, b: Point, c: Point): Point | undefined {
    const d = 2 * (a[0] * (b[1] - c[1]) + b[0] * (c[1] - a[1]) + c[0] * (a[1] - b[1]))
    if (Math.abs(d) < 1e-18) return undefined
    const aa = a[0] * a[0] + a[1] * a[1], bb = b[0] * b[0] + b[1] * b[1], cc = c[0] * c[0] + c[1] * c[1]
    return [
        (aa * (b[1] - c[1]) + bb * (c[1] - a[1]) + cc * (a[1] - b[1])) / d,
        (aa * (c[0] - b[0]) + bb * (a[0] - c[0]) + cc * (b[0] - a[0])) / d,
    ]
}

// The drawing's tiles (wire ids, from 1) and the land around it, framed with `margin` tiles on each side. A tile id
// the map does not hold is left out. Nothing when none is left.
export function pictureOf(places: Places, tileIds: number[], margin = 4): Picture | undefined {
    const count = places.latitudes.length
    const drawn = new Set(tileIds.filter(id => Number.isInteger(id) && id >= 1 && id <= count))
    if (drawn.size === 0) return undefined
    const vectors = [...drawn].map(id => vectorOf(places, id - 1))

    const sum = vectors.reduce<Vector>((total, v) => [total[0] + v[0], total[1] + v[1], total[2] + v[2]], [0, 0, 0])
    const length = Math.hypot(...sum)
    if (length === 0) return undefined
    const centre: Vector = [sum[0] / length, sum[1] / length, sum[2] / length]
    const latitude = Math.asin(centre[1])
    const longitude = Math.atan2(centre[2], -centre[0])
    const east: Vector = [Math.sin(longitude), 0, Math.cos(longitude)]
    const north: Vector = [
        Math.sin(latitude) * Math.cos(longitude), Math.cos(latitude), -Math.sin(latitude) * Math.sin(longitude),
    ]

    // The lattice's step, from the first tile's nearest neighbour: 0.17° to 0.25° on earth-1, as the lattice stretches.
    const scale = scaleOf(places.board)
    const first = vectors[0]
    const firstId = [...drawn][0]
    let spacing = Infinity
    for (const index of tilesNear(places, first, scale.length(NEIGHBOUR_REACH))) {
        if (index !== firstId - 1) spacing = Math.min(spacing, angle(vectorOf(places, index), first))
    }
    if (!Number.isFinite(spacing)) spacing = scale.length(LONE_STEP)
    const reach = Math.max(...vectors.map(v => angle(v, centre)))
    const half = reach + margin * spacing

    // The tiles a little past the frame, so a cell on its edge has all its neighbours.
    const points = new Map<number, Point>()
    for (const index of tilesNear(places, centre, half * Math.SQRT2 + 3 * spacing)) {
        const v = vectorOf(places, index)
        points.set(index + 1, [dot(v, east), dot(v, north)])
    }

    // Buckets the size of a step and a half, to find a tile's neighbours among the few around it.
    const size = spacing * 1.6
    const keyOf = (x: number, y: number) => `${Math.floor(x / size)},${Math.floor(y / size)}`
    const buckets = new Map<string, number[]>()
    for (const [tile, [x, y]] of points) {
        const key = keyOf(x, y)
        const bucket = buckets.get(key)
        if (bucket === undefined) buckets.set(key, [tile])
        else bucket.push(tile)
    }

    // A tile's neighbours, each where it is, in turn around it. The map holds land alone, so a tile on the coast misses
    // the ones the sea would be: each missing one stands where the lattice would put it, a step away at 60° from the
    // next, so the coast's cells are whole hexagons too.
    const neighboursOf = (tile: number, [x, y]: Point): Point[] => {
        const found: {point: Point, distance: number}[] = []
        const bx = Math.floor(x / size), by = Math.floor(y / size)
        for (let dx = -1; dx <= 1; dx++) {
            for (let dy = -1; dy <= 1; dy++) {
                for (const other of buckets.get(`${bx + dx},${by + dy}`) ?? []) {
                    if (other === tile) continue
                    const point = points.get(other)!
                    found.push({point, distance: Math.hypot(point[0] - x, point[1] - y)})
                }
            }
        }
        found.sort((a, b) => a.distance - b.distance)
        const near = found.length === 0 ? [] : found.slice(0, 6).filter(each => each.distance < found[0].distance * 1.45)
        const step = near.length === 0 ? spacing : near[0].distance
        const start = near.length === 0 ? 0 : Math.atan2(near[0].point[1] - y, near[0].point[0] - x)
        const around: Point[] = []
        for (let k = 0; k < 6; k++) {
            const direction = start + k * Math.PI / 3
            const real = near.find(each => {
                const off = Math.atan2(each.point[1] - y, each.point[0] - x) - direction
                return Math.abs(Math.atan2(Math.sin(off), Math.cos(off))) < Math.PI / 7
            })
            around.push(real?.point ?? [x + step * Math.cos(direction), y + step * Math.sin(direction)])
        }
        return around
    }

    const view = ([x, y]: Point): Point => [x / half, -y / half]
    const cells: Cell[] = []
    for (const [tile, point] of points) {
        if (Math.abs(point[0]) > half * 1.1 || Math.abs(point[1]) > half * 1.1) continue
        const around = neighboursOf(tile, point)
        const corners: Point[] = []
        for (let k = 0; k < around.length; k++) {
            const corner = circumcentre(point, around[k], around[(k + 1) % around.length])
            if (corner !== undefined) corners.push(view(corner))
        }
        cells.push({tile, corners, drawn: drawn.has(tile)})
    }
    // The drawing last, so it is painted over the land.
    cells.sort((a, b) => Number(a.drawn) - Number(b.drawn))

    return {cells, latitude: latitude * 180 / Math.PI, longitude: longitude * 180 / Math.PI}
}
