import {PlayerAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {wordsOf} from "../refusal.ts"
import {countOf} from "../support/labels.ts"
import {recountStats, reconcileTitles} from "./actions.ts"

// A run after a deploy, harmless twice: a second run changes nothing.
export function TitlesPanel({player}: {player: PlayerAdmin}) {
    return (
        <section className="panel" aria-label="Reconcile the titles">
            <p className="hint">
                After a deploy that adds a title or changes a rule: every account gets the titles the rules give it, no
                more. A guest holds none. Between such deploys there is nothing to run.
            </p>
            <ConfirmAction
                label="Reconcile titles"
                question="Grant every earned title and revoke every title no longer earned?"
                run={() => reconcileTitles(player)}
                said={done => done.granted === 0 && done.revoked === 0
                    ? "Nothing changed: every title was already right."
                    : `Granted ${countOf(done.granted, "title")}, revoked ${done.revoked}.`}
                refusal={wordsOf}
            />
            <p className="hint">
                Bombs and fortifies are in the ledger since October: this sets each player's count to it, and the titles
                follow. Run it once after the deploy that brought those tracks, then reconcile is enough.
            </p>
            <ConfirmAction
                label="Recount bombs and fortifies"
                question="Set every player's bombs and fortified tiles to what the ledger holds?"
                run={() => recountStats(player)}
                said={recounted => recounted === 0
                    ? "Nothing changed: every count was already the ledger's."
                    : `Recounted ${countOf(recounted, "player")}.`}
                refusal={wordsOf}
            />
        </section>
    )
}
