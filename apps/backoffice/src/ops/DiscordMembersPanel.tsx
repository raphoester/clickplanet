import {AlertsAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {ALERTS_OFF} from "../players/actions.ts"
import {wordsOf} from "../refusal.ts"
import {linkMembers} from "./actions.ts"
import {membersLinkedWords} from "./discordMembers.ts"

// For the players who signed in with Discord before the sign-in asked to join the server, or joined it by the invite:
// the bot links the ones in the server already. The others can only join by their own Discord sign-in, which the game
// offers them as "Join Discord". Every linked player is given its username as its nickname, the ones linked before the
// nicknames too. Harmless twice.
export function DiscordMembersPanel({alerts}: {alerts: AlertsAdmin}) {
    return (
        <section className="panel" aria-label="Link the Discord members">
            <p className="hint">
                A player who signed in with Discord before the sign-in joined the server, or who joined it by the
                invite, is not linked: no flag role, no pings. This links every such player who is in the server
                already, with the role of their flag. Discord lets nobody else add a player to the server: the others
                see "Join Discord" in the game, and their sign-in joins them. Every linked player is also given their
                username as their nickname in the server, if they do not wear it yet: the server's owner and the
                members above the bot keep their own, and are counted as not named.
            </p>
            <ConfirmAction
                label="Link Discord members"
                question="Give the flag's role to every player signed in with Discord who is in the server and not linked, and every linked player their username as their nickname?"
                run={() => linkMembers(alerts)}
                said={membersLinkedWords}
                refusal={err => wordsOf(err, {unimplemented: ALERTS_OFF})}
            />
        </section>
    )
}
