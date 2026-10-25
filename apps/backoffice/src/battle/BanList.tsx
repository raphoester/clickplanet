import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {clockOf} from "../components/time.ts"
import {When} from "../components/When.tsx"
import {Conviction} from "../gen/grpc/planet/v1/admin_pb.ts"
import {wholeOf} from "../home/activity.ts"
import {useOpenPlayer} from "../navigation.ts"
import {Authors, refusalOf, unban, unbannedWords} from "../players/actions.ts"
import {Caller} from "../players/caller.ts"
import {spanOf} from "../players/durations.ts"
import {PlayerName} from "../players/PlayerName.tsx"

type Props = {planet: PlanetAdmin, convictions: Conviction[], authors: Authors, unnamed: string | undefined}

// The jury's bans of the span, the latest first.
export function BanList({planet, convictions, authors, unnamed}: Props) {
    return (
        <section className="panel" aria-label="Bans">
            <div className="panel-head">
                <h2>Bans</h2>
                <span className="muted">{wholeOf(convictions.length)} by the antibot</span>
            </div>
            {unnamed !== undefined && <p className="muted">No names: {unnamed}</p>}
            {convictions.length === 0 && <p className="muted">The antibot banned nobody.</p>}
            {convictions.length > 0 && (
                <ul className="cards">
                    {[...convictions].reverse().map(conviction => (
                        <BanCard key={keyOf(conviction)} planet={planet} conviction={conviction} authors={authors}/>
                    ))}
                </ul>
            )}
        </section>
    )
}

function keyOf(conviction: Conviction): string {
    return `${conviction.at?.toDate().getTime() ?? 0}-${conviction.scope}-${conviction.accountId}`
}

type CardProps = {planet: PlanetAdmin, conviction: Conviction, authors: Authors}

function BanCard({planet, conviction, authors}: CardProps) {
    const openPlayer = useOpenPlayer()
    const name = authors.get(conviction.accountId)?.name ?? ""
    const fired = conviction.readings.filter(reading => reading.level !== "clear")
    const caller: Caller = conviction.accountId === "" ? {scope: conviction.scope} : {accountId: conviction.accountId}
    const alongside = authors.get(conviction.alongsideAccountId)?.name ?? "another account"

    return (
        <li className="card">
            <div className="card-head">
                <PlayerName name={name} flag={conviction.topCountry}
                            unnamed={conviction.accountId === "" ? conviction.scope : "Unnamed"}/>
                <span className="muted">{conviction.at !== undefined && clockOf(conviction.at.toDate())}</span>
            </div>
            <ul className="readings">
                {fired.map(reading => (
                    <li key={reading.watchdog}>
                        <b>{reading.watchdog}</b> {reading.level}{reading.evidence !== "" && <span className="muted"> {reading.evidence}</span>}
                    </li>
                ))}
            </ul>
            <p className="card-line muted">
                <span>
                    Offence {conviction.offence}, until <When at={conviction.bannedUntil}/>
                    {conviction.alongsideAccountId !== "" && <>, with {alongside}</>}
                </span>
                <span>
                    {wholeOf(conviction.clicks)} clicks in {spanOf(conviction.activeFor)},
                    longest gap {spanOf(conviction.longestGap)}
                </span>
            </p>
            <div className="card-foot">
                {openPlayer !== undefined && (
                    <button type="button" className="button"
                            onClick={() => openPlayer({caller, name: name === "" ? undefined : name})}>
                        Open player
                    </button>
                )}
                {conviction.accountId !== "" && (
                    <UnbanButton planet={planet} label="Unban account" caller={{accountId: conviction.accountId}}
                                 who={name === "" ? "this account" : name}/>
                )}
                <UnbanButton planet={planet} label="Unban address" caller={{scope: conviction.scope}} who={conviction.scope}/>
            </div>
        </li>
    )
}

type UnbanProps = {planet: PlanetAdmin, label: string, caller: Caller, who: string}

function UnbanButton({planet, label, caller, who}: UnbanProps) {
    return (
        <ConfirmAction
            label={label}
            question={`Lift the latest ban of ${who} and forget its offence?`}
            run={() => unban(planet, caller)}
            said={unbannedWords}
            refusal={err => refusalOf("unban", err)}
        />
    )
}
