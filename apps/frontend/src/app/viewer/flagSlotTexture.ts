import * as THREE from "three"

export const SLOT_SIZE = {width: 1536, height: 1024} as const

const ANISOTROPY = 4

type Wanted = {landmass: number, code: string}

type Canvas = {context: CanvasRenderingContext2D, texture: THREE.CanvasTexture}

// A flag whose SVG failed is not asked for again this session.
const images = new Map<string, Promise<HTMLImageElement | undefined>>()

export class FlagSlotTexture {
    readonly texture: THREE.IUniform<THREE.Texture | null> = {value: null}
    readonly frame: THREE.IUniform<THREE.Vector4> = {value: new THREE.Vector4()}
    readonly landmass: THREE.IUniform<number> = {value: 0}

    private canvas: Canvas | undefined
    private wanted: Wanted | undefined
    private drawn: string | undefined

    constructor(private readonly onChange: () => void) {}

    show(landmass: number, code: string, aspect: number) {
        if (this.wanted?.landmass === landmass && this.wanted.code === code) return
        const wanted = {landmass, code}
        this.wanted = wanted
        if (this.drawn === code) {
            this.hold(landmass)
            return
        }
        this.hold(0)
        void imageOf(code).then((image) => {
            if (this.wanted !== wanted || image === undefined) return
            this.draw(image, aspect)
            this.drawn = code
            this.hold(landmass)
        })
    }

    hide() {
        this.wanted = undefined
        this.hold(0)
    }

    dispose() {
        this.wanted = undefined
        this.canvas?.texture.dispose()
    }

    private hold(landmass: number) {
        if (this.landmass.value === landmass) return
        this.landmass.value = landmass
        this.onChange()
    }

    private draw(image: HTMLImageElement, aspect: number) {
        const {context, texture} = this.canvas ??= createCanvas()
        const width = Math.min(SLOT_SIZE.width, Math.round(SLOT_SIZE.height * aspect))
        const height = Math.min(SLOT_SIZE.height, Math.round(SLOT_SIZE.width / aspect))

        context.clearRect(0, 0, SLOT_SIZE.width, SLOT_SIZE.height)
        context.drawImage(image, 0, 0, width, height)
        // The margins repeat the flag's last column and row, so a coarse mip level at its edge finds the flag's own colours.
        if (width < SLOT_SIZE.width) context.drawImage(context.canvas, width - 1, 0, 1, height, width, 0, SLOT_SIZE.width - width, height)
        if (height < SLOT_SIZE.height) context.drawImage(context.canvas, 0, height - 1, SLOT_SIZE.width, 1, 0, height, SLOT_SIZE.width, SLOT_SIZE.height - height)

        this.frame.value.set(0, 1 - height / SLOT_SIZE.height, width / SLOT_SIZE.width, height / SLOT_SIZE.height)
        texture.needsUpdate = true
        this.texture.value = texture
    }
}

function createCanvas(): Canvas {
    const canvas = document.createElement("canvas")
    canvas.width = SLOT_SIZE.width
    canvas.height = SLOT_SIZE.height
    const context = canvas.getContext("2d")
    if (!context) throw new Error("failed to get a 2d context for the flag slot")
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
