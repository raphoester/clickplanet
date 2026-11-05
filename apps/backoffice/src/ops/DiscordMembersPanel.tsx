import {AlertsAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {ALERTS_OFF} from "../players/actions.ts"
import {wordsOf} from "../refusal.ts"
import {linkMembers} from "./actions.ts"
import {membersLinkedWords} from "./discordMembers.ts"

// For the players who signed in with Discord before the sign-in asked to join the server, or joined it by the invite:
// the bot links the ones in the server already. The others can only join by their own Discord sign-in, which the game
// offers them as "Join Discord". Harmless twice.
export function DiscordMembersPanel({alerts}: {alerts: AlertsAdmin}) {
    return (
        <section className="panel" aria-label="Link the Discord members">
            <p className="hint">
                A player who signed in with Discord before the sign-in joined the server, or who joined it by the
                invite, is not linked: no flag role, no pings. This links every such player who is in the server
                already, with the role of their flag. Discord lets nobody else add a player to the server: the others
                see "Join Discord" in the game, and their sign-in joins them.
            </p>
            <ConfirmAction
                label="Link Discord members"
                question="Give the flag's role to every player signed in with Discord who is in the server and not linked?"
                run={() => linkMembers(alerts)}
                said={membersLinkedWords}
                refusal={err => wordsOf(err, {unimplemented: ALERTS_OFF})}
            />
        </section>
    )
}
