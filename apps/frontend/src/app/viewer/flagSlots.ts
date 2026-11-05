import * as THREE from "three"
import {SLOT_COUNT, Slots} from "./slotRanking.ts"
import type {Region} from "./atlas.ts"

export const SLOT_CELL = {width: 768, height: 512} as const

export const SLOT_COLUMNS = 2

// A flag sits this far inside its cell, its last column and row repeated out to the cell's edge for the coarse mip levels.
const MARGIN = 32

const ANISOTROPY = 4

type Canvas = {context: CanvasRenderingContext2D, texture: THREE.CanvasTexture}

// A flag whose SVG failed is not asked for again this session.
const images = new Map<string, Promise<HTMLImageElement | undefined>>()

export class FlagSlots {
    readonly texture: THREE.IUniform<THREE.Texture | null> = {value: null}
    readonly frames: THREE.IUniform<THREE.Vector4[]>
    readonly regions: THREE.IUniform<THREE.Vector4[]>

    private canvas: Canvas | undefined
    private readonly wanted: (string | undefined)[]

    constructor(private readonly onChange: () => void, count = SLOT_COUNT) {
        this.frames = {value: Array.from({length: count}, () => new THREE.Vector4())}
        this.regions = {value: Array.from({length: count}, () => new THREE.Vector4())}
        this.wanted = new Array<string | undefined>(count).fill(undefined)
    }

    show(slots: Slots, regionOf: (code: string) => Region | undefined) {
        slots.forEach((code, slot) => {
            if (code === this.wanted[slot]) return
            this.wanted[slot] = code
            this.empty(slot)
            const region = code === undefined ? undefined : regionOf(code)
            if (code === undefined || region === undefined) return
            void imageOf(code).then((image) => {
                if (this.wanted[slot] !== code || image === undefined) return
                this.draw(slot, image, region)
            })
        })
    }

    dispose() {
        this.wanted.fill(undefined)
        this.canvas?.texture.dispose()
    }

    private empty(slot: number) {
        const frame = this.frames.value[slot]
        if (frame.z === 0) return
        frame.set(0, 0, 0, 0)
        this.regions.value[slot].set(0, 0, 0, 0)
        this.onChange()
    }

    private draw(slot: number, image: HTMLImageElement, region: Region) {
        const {context, texture} = this.canvas ??= createCanvas(this.wanted.length)
        const {canvas} = context
        const aspect = region.width / region.height
        const cellX = (slot % SLOT_COLUMNS) * SLOT_CELL.width
        const cellY = Math.floor(slot / SLOT_COLUMNS) * SLOT_CELL.height
        const roomWidth = SLOT_CELL.width - 2 * MARGIN
        const roomHeight = SLOT_CELL.height - 2 * MARGIN
        const width = Math.min(roomWidth, Math.round(roomHeight * aspect))
        const height = Math.min(roomHeight, Math.round(roomWidth / aspect))
        const x = cellX + MARGIN
        const y = cellY + MARGIN

        context.clearRect(cellX, cellY, SLOT_CELL.width, SLOT_CELL.height)
        context.drawImage(image, x, y, width, height)
        context.drawImage(canvas, x, y, 1, height, cellX, y, MARGIN, height)
        context.drawImage(canvas, x + width - 1, y, 1, height, x + width, y, cellX + SLOT_CELL.width - x - width, height)
        context.drawImage(canvas, cellX, y, SLOT_CELL.width, 1, cellX, cellY, SLOT_CELL.width, MARGIN)
        context.drawImage(canvas, cellX, y + height - 1, SLOT_CELL.width, 1, cellX, y + height, SLOT_CELL.width, cellY + SLOT_CELL.height - y - height)

        this.frames.value[slot].set(x / canvas.width, 1 - (y + height) / canvas.height, width / canvas.width, height / canvas.height)
        this.regions.value[slot].set(region.x, region.y, region.width, region.height)
        texture.needsUpdate = true
        this.texture.value = texture
        this.onChange()
    }
}

function createCanvas(count: number): Canvas {
    const canvas = document.createElement("canvas")
    canvas.width = SLOT_COLUMNS * SLOT_CELL.width
    canvas.height = Math.ceil(count / SLOT_COLUMNS) * SLOT_CELL.height
    const context = canvas.getContext("2d")
    if (!context) throw new Error("failed to get a 2d context for the flag slots")
    const texture = new THREE.CanvasTexture(canvas)
    texture.anisotropy = ANISOTROPY
    return {context, texture}
}

function imageOf(code: string): Promise<HTMLImageElement | undefined> {
    let loading = images.get(code)
    if (loading === undefined) {
        loading = loadSvg(`/static/countries/svg/${code}.svg`).catch((e: unknown) => {
            console.warn(`the flag of ${code} stays on the atlas`, e)
            return undefined
        })
        images.set(code, loading)
    }
    return loading
}

// Firefox draws nothing from an SVG without a width and height of its own, and many of the flags have only a viewBox.
async function loadSvg(url: string): Promise<HTMLImageElement> {
    const response = await fetch(url)
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    const svg = new DOMParser().parseFromString(await response.text(), "image/svg+xml").documentElement
    if (!svg.hasAttribute("width") || !svg.hasAttribute("height")) {
        const [, , width, height] = (svg.getAttribute("viewBox") ?? "").split(/[\s,]+/)
        if (!width || !height) throw new Error(`${url} has no size`)
        svg.setAttribute("width", width)
        svg.setAttribute("height", height)
    }
    const blob = URL.createObjectURL(new Blob([new XMLSerializer().serializeToString(svg)], {type: "image/svg+xml"}))
    try {
        return await decoded(blob)
    } finally {
        URL.revokeObjectURL(blob)
    }
}

function decoded(src: string): Promise<HTMLImageElement> {
    return new Promise((resolve, reject) => {
        const image = new Image()
        image.onload = () => resolve(image)
        image.onerror = () => reject(new Error(`${src} could not be decoded`))
        image.src = src
    })
}
