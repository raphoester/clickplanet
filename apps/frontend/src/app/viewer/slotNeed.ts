export type Vec3 = {x: number, y: number, z: number}

export type ScreenView = {
    toward: Vec3
    right: Vec3
    up: Vec3
    origin: {x: number, y: number}
    pixelsPerRadian: number
    width: number
    height: number
}

const SAMPLES = 32

export class LandmassSight {
    private readonly starts: Uint32Array
    private readonly points: Float32Array
    private readonly weights: Float32Array

    constructor(tilesOf: (landmass: number) => ArrayLike<number>, count: number, positions: Float32Array, private readonly tileArea: number) {
        this.starts = new Uint32Array(count + 1)
        const picked: number[][] = []
        for (let landmass = 0; landmass < count; landmass++) {
            const tiles = tilesOf(landmass)
            const step = Math.max(1, tiles.length / SAMPLES)
            const chosen: number[] = []
            for (let at = 0; at < tiles.length; at += step) chosen.push(tiles[Math.floor(at)])
            picked.push(chosen)
            this.starts[landmass + 1] = this.starts[landmass] + chosen.length
        }
        this.points = new Float32Array(this.starts[count] * 3)
        this.weights = new Float32Array(count)
        picked.forEach((chosen, landmass) => {
            this.weights[landmass] = chosen.length === 0 ? 0 : tilesOf(landmass).length / chosen.length
            chosen.forEach((tile, i) => {
                const at = (tile - 1) * 3
                const length = Math.hypot(positions[at], positions[at + 1], positions[at + 2])
                const to = (this.starts[landmass] + i) * 3
                this.points[to] = positions[at] / length
                this.points[to + 1] = positions[at + 1] / length
                this.points[to + 2] = positions[at + 2] / length
            })
        })
    }

    areasOnScreen(view: ScreenView): Float32Array {
        const areas = new Float32Array(this.weights.length)
        for (let landmass = 0; landmass < areas.length; landmass++) areas[landmass] = this.areaOnScreen(landmass, view)
        return areas
    }

    private areaOnScreen(landmass: number, view: ScreenView): number {
        const scale = view.pixelsPerRadian
        let seen = 0
        const {toward, right, up} = view
        for (let i = this.starts[landmass]; i < this.starts[landmass + 1]; i++) {
            const px = this.points[i * 3]
            const py = this.points[i * 3 + 1]
            const pz = this.points[i * 3 + 2]
            const towards = px * toward.x + py * toward.y + pz * toward.z
            if (towards <= 0) continue
            const x = view.origin.x + (px * right.x + py * right.y + pz * right.z) * scale
            const y = view.origin.y + (px * up.x + py * up.y + pz * up.z) * scale
            if (Math.abs(x) > view.width / 2 || Math.abs(y) > view.height / 2) continue
            seen += towards
        }
        return seen * this.weights[landmass] * this.tileArea * scale * scale
    }
}

// The lattice has 10 n² + 2 points over the sphere.
export function tileAreaOf(cols: number): number {
    return 4 * Math.PI / (10 * cols * cols + 2)
}

export function flagNeedOf(centre: Vec3, halfU: number, halfV: number, aspect: number, view: ScreenView): number {
    const polar = Math.abs(centre.y) > 0.9999
    const east = polar ? {x: 1, y: 0, z: 0} : normalised({x: centre.z, y: 0, z: -centre.x})
    const north = {
        x: centre.y * east.z - centre.z * east.y,
        y: centre.z * east.x - centre.x * east.z,
        z: centre.x * east.y - centre.y * east.x,
    }
    const across = Math.sqrt(Math.max(0, 1 - dot(east, view.toward) ** 2))
    const upward = Math.sqrt(Math.max(0, 1 - dot(north, view.toward) ** 2))
    return Math.max(2 * halfU * across, 2 * halfV * aspect * upward) * view.pixelsPerRadian
}

function normalised({x, y, z}: Vec3): Vec3 {
    const length = Math.hypot(x, y, z)
    return {x: x / length, y: y / length, z: z / length}
}

function dot(a: Vec3, b: Vec3): number {
    return a.x * b.x + a.y * b.y + a.z * b.z
}
