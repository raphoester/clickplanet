import {useCallback} from "react"
import {Copyable} from "../components/AccountId.tsx"
import {Fact} from "../components/Fact.tsx"
import {When} from "../components/When.tsx"
import {PlanetAdmin} from "../clients.ts"
import {InspectPlayerResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {useAction} from "../useAction.ts"
import {inspect, refusalOf} from "./actions.ts"
import {Caller, isAccount} from "./caller.ts"
import {spanOf} from "./durations.ts"

const refused = (err: unknown) => refusalOf("inspect", err)

// A read: it changes nothing, so it asks for no confirmation.
export function InspectPanel({planet, caller}: {planet: PlanetAdmin, caller: Caller}) {
    const [state, start] = useAction<InspectPlayerResponse>(refused)
    const run = useCallback(() => start(() => inspect(planet, caller)), [start, planet, caller])

    return (
        <div className="action">
            <button type="button" className="button" disabled={state.step === "running"} onClick={run}>
                {state.step === "done" ? "Inspect again" : "Inspect"}
            </button>
            {state.step === "refused" && <p className="action-refused" role="alert">{state.words}</p>}
            {state.step === "done" && <Inspection value={state.value} caller={caller}/>}
        </div>
    )
}

type InspectionProps = {
    value: InspectPlayerResponse
    caller: Caller
    // Opens the address an account was read on, where its guest bans are.
    onOpenScope?: (scope: string) => void
}

export function Inspection({value, caller, onOpenScope}: InspectionProps) {
    return (
        <dl className="facts">
            {isAccount(caller) && (
                <Fact label="Address">
                    {value.scope === ""
                        ? <span className="muted">no take in the last 72 hours</span>
                        : (
                            <span className="account">
                                <Copyable text={value.scope}/>
                                {onOpenScope !== undefined && (
                                    <button type="button" className="button-small" onClick={() => onOpenScope(value.scope)}>
                                        Open
                                    </button>
                                )}
                            </span>
                        )}
                </Fact>
            )}
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
