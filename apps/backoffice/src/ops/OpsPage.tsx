import {Clients, CLIENTS} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {wordsOf} from "../refusal.ts"
import {countOf} from "../support/labels.ts"
import {nameAccounts, reconcileTitles} from "./actions.ts"

// The runs after a deploy. Each is harmless twice: a second run changes nothing.
export function OpsPage({clients = CLIENTS}: {clients?: Clients}) {
    return (
        <div className="detail">
            <section className="panel" aria-label="Reconcile the titles">
                <h2>Reconcile the players' titles</h2>
                <p className="hint">
                    After a deploy that adds a title or changes a rule: every account gets the titles the rules give it,
                    no more. A guest holds none. Between such deploys there is nothing to run.
                </p>
                <ConfirmAction
                    label="Reconcile titles"
                    question="Grant every earned title and revoke every title no longer earned?"
                    run={() => reconcileTitles(clients.player)}
                    said={done => done.granted === 0 && done.revoked === 0
                        ? "Nothing changed: every title was already right."
                        : `Granted ${countOf(done.granted, "title")}, revoked ${done.revoked}.`}
                    refusal={wordsOf}
                />
            </section>

            <section className="panel" aria-label="Name the accounts">
                <h2>Name the signed-in players with no username</h2>
                <p className="hint">
                    Once, after the deploy that brought generated names: an account that signs in since is named at
                    once. A name, chosen or given, is never replaced.
                </p>
                <ConfirmAction
                    label="Name accounts"
                    question="Give a generated name to every signed-in account that has none?"
                    run={() => nameAccounts(clients.player)}
                    said={done => done.named === 0
                        ? "Nobody was named: every signed-in account has a name."
                        : `Named ${countOf(done.named, "account")}.`}
                    refusal={wordsOf}
                />
            </section>
        </div>
    )
}
