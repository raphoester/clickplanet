import {useCallback, useState} from "react"
import {BonusesAdmin, QuizzesAdmin} from "../clients.ts"
import {RuleSet as BonusesRuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {RuleSet as QuizzesRuleSet} from "../gen/grpc/quizzes/v1/admin_pb.ts"
import * as bonusRules from "./bonusRules.ts"
import * as quizRules from "./quizRules.ts"
import {Column, RuleSetsPanel} from "./RuleSetsPanel.tsx"

const BOX_COLUMNS: Column<BonusesRuleSet>[] = [
    {title: "Offers", of: bonusRules.offersOf},
    {title: "Boxes", of: set => bonusRules.boxesOf(set.rules?.boxes)},
]

const QUIZ_COLUMNS: Column<QuizzesRuleSet>[] = [
    {title: "Offers", of: quizRules.offersOf},
    {title: "Quizzes", of: set => quizRules.quizzesOf(set.rules)},
]

// The offers' rule sets and the quizzes', on one page. While a set is edited, its form takes the page's place: the
// other list goes, and is read again when the form closes.
export function BonusRulesPanel({bonuses, quizzes}: {bonuses: BonusesAdmin, quizzes: QuizzesAdmin}) {
    const [open, setOpen] = useState<"boxes" | "quizzes" | undefined>(undefined)
    const readBoxes = useCallback((signal: AbortSignal) => bonuses.getRules({}, {signal}), [bonuses])
    const readQuizzes = useCallback((signal: AbortSignal) => quizzes.getRules({}, {signal}), [quizzes])

    return (
        <div className="detail">
            {open !== "quizzes" && (
                <RuleSetsPanel
                    title="Boxes"
                    hint="What a hand holds and what a charge does are the Game rules' Charges."
                    book={bonusRules.BOOK}
                    columns={BOX_COLUMNS}
                    read={readBoxes}
                    save={draft => bonusRules.saveRuleSet(bonuses, draft)}
                    onOpen={opened => setOpen(opened ? "boxes" : undefined)}
                />
            )}
            {open !== "boxes" && (
                <RuleSetsPanel
                    title="Quizzes"
                    hint="What a right answer grants is in the boxes' sets."
                    book={quizRules.BOOK}
                    columns={QUIZ_COLUMNS}
                    read={readQuizzes}
                    save={draft => quizRules.saveRuleSet(quizzes, draft)}
                    onOpen={opened => setOpen(opened ? "quizzes" : undefined)}
                />
            )}
        </div>
    )
}
