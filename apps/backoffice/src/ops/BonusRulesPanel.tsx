import {useCallback, useState} from "react"
import {BonusesAdmin, BoxesAdmin, QuizzesAdmin} from "../clients.ts"
import {RuleSet as BonusesRuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {RuleSet as BoxesRuleSet} from "../gen/grpc/boxes/v1/admin_pb.ts"
import {RuleSet as QuizzesRuleSet} from "../gen/grpc/quizzes/v1/admin_pb.ts"
import * as bonusRules from "./bonusRules.ts"
import * as boxRules from "./boxRules.ts"
import * as quizRules from "./quizRules.ts"
import {Column, RuleSetsPanel} from "./RuleSetsPanel.tsx"

const BOX_COLUMNS: Column<BoxesRuleSet>[] = [
    {title: "Offers", of: boxRules.offersOf},
    {title: "Boxes", of: set => boxRules.boxesOf(set.rules)},
]

const BONUS_COLUMNS: Column<BonusesRuleSet>[] = [
    {title: "Drawn as", of: set => bonusRules.kindsOf(set.rules)},
    {title: "Gates", of: set => bonusRules.gatesOf(set.rules)},
]

const QUIZ_COLUMNS: Column<QuizzesRuleSet>[] = [
    {title: "Offers", of: quizRules.offersOf},
    {title: "Quizzes", of: set => quizRules.quizzesOf(set.rules)},
]

type Opened = "boxes" | "bonuses" | "quizzes"

// The boxes' rule sets, what a box or a right answer holds, and the quizzes', on one page. While a set is edited, its
// form takes the page's place: the other lists go, and are read again when the form closes.
export function BonusRulesPanel({bonuses, boxes, quizzes}: {bonuses: BonusesAdmin, boxes: BoxesAdmin, quizzes: QuizzesAdmin}) {
    const [open, setOpen] = useState<Opened | undefined>(undefined)
    const readBoxes = useCallback((signal: AbortSignal) => boxes.getRules({}, {signal}), [boxes])
    const readBonuses = useCallback((signal: AbortSignal) => bonuses.getRules({}, {signal}), [bonuses])
    const readQuizzes = useCallback((signal: AbortSignal) => quizzes.getRules({}, {signal}), [quizzes])
    const shown = (list: Opened) => open === undefined || open === list
    const opens = (list: Opened) => (opened: boolean) => setOpen(opened ? list : undefined)

    return (
        <div className="detail">
            {shown("boxes") && (
                <RuleSetsPanel
                    title="Boxes"
                    hint="What a box holds is in the next list."
                    book={boxRules.BOOK}
                    columns={BOX_COLUMNS}
                    read={readBoxes}
                    save={draft => boxRules.saveRuleSet(boxes, draft)}
                    onOpen={opens("boxes")}
                />
            )}
            {shown("bonuses") && (
                <RuleSetsPanel
                    title="What a box or a right answer holds"
                    hint="What a hand holds and what a charge does are the Game rules' Charges."
                    book={bonusRules.BOOK}
                    columns={BONUS_COLUMNS}
                    read={readBonuses}
                    save={draft => bonusRules.saveRuleSet(bonuses, draft)}
                    onOpen={opens("bonuses")}
                />
            )}
            {shown("quizzes") && (
                <RuleSetsPanel
                    title="Quizzes"
                    hint="What a right answer holds is in the list above."
                    book={quizRules.BOOK}
                    columns={QUIZ_COLUMNS}
                    read={readQuizzes}
                    save={draft => quizRules.saveRuleSet(quizzes, draft)}
                    onOpen={opens("quizzes")}
                />
            )}
        </div>
    )
}
