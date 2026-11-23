import * as THREE from "three"
import {aspectOf, FLAG_NUMBERS, flagNumberOf, pictureOf} from "./atlas.ts"
import {type Cell, flagIn, overlap, type SlotBudget, textureSizeOf} from "./slotCells.ts"
import {type FlagNeed, type Placed, SlotLayout} from "./slotRanking.ts"

const ANISOTROPY = 4

const FRAME = 4

type Drawn = {placed: Placed, flag: Cell, bitmap: ImageBitmap}

const images = new Map<string, Promise<HTMLImageElement | undefined>>()
// A flag whose SVG failed is not asked for again this session.
const failed = new Set<string>()

export class FlagSlots {
    readonly texture: THREE.IUniform<THREE.Texture | null> = {value: null}
    readonly frames: THREE.IUniform<THREE.DataTexture>

    private readonly frameData = new Float32Array(FLAG_NUMBERS * FRAME)
    private layout: SlotLayout | undefined
    private target: THREE.DataTexture | undefined
    private wanted: Placed[] = []
    private readonly shown = new Map<string, Cell>()
    private canvas: HTMLCanvasElement | undefined
    private readonly source = new THREE.Texture()
    private drawn: Drawn | undefined
    private busy = false

    constructor(private readonly onChange: () => void) {
        const frames = new THREE.DataTexture(this.frameData, FLAG_NUMBERS, 1, THREE.RGBAFormat, THREE.FloatType)
        frames.needsUpdate = true
        this.frames = {value: frames}
    }

    arrange(needs: ReadonlyMap<string, FlagNeed>, budget: SlotBudget) {
        if (this.layout && !sameBudget(this.layout.budget, budget)) this.reset()
        this.layout ??= new SlotLayout(budget)
        this.wanted = this.layout.arrange(new Map([...needs].filter(([code]) => !failed.has(code) && aspectOf(code) !== undefined)))
        this.prepare()
    }

    upload(renderer: THREE.WebGLRenderer) {
        const drawn = this.drawn
        if (drawn === undefined) return
        this.drawn = undefined
        this.busy = false
        if (this.layout && this.wants(drawn.placed)) {
            const {code, cell} = drawn.placed
            const target = this.target ??= createTarget(textureSizeOf(this.layout.budget))
            for (const [other, shown] of this.shown) {
                if (other !== code && overlap(shown, cell)) this.hide(other)
            }
            this.source.image = drawn.bitmap
            renderer.copyTextureToTexture(this.source, target, null, new THREE.Vector2(cell.x, cell.y))
            this.show(code, cell, drawn.flag, target)
        }
        drawn.bitmap.close()
        this.prepare()
    }

    dispose() {
        this.wanted = []
        this.drawn?.bitmap.close()
        this.target?.dispose()
        this.frames.value.dispose()
    }

    private show(code: string, cell: Cell, flag: Cell, target: THREE.DataTexture) {
        const {width, height} = target.image
        this.frameData.set([flag.x / width, flag.y / height, flag.width / width, flag.height / height], frameAt(code))
        this.frames.value.needsUpdate = true
        this.shown.set(code, cell)
        this.texture.value = target
        this.onChange()
    }

    private hide(code: string) {
        this.frameData.fill(0, frameAt(code), frameAt(code) + FRAME)
        this.frames.value.needsUpdate = true
        this.shown.delete(code)
        this.onChange()
    }

    private reset() {
        for (const code of [...this.shown.keys()]) this.hide(code)
        this.target?.dispose()
        this.target = undefined
        this.texture.value = null
        this.layout = undefined
    }

    private prepare() {
        if (this.busy) return
        const next = this.wanted.find(({code, cell}) => !sameCell(this.shown.get(code), cell) && !failed.has(code))
        if (next === undefined) {
            this.canvas = undefined
            return
        }
        this.busy = true
        void this.rasterise(next).then((drawn) => {
            if (drawn !== undefined) {
                this.drawn = drawn
                return
            }
            this.busy = false
            this.prepare()
        })
    }

    private wants({code, cell}: Placed): boolean {
        return this.wanted.some((placed) => placed.code === code && sameCell(placed.cell, cell))
    }

    private async rasterise(placed: Placed): Promise<Drawn | undefined> {
        const image = await imageOf(placed.code)
        const aspect = aspectOf(placed.code)
        if (image === undefined || aspect === undefined) {
            this.layout?.forget(placed.code)
            this.wanted = this.wanted.filter((other) => other !== placed)
            return undefined
        }
        await idle()
        if (!this.wants(placed)) return undefined
        this.canvas = drawFlag(this.canvas, image, placed.cell, aspect)
        const bitmap = await createImageBitmap(this.canvas)
        if (!this.wants(placed)) {
            bitmap.close()
            return undefined
        }
        const flag = flagIn({...placed.cell, x: 0, y: 0}, aspect)
        const {cell} = placed
        return {placed, bitmap, flag: {...flag, x: cell.x + flag.x, y: cell.y + cell.height - flag.y - flag.height}}
    }
}

// The texture's rows run up from the cell's bottom: the flag is drawn upside down.
function drawFlag(reused: HTMLCanvasElement | undefined, image: HTMLImageElement, cell: Cell, aspect: number): HTMLCanvasElement {
    const canvas = reused ?? document.createElement("canvas")
    if (canvas.width !== cell.width || canvas.height !== cell.height) {
        canvas.width = cell.width
        canvas.height = cell.height
    }
    const context = canvas.getContext("2d")
    if (!context) throw new Error("failed to get a 2d context for the flag slots")
    const {width, height} = cell
    const flag = flagIn({...cell, x: 0, y: 0}, aspect)
    const x = flag.x
    const y = height - flag.y - flag.height

    context.clearRect(0, 0, width, height)
    context.save()
    context.translate(0, height)
    context.scale(1, -1)
    context.drawImage(image, flag.x, flag.y, flag.width, flag.height)
    context.restore()
    context.drawImage(canvas, x, y, 1, flag.height, 0, y, x, flag.height)
    context.drawImage(canvas, x + flag.width - 1, y, 1, flag.height, x + flag.width, y, width - x - flag.width, flag.height)
    context.drawImage(canvas, 0, y, width, 1, 0, 0, width, y)
    context.drawImage(canvas, 0, y + flag.height - 1, width, 1, 0, y + flag.height, width, height - y - flag.height)
    // Rasterises it now, between two frames, not in the frame that sends it.
    context.getImageData(0, 0, 1, 1)
    return canvas
}

function createTarget({width, height}: {width: number, height: number}): THREE.DataTexture {
    const texture = new THREE.DataTexture(null, width, height)
    // Storage alone: each cell is sent on its own.
    texture.source.dataReady = false
    texture.minFilter = THREE.LinearMipmapLinearFilter
    texture.magFilter = THREE.LinearFilter
    texture.generateMipmaps = true
    texture.anisotropy = ANISOTROPY
    texture.needsUpdate = true
    return texture
}

function frameAt(code: string): number {
    return flagNumberOf(code) * FRAME
}

function sameCell(a: Cell | undefined, b: Cell | undefined): boolean {
    return a !== undefined && b !== undefined && a.x === b.x && a.y === b.y && a.width === b.width && a.height === b.height
}

function sameBudget(a: SlotBudget, b: SlotBudget): boolean {
    return a.roots === b.roots && a.largest === b.largest
}

function idle(): Promise<void> {
    return new Promise((resolve) => {
        if (typeof requestIdleCallback === "function") requestIdleCallback(() => resolve(), {timeout: 100})
        else setTimeout(resolve, 0)
    })
}

// A built flag from its SVG; a sold one from its picture, 768 x 512 at most.
function imageOf(code: string): Promise<HTMLImageElement | undefined> {
    let loading = images.get(code)
    if (loading === undefined) {
        const picture = pictureOf(code)
        loading = (picture ? decoded(picture.url, "anonymous") : loadSvg(`/static/countries/svg/${code}.svg`)).catch((e: unknown) => {
            console.warn(`the flag of ${code} stays on the atlas`, e)
            failed.add(code)
            return undefined
        })
        images.set(code, loading)
    }
    return loading
}

async function loadSvg(url: string): Promise<HTMLImageElement> {
    const response = await fetch(url)
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    const blob = URL.createObjectURL(new Blob([sizedSvg(await response.text(), url)], {type: "image/svg+xml"}))
    try {
        return await decoded(blob)
    } finally {
        URL.revokeObjectURL(blob)
    }
}

// Firefox draws nothing from an SVG without a width and height of its own, and many of the flags have only a viewBox.
export function sizedSvg(svg: string, url: string): string {
    const root = /<svg\b[^>]*>/.exec(svg)?.[0]
    if (root === undefined) throw new Error(`${url} has no svg element`)
    if (/\swidth\s*=/.test(root) && /\sheight\s*=/.test(root)) return svg
    const [, , width, height] = (/\sviewBox\s*=\s*["']([^"']*)["']/.exec(root)?.[1] ?? "").trim().split(/[\s,]+/)
    if (!width || !height) throw new Error(`${url} has no size`)
    const bare = root.replace(/\s(width|height)\s*=\s*("[^"]*"|'[^']*')/g, "")
    return svg.replace(root, () => `<svg width="${width}" height="${height}"${bare.slice(4)}`)
}

// A picture from another origin is asked for with CORS, or the canvas it is drawn on could not be sent to the GPU.
function decoded(src: string, crossOrigin?: "anonymous"): Promise<HTMLImageElement> {
    return new Promise((resolve, reject) => {
        const image = new Image()
        if (crossOrigin) image.crossOrigin = crossOrigin
        image.onload = () => resolve(image)
        image.onerror = () => reject(new Error(`${src} could not be decoded`))
        image.src = src
    })
}
