import "./PlacePicking.css"

export type PlacePickingProps = {
    // Under the quiz's banner, which takes the same place.
    lowered?: boolean
    onCancel: () => void
}

// Shown while the next tap on the globe picks the drawing to report.
export default function PlacePicking({lowered, onCancel}: PlacePickingProps) {
    return <div className={`place-picking${lowered ? " place-picking--lowered" : ""}`} role="status">
        <div className="place-picking-line panel">
            <span>Tap the drawing to report</span>
            <button type="button" className="button button-mini" onClick={onCancel}>Cancel</button>
        </div>
    </div>
}
