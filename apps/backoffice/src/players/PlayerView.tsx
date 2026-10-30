import {useCallback, useState} from "react"
import {theirTitleOf} from "../chat/theirConversations.ts"
import {TheirMessages} from "../chat/TheirMessages.tsx"
import {Clients} from "../clients.ts"
import {Copyable} from "../components/AccountId.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {Fold} from "../components/Fold.tsx"
import {Ago} from "../components/When.tsx"
import {Player} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Opened} from "../navigation.ts"
import {useCall} from "../useCall.ts"
import {ANTIBOT_OFF, mutingOf, Profile, profileOf, readingOf} from "./actions.ts"
import {Caller, idOf, isAccount, whoOf} from "./caller.ts"
import {GrantPanel} from "./GrantPanel.tsx"
import {Inspection} from "./InspectPanel.tsx"
import {BanAction, MuteAction, UnbanAction, UnmuteAction} from "./Moderation.tsx"
import {PlayerName} from "./PlayerName.tsx"
import {PlayerStats} from "./PlayerStats.tsx"
import {ProfilePanel} from "./ProfilePanel.tsx"
import {RevertPanel} from "./RevertPanel.tsx"
import {Standing} from "./Standing.tsx"

type Props = {
    clients: Clients
    opened: Opened
    onBack: () => void
    onOpen: (opened: Opened) => void
}

type Tool = "ban" | "unban" | "mute" | "unmute" | "revert" | "charges"

// One player, by account or by address: who it is and where it stands, the tools that take it, then what the game
// and the antibot keep about it, and what it said.
export function PlayerView({clients, opened, onBack, onOpen}: Props) {
    const {caller, row} = opened
    const account = isAccount(caller)

    const readProfile = useCallback(
        (signal: AbortSignal) => isAccount(caller) ? profileOf(clients.player, caller.accountId, signal) : Promise.resolve(undefined),
        [clients.player, caller],
    )
    const [profile, readProfileAgain] = useCall(readProfile)
    const kept = profile.state === "done" ? profile.value : undefined
    const name = kept?.name ?? opened.name ?? ""
    const who = whoOf(caller, name)

    const read = useCallback((signal: AbortSignal) => readingOf(clients.antibot, caller, signal), [clients.antibot, caller])
    const [reading, readAgain] = useCall(read)
    const readMuting = useCallback(
        (signal: AbortSignal) => isAccount(caller) ? mutingOf(clients.chat, caller.accountId, signal) : Promise.resolve(undefined),
        [clients.chat, caller],
    )
    const [muting, readMutingAgain] = useCall(readMuting)
    const other = otherOf(caller, row, reading.state === "done" && reading.value.on ? reading.value.value.scope : "")

    return (
        <section className="detail" aria-label="Player">
            <button type="button" className="button-small back" onClick={onBack}>← Players</button>
            <article className="card player-head">
                <div className="card-head">
                    <h2 className="player-heading">
                        {account
                            ? <PlayerName name={name} flag={shownFlagOf(row, kept)} unnamed="Unnamed"/>
                            : <span className="mono">{idOf(caller)}</span>}
                    </h2>
                    {row?.lastAt !== undefined && <span className="muted">last take <Ago at={row.lastAt}/></span>}
                </div>
                <Standing reading={reading} muting={account ? muting : undefined}/>
                <dl className="people">
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
            <Tools clients={clients} caller={caller} who={who} onBanned={readAgain} onMuted={readMutingAgain}/>
            <div className="columns">
                {account && <ProfilePanel profile={profile} onReload={readProfileAgain}/>}
                <section className="panel" aria-label="Antibot">
                    <div className="panel-head">
                        <h2>Antibot</h2>
                        <button type="button" className="button-small" onClick={readAgain}>Read again</button>
                    </div>
                    {reading.state === "loading" && <p className="muted">Loading…</p>}
                    {reading.state === "failed" && <FailureNotice failure={reading} onRetry={readAgain}/>}
                    {reading.state === "done" && !reading.value.on && <p className="muted">{ANTIBOT_OFF}</p>}
                    {reading.state === "done" && reading.value.on && <Inspection value={reading.value.value} caller={caller}/>}
                </section>
            </div>
            {isAccount(caller) && (
                <Fold label="Their messages" title={theirTitleOf(name)} sub={messagesOf(kept)}>
                    <TheirMessages chat={clients.chat} accountId={caller.accountId} parties={undefined}/>
                </Fold>
            )}
        </section>
    )
}

type ToolsProps = {clients: Clients, caller: Caller, who: string, onBanned: () => void, onMuted: () => void}

// Every tool on one player, one at a time: the page stays short, and the one in use sits under the player.
function Tools({clients, caller, who, onBanned, onMuted}: ToolsProps) {
    const [tool, setTool] = useState<Tool>("ban")
    const tools: {key: Tool, label: string}[] = [
        {key: "ban", label: "Ban"},
        {key: "unban", label: "Unban"},
        ...(isAccount(caller) ? [{key: "mute" as const, label: "Mute"}, {key: "unmute" as const, label: "Unmute"}] : []),
        {key: "revert", label: "Revert"},
        ...(isAccount(caller) ? [{key: "charges" as const, label: "Charges"}] : []),
    ]
    return (
        <section className="panel tools" aria-label="Tools">
            <div className="segmented" role="tablist" aria-label="Tool">
                {tools.map(each => (
                    <button
                        key={each.key}
                        type="button"
                        role="tab"
                        aria-selected={tool === each.key}
                        onClick={() => setTool(each.key)}
                    >
                        {each.label}
                    </button>
                ))}
            </div>
            <div role="tabpanel" aria-label={tools.find(each => each.key === tool)?.label}>
                {tool === "ban" && (
                    <>
                        <p className="hint">
                            {isAccount(caller)
                                ? "An account is banned alone, from any address. A guest drops its account with a new cookie: ban its address too."
                                : "An address is banned as its scope (the address over IPv4, the /64 over IPv6): every account behind it."}
                        </p>
                        <BanAction antibot={clients.antibot} caller={caller} who={who} onDone={onBanned}/>
                    </>
                )}
                {tool === "unban" && <UnbanAction antibot={clients.antibot} caller={caller} who={who} onDone={onBanned}/>}
                {tool === "mute" && isAccount(caller) && (
                    <MuteAction chat={clients.chat} accountId={caller.accountId} who={who} onDone={onMuted}/>
                )}
                {tool === "unmute" && isAccount(caller) && (
                    <UnmuteAction chat={clients.chat} accountId={caller.accountId} who={who} onDone={onMuted}/>
                )}
                {tool === "revert" && <RevertPanel planet={clients.planet} caller={caller} who={who}/>}
                {tool === "charges" && isAccount(caller) && <GrantPanel planet={clients.planet} accountId={caller.accountId} who={who}/>}
            </div>
        </section>
    )
}

function messagesOf(profile: Profile | undefined): string | undefined {
    if (profile === undefined) return undefined
    const sent = Number(profile.messagesSent)
    return sent === 1 ? "1 message" : `${sent} messages`
}

// The flag the row painted most in the ledger's 72 hours, or else the one the account took the most tiles for.
function shownFlagOf(row: Player | undefined, profile: Profile | undefined): string {
    if (row !== undefined && row.flagCountryId !== "") return row.flagCountryId
    return profile?.playsFor[0]?.countryId ?? ""
}

// The other half of a player: the address of an account, from its row or else where the antibot last read it, or
// the account of an address's row.
function otherOf(caller: Caller, row: Player | undefined, scope: string): Caller | undefined {
    if (isAccount(caller)) {
        const address = row?.scope || scope
        return address !== "" ? {scope: address} : undefined
    }
    return row !== undefined && row.accountId !== "" ? {accountId: row.accountId} : undefined
}

function labelOf(caller: Caller): string {
    return isAccount(caller) ? "Account" : "Address"
}
