import {Code, ConnectError} from "@connectrpc/connect"
import {useCallback} from "react"
import {Clients} from "../clients.ts"
import {Copyable} from "../components/AccountId.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {InspectPlayerResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Opened} from "../navigation.ts"
import {useCall} from "../useCall.ts"
import {ANTIBOT_OFF, inspect} from "./actions.ts"
import {Caller, idOf, isAccount, whoOf} from "./caller.ts"
import {GrantPanel} from "./GrantPanel.tsx"
import {Inspection} from "./InspectPanel.tsx"
import {BanAction, MuteAction, UnbanAction} from "./Moderation.tsx"
import {BanLine, PlayerStats} from "./PlayerStats.tsx"
import {RevertPanel} from "./RevertPanel.tsx"

type Props = {
    clients: Clients
    opened: Opened
    onBack: () => void
    onOpen: (opened: Opened) => void
}

type Read = {on: true, value: InspectPlayerResponse} | {on: false}

// One player, by account or by address: what the antibot reads of it, and every tool that takes it.
export function PlayerView({clients, opened, onBack, onOpen}: Props) {
    const {caller, name = "", row} = opened
    const who = whoOf(caller, name)
    const other = otherOf(opened)

    const read = useCallback(
        (signal: AbortSignal): Promise<Read> => inspect(clients.planet, caller, signal).then(
            value => ({on: true, value}),
            (err: unknown) => {
                if (ConnectError.from(err).code === Code.FailedPrecondition) return {on: false}
                throw err
            },
        ),
        [clients.planet, caller],
    )
    const [inspection, again] = useCall(read)

    return (
        <section className="detail" aria-label="Player">
            <button type="button" className="button-small" onClick={onBack}>← Players</button>

            <article className="card">
                <div className="card-head">
                    {row !== undefined ? <BanLine player={row}/> : <span/>}
                    {row?.lastAt !== undefined && <span className="muted">last take <When at={row.lastAt}/></span>}
                </div>
                <dl className="people">
                    {name !== "" && (
                        <div>
                            <dt>Name</dt>
                            <dd className="name">{name}</dd>
                        </div>
                    )}
                    <div>
                        <dt>{labelOf(caller)}</dt>
                        <dd><Copyable text={idOf(caller)}/></dd>
                    </div>
                    {other !== undefined && (
                        <div>
                            <dt>{labelOf(other)}</dt>
                            <dd>
                                <Copyable text={idOf(other)}/>
                                <button type="button" className="button-small" onClick={() => onOpen({caller: other, row})}>
                                    Open
                                </button>
                            </dd>
                        </div>
                    )}
                </dl>
                {row !== undefined && <PlayerStats player={row}/>}
            </article>

            <section className="panel" aria-label="Antibot">
                <div className="panel-head">
                    <h2>Antibot</h2>
                    <button type="button" className="button-small" onClick={again}>Inspect again</button>
                </div>
                {inspection.state === "loading" && <p className="muted">Loading…</p>}
                {inspection.state === "failed" && <FailureNotice failure={inspection} onRetry={again}/>}
                {inspection.state === "done" && !inspection.value.on && <p className="muted">{ANTIBOT_OFF}</p>}
                {inspection.state === "done" && inspection.value.on && (
                    <Inspection
                        value={inspection.value.value}
                        caller={caller}
                        onOpenScope={scope => onOpen({caller: {scope}})}
                    />
                )}
            </section>

            <section className="panel" aria-label="Ban and mute">
                <h2>{isAccount(caller) ? "Ban and mute" : "Ban"}</h2>
                <p className="hint">
                    {isAccount(caller)
                        ? "An account is banned alone, from any address. A guest drops its account with a new cookie: ban its address too."
                        : "An address is banned as its scope (the address over IPv4, the /64 over IPv6): every account behind it."}
                </p>
                <BanAction planet={clients.planet} caller={caller} who={who} onDone={again}/>
                <UnbanAction planet={clients.planet} caller={caller} who={who} onDone={again}/>
                {isAccount(caller) && <MuteAction chat={clients.chat} accountId={caller.accountId} who={who}/>}
            </section>

            <RevertPanel planet={clients.planet} caller={caller} who={who}/>

            {isAccount(caller) && <GrantPanel planet={clients.planet} accountId={caller.accountId} who={who}/>}
        </section>
    )
}

// The other half of a ledger row: the address of an account opened from it, or the account of an address.
function otherOf({caller, row}: Opened): Caller | undefined {
    if (row === undefined) return undefined
    if (isAccount(caller)) return row.scope !== "" ? {scope: row.scope} : undefined
    return row.accountId !== "" ? {accountId: row.accountId} : undefined
}

function labelOf(caller: Caller): string {
    return isAccount(caller) ? "Account" : "Address"
}
