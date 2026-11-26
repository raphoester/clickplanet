import {useEffect, useState} from "react"
import type {FlagPicture, Pin} from "../../domain/flags.ts"
import {cropAround} from "../../domain/flagShot.ts"
import type {FlagShot} from "../viewer/globe.ts"

// The map at a flag's capital with the flag on it, its size and the capital's offset from its middle, in CSS pixels: it
// is shown at the size the globe drew it, so a tile on it is a tile's size in the game.
export type MapShot = {url: string, width: number, height: number, capital: {x: number, y: number}}

// A picture the player picked, or the public one once approved.
export type ShotPicture = File | FlagPicture

export type FlagShooter = (pin: Pin, picture: ShotPicture) => Promise<MapShot>

// In CSS pixels: as wide as the widest card that shows it.
export const SHOT_SIZE = {width: 560, height: 280}

const pictures = new WeakMap<File, Promise<FlagPicture>>()

// Its address lives with the page: the globe's atlas draws it again at each shot.
function pictureOfFile(file: File): Promise<FlagPicture> {
    let picture = pictures.get(file)
    if (picture === undefined) {
        picture = createImageBitmap(file).then((bitmap) => {
            const {width, height} = bitmap
            bitmap.close()
            return {url: URL.createObjectURL(file), width, height, stretch: false, focusX: 0.5, focusY: 0.5}
        })
        pictures.set(file, picture)
    }
    return picture
}

async function urlOf({frame, at}: FlagShot): Promise<MapShot> {
    const ratio = frame.width / window.innerWidth
    const crop = cropAround(frame, at, {width: SHOT_SIZE.width * ratio, height: SHOT_SIZE.height * ratio})
    const canvas = document.createElement("canvas")
    canvas.width = crop.width
    canvas.height = crop.height
    const context = canvas.getContext("2d")
    if (!context) throw new Error("no 2D canvas to crop the map's shot in")
    context.putImageData(new ImageData(frame.pixels, frame.width, frame.height), -crop.x, -crop.y, crop.x, crop.y, crop.width, crop.height)
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, "image/png"))
    if (!blob) throw new Error("the map's shot did not encode")
    return {
        url: URL.createObjectURL(blob),
        width: crop.width / ratio,
        height: crop.height / ratio,
        capital: {x: (at.x - crop.x - crop.width / 2) / ratio, y: (at.y - crop.y - crop.height / 2) / ratio},
    }
}

export function flagShooter(previewFlag: (pin: Pin, picture: FlagPicture) => Promise<FlagShot>): FlagShooter {
    return async (pin, picture) => {
        const shown = picture instanceof File ? await pictureOfFile(picture) : picture
        return urlOf(await previewFlag(pin, shown))
    }
}

// Each picture's shots by pin, taken once a visit: the paywall shows the one the form took.
const shots = new WeakMap<object, Map<string, MapShot>>()
const publicPictures = new Map<string, object>()

function keyOf(picture: ShotPicture): object {
    if (picture instanceof File) return picture
    let key = publicPictures.get(picture.url)
    if (key === undefined) {
        key = {}
        publicPictures.set(picture.url, key)
    }
    return key
}

const pinKeyOf = ({latitude, longitude}: Pin) => `${latitude},${longitude}`

function keptShot(picture: ShotPicture, pin: Pin): MapShot | undefined {
    return shots.get(keyOf(picture))?.get(pinKeyOf(pin))
}

function keepShot(picture: ShotPicture, pin: Pin, shot: MapShot) {
    const key = keyOf(picture)
    const byPin = shots.get(key) ?? new Map<string, MapShot>()
    byPin.set(pinKeyOf(pin), shot)
    shots.set(key, byPin)
}

function cancelled(e: unknown): boolean {
    return e instanceof DOMException && e.name === "AbortError"
}

// None until the globe took it, or with no picture or no capital yet.
export function useMapShot(shooter: FlagShooter, pin: Pin | undefined, picture: ShotPicture | undefined): MapShot | undefined {
    const [, setTaken] = useState(0)

    useEffect(() => {
        if (pin === undefined || picture === undefined || keptShot(picture, pin)) return
        let current = true
        shooter(pin, picture).then(
            (shot) => {
                keepShot(picture, pin, shot)
                if (current) setTaken((taken) => taken + 1)
            },
            (e: unknown) => {
                if (!cancelled(e)) console.warn("could not show the flag on the map", e)
            },
        )
        return () => {
            current = false
        }
    }, [shooter, pin, picture])

    return pin === undefined || picture === undefined ? undefined : keptShot(picture, pin)
}
