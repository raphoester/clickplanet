import type {FlagPicture} from "../domain/flags.ts"

const SHAPES = [{width: 768, height: 512}, {width: 768, height: 384}, {width: 512, height: 512}] as const

// A made-up flag for the fake: three bands and a disc, in colours and a shape drawn from its name, as a blob URL.
export async function paintFakeFlag(name: string): Promise<FlagPicture> {
    const seed = [...name].reduce((hash, letter) => (hash * 31 + letter.charCodeAt(0)) >>> 0, 7)
    const {width, height} = SHAPES[seed % SHAPES.length]
    const canvas = document.createElement("canvas")
    canvas.width = width
    canvas.height = height
    const context = canvas.getContext("2d")
    if (!context) throw new Error("no 2d context to paint a fake flag")
    for (let band = 0; band < 3; band++) {
        context.fillStyle = `hsl(${(seed >> (band * 4)) % 360} 70% ${35 + band * 15}%)`
        context.fillRect(0, band * height / 3, width, height / 3)
    }
    context.fillStyle = "#ffffff"
    context.beginPath()
    context.arc(width / 2, height / 2, height / 5, 0, 2 * Math.PI)
    context.fill()
    const blob = await new Promise<Blob>((resolve, reject) => canvas.toBlob(
        (made) => made ? resolve(made) : reject(new Error("the fake flag did not encode")), "image/png"))
    return {url: URL.createObjectURL(blob), width, height, stretch: false, focusX: 0.5, focusY: 0.5}
}

// An upload as the server would draw it again: its own pixels, at its own size.
export async function pictureOfUpload(upload: Blob): Promise<FlagPicture> {
    const bitmap = await createImageBitmap(upload)
    const {width, height} = bitmap
    bitmap.close()
    return {url: URL.createObjectURL(upload), width, height, stretch: false, focusX: 0.5, focusY: 0.5}
}
