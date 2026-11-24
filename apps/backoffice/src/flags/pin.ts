// A point on the globe, in degrees, as flags.v1.Pin holds it.
export type Place = {latitude: number, longitude: number}

const EARTH_KM = 6371
const GAME = "https://clickplanet.lol/play"
// The camera's zoom of a link to the globe: a region and its tiles, on any board.
const GLOBE_ZOOM = 20

// A degree as typed: "48.137" or "48,137". Undefined for anything else.
export function degreesOf(typed: string): number | undefined {
    const text = typed.trim().replace(",", ".")
    if (text === "") return undefined
    const value = Number(text)
    return Number.isFinite(value) ? value : undefined
}

// A latitude and a longitude pasted together, as a map copies them: "48.137, 11.575".
export function pastedPlaceOf(typed: string): Place | undefined {
    const both = typed.trim().match(/^(-?\d+(?:\.\d+)?)\s*[,;\s]\s*(-?\d+(?:\.\d+)?)$/)
    if (both === null) return undefined
    const place = {latitude: Number(both[1]), longitude: Number(both[2])}
    return pinProblems(place).length === 0 ? place : undefined
}

// What the server would refuse of a pin.
export function pinProblems({latitude, longitude}: Place): string[] {
    const problems: string[] = []
    if (!(Math.abs(latitude) <= 90)) problems.push("A latitude from −90 to 90.")
    if (!(Math.abs(longitude) <= 180)) problems.push("A longitude from −180 to 180.")
    return problems
}

// "48.1374, 11.5755": four decimals are about ten metres.
export function placeWords({latitude, longitude}: Place): string {
    return `${latitude.toFixed(4)}, ${longitude.toFixed(4)}`
}

// The game's globe, turned to the place.
export function globeLinkOf({latitude, longitude}: Place): string {
    return `${GAME}?lat=${latitude.toFixed(4)}&lon=${longitude.toFixed(4)}&zoom=${GLOBE_ZOOM}`
}

// The great-circle distance, in km.
export function distanceKm(a: Place, b: Place): number {
    const radians = (degrees: number) => degrees * Math.PI / 180
    const dLat = radians(b.latitude - a.latitude)
    const dLon = radians(b.longitude - a.longitude)
    const h = Math.sin(dLat / 2) ** 2 + Math.cos(radians(a.latitude)) * Math.cos(radians(b.latitude)) * Math.sin(dLon / 2) ** 2
    return 2 * EARTH_KM * Math.asin(Math.min(1, Math.sqrt(h)))
}

export function kmWords(km: number): string {
    return km < 10 ? `${km.toFixed(1)} km` : `${Math.round(km).toLocaleString("en")} km`
}
