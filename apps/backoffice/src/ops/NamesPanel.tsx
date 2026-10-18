import {PlayerAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {wordsOf} from "../refusal.ts"
import {countOf} from "../support/labels.ts"
import {nameAccounts} from "./actions.ts"

// A run after a deploy, harmless twice: a name, chosen or given, is never replaced.
export function NamesPanel({player}: {player: PlayerAdmin}) {
    return (
        <section className="panel" aria-label="Name the accounts">
            <p className="hint">
                Gives a generated name to every signed-in player with no username. Once, after the deploy that brought
                generated names: an account that signs in since is named at once.
            </p>
            <ConfirmAction
                label="Name accounts"
                question="Give a generated name to every signed-in account that has none?"
                run={() => nameAccounts(player)}
                said={done => done.named === 0
                    ? "Nobody was named: every signed-in account has a name."
                    : `Named ${countOf(done.named, "account")}.`}
                refusal={wordsOf}
            />
        </section>
    )
}
