import {useMemo} from "react"
import {Places} from "../home/places.ts"
import {pictureOf} from "./picture.ts"

type Props = {
    places: Places
    tileIds: number[]
    // What a screen reader says of it.
    label: string
    // The list's: the drawing alone, filling the square, a tile of margin.
    small?: boolean
}

const MARGIN_TILES = 4

// A drawing as the globe shows it from straight above its middle, north up: its tiles over the land around it, the
// sea left blank. The squad's tiles are the ones it last saw, not the map now.
export function GraffitiPicture({places, tileIds, label, small = false}: Props) {
    const picture = useMemo(() => pictureOf(places, tileIds, small ? 1 : MARGIN_TILES), [places, tileIds, small])
    if (picture === undefined) return <p className="muted">None of its tiles is on this map.</p>
    const cells = small ? picture.cells.filter(cell => cell.drawn) : picture.cells

    return (
        <svg
            className={small ? "graffiti-thumb" : "graffiti-picture"}
            viewBox="-1 -1 2 2"
            role="img"
            aria-label={label}
            preserveAspectRatio="xMidYMid slice"
        >
            {cells.map(cell => (
                <polygon
                    key={cell.tile}
                    className={cell.drawn ? "graffiti-drawn" : "graffiti-land"}
                    points={cell.corners.map(([x, y]) => `${x.toFixed(4)},${y.toFixed(4)}`).join(" ")}
                    data-drawn={cell.drawn ? "" : undefined}
                />
            ))}
        </svg>
    )
}
