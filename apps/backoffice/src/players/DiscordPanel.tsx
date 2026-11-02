import {useCallback} from "react"
import {AlertsAdmin} from "../clients.ts"
import {Copyable} from "../components/AccountId.tsx"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {Country} from "../components/Country.tsx"
import {Fact} from "../components/Fact.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {DiscordLink} from "../gen/grpc/alerts/v1/admin_pb.ts"
import {useCall} from "../useCall.ts"
import {ALERTS_OFF, linkingOf, refusalOf, unlink} from "./actions.ts"

type Props = {alerts: AlertsAdmin, accountId: string, who: string}

// The account's Discord link, which a Discord sign-in made: the user in the game's server, and the flag whose role it
// holds. Unlinked, the game offers the account to connect Discord again, which is how its offer is tried.
export function DiscordPanel({alerts, accountId, who}: Props) {
    const read = useCallback((signal: AbortSignal) => linkingOf(alerts, accountId, signal), [alerts, accountId])
    const [linking, readAgain] = useCall(read)

    if (linking.state === "loading") return <p className="muted">Loading…</p>
    if (linking.state === "failed") return <FailureNotice failure={linking} onRetry={readAgain}/>
    if (!linking.value.on) return <p className="muted">{ALERTS_OFF}</p>

    const link = linking.value.link
    if (link === undefined) {
        return <p className="hint">Not linked: the game offers {who} to join Discord, by signing in with it.</p>
    }
    return (
        <>
            <LinkFacts link={link}/>
            <div className="act">
                <ConfirmAction
                    label="Unlink Discord"
                    question={`Unlink ${who} from Discord? The bot takes the flag's role off them, they stay in the server, `
                        + "and the game offers them to join Discord again."}
                    run={() => unlink(alerts, accountId)}
                    said={unlinked => unlinked ? "Unlinked." : "It was not linked any more."}
                    refusal={err => refusalOf("unlink", err)}
                    onDone={readAgain}
                />
            </div>
        </>
    )
}

function LinkFacts({link}: {link: DiscordLink}) {
    return (
        <dl className="facts">
            <Fact label="Discord user"><Copyable text={link.discordUserId}/></Fact>
            <Fact label="Role">{link.countryId === "" ? <span className="muted">none yet: no flag pledged</span> : <Country id={link.countryId}/>}</Fact>
            <Fact label="Linked"><When at={new Date(Number(link.linkedAtUnixMs))}/></Fact>
        </dl>
    )
}
