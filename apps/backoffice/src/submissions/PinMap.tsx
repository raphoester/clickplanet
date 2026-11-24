import {useMemo} from "react"
import {Places} from "../boards/places.ts"
import {kmWords, Place} from "../flags/pin.ts"
import {groundOf} from "../graffiti/picture.ts"

// From the pin to each side of the map.
export const MAP_HALF_KM = 250

type Props = {
    places: Places
    pin: Place
    // The other pins to mark, each with what a pointer over it says.
    others: {place: Place, name: string}[]
    label: string
}

// The land around a pin as the globe shows it from straight above, north up, the sea left blank, the pin marked and
// the other flags' pins in the frame beside it.
export function PinMap({places, pin, others, label}: Props) {
    const ground = useMemo(
        () => groundOf(places, pin, MAP_HALF_KM, others.map(other => other.place)),
        [places, pin, others],
    )

    return (
        <figure className="pin-figure">
            <svg className="pin-map" viewBox="-1 -1 2 2" role="img" aria-label={label}>
                {ground.cells.map(cell => (
                    <polygon
                        key={cell.tile}
                        className="graffiti-land"
                        points={cell.corners.map(([x, y]) => `${x.toFixed(4)},${y.toFixed(4)}`).join(" ")}
                    />
                ))}
                {ground.marks.map((at, i) => at !== undefined && Math.abs(at[0]) < 1 && Math.abs(at[1]) < 1 && (
                    <circle key={i} className="pin-other" cx={at[0]} cy={at[1]} r={0.025}>
                        <title>{others[i].name}</title>
                    </circle>
                ))}
                <circle className="pin-ring" cx={0} cy={0} r={0.06}/>
                <circle className="pin-mark" cx={0} cy={0} r={0.018}/>
            </svg>
            <figcaption className="hint">
                {places.board.name}, {kmWords(2 * MAP_HALF_KM)} across, north up.
                {ground.cells.length === 0 && " No land in the frame."}
            </figcaption>
        </figure>
    )
}
