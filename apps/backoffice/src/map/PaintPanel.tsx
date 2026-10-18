import {useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {CountryField} from "../components/CountryField.tsx"
import {countryIdOf, countryLabel} from "../countries.ts"
import {wordsOf} from "../refusal.ts"
import {countOf} from "../support/labels.ts"
import {useAction} from "../useAction.ts"
import {paint, Painted, Painting} from "./actions.ts"

type Counted = {painting: Painting, counts: Painted}

// `count` tiles get a flag, in patches that start on a country's ground and may grow past its border.
export function PaintPanel({planet}: {planet: PlanetAdmin}) {
    const [flag, setFlag] = useState("")
    const [area, setArea] = useState("")
    const [count, setCount] = useState(500)
    const [proximity, setProximity] = useState(0.8)
    const [dry, start] = useAction<Counted>(wordsOf)

    const painting: Painting = {flag: countryIdOf(flag), area: countryIdOf(area), count, proximity}
    const ready = painting.flag !== "" && count > 0
    // The question quotes the dry run, so it holds only for the painting it drew.
    const counted = dry.step === "done" && samePainting(dry.value.painting, painting) ? dry.value.counts : undefined
    const where = painting.area === "" ? "anywhere on the map" : `from ${countryLabel(painting.area)}'s ground`

    return (
        <section className="panel" aria-label="Paint random tiles">
            <h2>Paint random tiles with a flag</h2>
            <p className="hint">
                Nothing undoes it. A tile somebody takes while it runs stays theirs. Copy the table first if you may
                want to return.
            </p>
            <div className="fields">
                <CountryField label="Flag" value={flag} onChange={setFlag}/>
                <CountryField label="Starting on" value={area} onChange={setArea} empty="Empty: anywhere on the map"/>
                <label className="field">
                    <span className="muted">Tiles</span>
                    <input
                        type="number"
                        inputMode="numeric"
                        min={1}
                        value={count}
                        onChange={event => setCount(Math.max(0, Math.floor(Number(event.target.value)) || 0))}
                    />
                </label>
            </div>
            <label className="field">
                <span className="muted">Proximity {proximity.toFixed(2)}: {proximityWords(proximity)}</span>
                <input
                    type="range"
                    min={0}
                    max={1}
                    step={0.05}
                    value={proximity}
                    onChange={event => setProximity(Number(event.target.value))}
                />
            </label>
            <div className="action">
                <button
                    type="button"
                    className="button"
                    disabled={!ready || dry.step === "running"}
                    onClick={() => start(() => paint(planet, painting, true).then(counts => ({painting, counts})))}
                >
                    Dry run
                </button>
                {dry.step === "refused" && <p className="action-refused" role="alert">{dry.words}</p>}
                {counted !== undefined && (
                    <p className="action-said" role="status">
                        {countOf(counted.eligible, "tile")} of the area do not wear the flag yet. The draw picked{" "}
                        {counted.picked}, {counted.outsideArea} of them past its border.
                    </p>
                )}
            </div>
            <ConfirmAction
                label="Paint"
                tone="danger"
                disabled={counted === undefined || counted.picked === 0}
                question={`Paint about ${countOf(counted?.picked ?? 0, "tile")} with ${countryLabel(painting.flag)}, ${where}?`}
                run={() => paint(planet, painting, false)}
                said={done => `Painted ${countOf(done.painted, "tile")} of ${done.picked} picked, `
                    + `${done.outsideArea} past the border.`}
                refusal={wordsOf}
            />
        </section>
    )
}

function samePainting(a: Painting, b: Painting): boolean {
    return a.flag === b.flag && a.area === b.area && a.count === b.count && a.proximity === b.proximity
}

function proximityWords(proximity: number): string {
    if (proximity === 0) return "scattered over the area"
    if (proximity === 1) return "one patch"
    return "a few patches"
}
