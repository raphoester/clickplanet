import {ReactNode, useState} from "react"
import {When} from "../components/When.tsx"
import {InspectPlayerResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {inspect, PlanetAdmin, refusalOf} from "./actions.ts"
import {spanOf} from "./durations.ts"

type State =
    | {step: "idle"}
    | {step: "running"}
    | {step: "done", value: InspectPlayerResponse}
    | {step: "refused", words: string}

// A read: it changes nothing, so it asks for no confirmation.
export function InspectPanel({planet, accountId}: {planet: PlanetAdmin, accountId: string}) {
    const [state, setState] = useState<State>({step: "idle"})

    const run = () => {
        setState({step: "running"})
        inspect(planet, accountId).then(
            value => setState({step: "done", value}),
            (err: unknown) => setState({step: "refused", words: refusalOf("inspect", err)}),
        )
    }

    return (
        <div className="action">
            <button type="button" className="button" disabled={state.step === "running"} onClick={run}>
                {state.step === "done" ? "Inspect again" : "Inspect"}
            </button>
            {state.step === "refused" && <p className="action-refused" role="alert">{state.words}</p>}
            {state.step === "done" && <Inspection value={state.value}/>}
        </div>
    )
}

function Inspection({value}: {value: InspectPlayerResponse}) {
    return (
        <dl className="facts">
            <Fact label="Banned">
                {value.banned === true
                    ? <>yes, offence {value.offence}, until <When at={value.bannedUntil}/></>
                    : "no"}
            </Fact>
            <Fact label="Watched">{value.tracked === true ? `yes, ${value.flags} flags` : "no"}</Fact>
            <Fact label="Suspects">
                {`${value.suspects} of ${value.minSuspects}`}{value.guilty === true && ", guilty"}
            </Fact>
            <Fact label="Clicks">
                {`${value.clicks} in ${spanOf(value.activeFor)}, longest gap ${spanOf(value.longestGap)}`}
            </Fact>
            {value.lastClickAt !== undefined && <Fact label="Last click"><When at={value.lastClickAt}/></Fact>}
            {value.topCountry !== "" && (
                <Fact label="Plays for">{`${value.topCountry.toUpperCase()}, ${value.topCountryClicks} clicks`}</Fact>
            )}
            {value.readings.map(reading => (
                <Fact key={`${reading.watchdog}-${reading.at?.toDate().getTime() ?? 0}`} label={reading.watchdog}>
                    {reading.evidence === "" ? reading.level : `${reading.level}: ${reading.evidence}`}
                </Fact>
            ))}
        </dl>
    )
}

function Fact({label, children}: {label: string, children: ReactNode}) {
    return (
        <div>
            <dt>{label}</dt>
            <dd>{children}</dd>
        </div>
    )
}
