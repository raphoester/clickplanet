import * as THREE from "three"
import type {FlagPicture} from "../../domain/flags.ts"
import {FIRST_SOLD, flagNumberOf, knowPicture, Region, showPicture, SPRITE_WIDTH} from "./atlas.ts"
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts"

// A sold flag's cell: a built sprite's width, its picture's height at that width (half to all of it), row by row under
// the built atlas.
const CELL_HEIGHT = 100
const PER_ROW = Math.floor(ATLAS_SIZE.width / SPRITE_WIDTH)
// The built atlas's last row repeated under it: a sample past a bottom flag's edge finds its colours, as at the edge.
const GUTTER = 16
// The sold area grows by this much, so the built atlas keeps its texels' place in every mip level up to it.
const GROWTH = 256
// Some players' phones take no texture higher.
export const MOST_HEIGHT = 4096

export function soldCellOf(number: number, picture: Pick<FlagPicture, "width" | "height">): Region {
    const index = number - FIRST_SOLD
    const height = Math.min(CELL_HEIGHT, Math.max(1, Math.round(SPRITE_WIDTH * picture.height / picture.width)))
    return {
        x: (index % PER_ROW) * SPRITE_WIDTH,
        y: ATLAS_SIZE.height + GUTTER + Math.floor(index / PER_ROW) * CELL_HEIGHT,
        width: SPRITE_WIDTH,
        height,
    }
}

// The texture's height that holds every row down to `bottom`.
export function heightFor(bottom: number): number {
    return ATLAS_SIZE.height + Math.ceil(Math.max(0, bottom - ATLAS_SIZE.height) / GROWTH) * GROWTH
}

type Decoded = {code: string, number: number, cell: Region, bitmap: ImageBitmap}

// The globe's flag texture: the built atlas as it loads, then, once a sold flag's picture is decoded, a taller copy
// with each sold flag in its cell under it. A picture is decoded off the frame and sent whole, one a frame; the built
// atlas is sent again only when the texture grows.
export class FlagAtlas {
    readonly texture: THREE.IUniform<THREE.Texture>
    readonly size: THREE.IUniform<THREE.Vector2>

    private readonly built: THREE.Texture
    private grown: THREE.DataTexture | undefined
    private gutter: Promise<ImageBitmap> | undefined
    private gutterBitmap: ImageBitmap | undefined
    private readonly wanted = new Map<string, string>()
    private readonly decoded: Decoded[] = []
    private readonly sent = new Map<number, Decoded>()
    private readonly source = new THREE.Texture()
    private disposed = false

    constructor(
        private readonly onChange: () => void,
        private readonly onShown: (code: string, number: number) => void,
        loader: THREE.TextureLoader = new THREE.TextureLoader(),
    ) {
        this.built = loader.load(ATLAS_URL, () => onChange())
        this.texture = {value: this.built}
        this.size = {value: new THREE.Vector2(ATLAS_SIZE.width, ATLAS_SIZE.height)}
    }

    // Every sold flag's picture, the closed ones' too: a closed flag still holds land. Each is drawn once.
    want(pictures: ReadonlyMap<string, FlagPicture>) {
        for (const [code, picture] of pictures) {
            knowPicture(code, picture)
            if (this.wanted.get(code) === picture.url) continue
            this.wanted.set(code, picture.url)
            const number = flagNumberOf(code)
            if (number < FIRST_SOLD) continue
            void this.decode(code, number, picture)
        }
    }

    upload(renderer: THREE.WebGLRenderer) {
        const next = this.decoded[0]
        const image = this.built.image as HTMLImageElement | undefined
        if (next === undefined || !image?.complete || this.gutterBitmap === undefined) return
        this.decoded.shift()
        const height = heightFor(next.cell.y + CELL_HEIGHT)
        if (height > MOST_HEIGHT) {
            next.bitmap.close()
            return
        }
        if (this.grown === undefined || this.grown.image.height < height) this.grow(renderer, height, image, this.gutterBitmap)
        this.send(renderer, next.bitmap, next.cell, true)
        this.sent.get(next.number)?.bitmap.close()
        this.sent.set(next.number, next)
        showPicture(next.number, next.cell)
        this.onShown(next.code, next.number)
        if (this.decoded.length > 0) this.onChange()
    }

    dispose() {
        this.disposed = true
        for (const {number, bitmap} of this.sent.values()) {
            showPicture(number, undefined)
            bitmap.close()
        }
        this.decoded.forEach(({bitmap}) => bitmap.close())
        this.gutterBitmap?.close()
        this.built.dispose()
        this.grown?.dispose()
    }

    private async decode(code: string, number: number, picture: FlagPicture) {
        const cell = soldCellOf(number, picture)
        try {
            const [bitmap] = await Promise.all([pictureBitmapOf(picture.url, cell), this.gutterOf()])
            if (this.disposed || this.wanted.get(code) !== picture.url) {
                bitmap.close()
                return
            }
            this.decoded.push({code, number, cell, bitmap})
            this.onChange()
        } catch (e) {
            console.warn(`the picture of the flag ${code} did not load: it stays blank`, e)
        }
    }

    private gutterOf(): Promise<ImageBitmap> {
        this.gutter ??= gutterBitmapOf().then((bitmap) => {
            this.gutterBitmap = bitmap
            return bitmap
        })
        return this.gutter
    }

    // The built atlas and every picture sent are sent again into a taller texture; the last send builds the mip levels.
    private grow(renderer: THREE.WebGLRenderer, height: number, image: HTMLImageElement, gutter: ImageBitmap) {
        const target = new THREE.DataTexture(null, ATLAS_SIZE.width, height)
        // Storage alone: each part is sent on its own.
        target.source.dataReady = false
        target.minFilter = THREE.LinearMipmapLinearFilter
        target.magFilter = THREE.LinearFilter
        target.needsUpdate = true
        this.grown = target
        // The built atlas is sent as TextureLoader sent it: flipped by WebGL, its pixels as they are in the file.
        target.flipY = true
        this.send(renderer, image, {x: 0, y: 0, width: ATLAS_SIZE.width, height: ATLAS_SIZE.height}, false)
        target.flipY = false
        this.send(renderer, gutter, {x: 0, y: ATLAS_SIZE.height, width: ATLAS_SIZE.width, height: GUTTER}, false)
        for (const {bitmap, cell} of this.sent.values()) this.send(renderer, bitmap, cell, false)

        // The built atlas's own texture goes too: its image stays, for the next growth.
        this.texture.value.dispose()
        this.texture.value = target
        this.size.value.set(ATLAS_SIZE.width, height)
    }

    // `at` is from the picture's top, as the regions are; the texture's rows run up from its bottom.
    private send(renderer: THREE.WebGLRenderer, image: TexImageSource, at: Region, mipmaps: boolean) {
        const target = this.grown
        if (target === undefined) return
        target.generateMipmaps = mipmaps
        this.source.image = image
        renderer.copyTextureToTexture(this.source, target, null, new THREE.Vector2(at.x, target.image.height - at.y - at.height))
    }
}

// The picture fitted to its cell and turned upside down, as the texture's rows run, then made a bitmap off the frame.
async function pictureBitmapOf(url: string, cell: Region): Promise<ImageBitmap> {
    const response = await fetch(url, {mode: "cors"})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    const picture = await createImageBitmap(await response.blob())
    try {
        return await flippedBitmapOf(picture, 0, 0, picture.width, picture.height, cell.width, cell.height)
    } finally {
        picture.close()
    }
}

// The built atlas's last row, as many times as the gutter is high.
async function gutterBitmapOf(): Promise<ImageBitmap> {
    const response = await fetch(ATLAS_URL)
    if (!response.ok) throw new Error(`${ATLAS_URL} answered ${response.status}`)
    const atlas = await createImageBitmap(await response.blob())
    try {
        return await flippedBitmapOf(atlas, 0, ATLAS_SIZE.height - 1, ATLAS_SIZE.width, 1, ATLAS_SIZE.width, GUTTER)
    } finally {
        atlas.close()
    }
}

async function flippedBitmapOf(
    image: ImageBitmap, sx: number, sy: number, sw: number, sh: number, width: number, height: number,
): Promise<ImageBitmap> {
    const canvas = document.createElement("canvas")
    canvas.width = width
    canvas.height = height
    const context = canvas.getContext("2d")
    if (!context) throw new Error("no 2d context to fit a flag's picture")
    context.imageSmoothingQuality = "high"
    context.translate(0, height)
    context.scale(1, -1)
    context.drawImage(image, sx, sy, sw, sh, 0, 0, width, height)
    // Rasterises it now, off the frame that sends it.
    context.getImageData(0, 0, 1, 1)
    return createImageBitmap(canvas)
}
