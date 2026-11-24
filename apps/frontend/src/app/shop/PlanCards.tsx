import {Plan, PlanPrice, bestValueOf, savingOf} from "../../domain/shop.ts"
import {t} from "../../i18n/page.ts"
import {CheckIcon} from "../components/icons.tsx"
import Money from "./Money.tsx"
import {cadenceOf, perDayLine, planName, savingLine} from "./prices.ts"
import "./Shop.css"

export type PlanCardsProps = {
    plans: readonly PlanPrice[]
    picked: Plan
    onPick: (plan: Plan) => void
    disabled?: boolean
}

// The plans side by side, each priced by the day against the others, the cheapest by the day marked the best value.
export default function PlanCards({plans, picked, onPick, disabled}: PlanCardsProps) {
    const best = bestValueOf(plans)

    return <div className="shop-plans" role="radiogroup" aria-label={t("shop.plans")}>
        {plans.map((price) => {
            const saving = savingOf(price, plans)
            const chosen = price.plan === picked
            const classes = ["shop-plan", price.plan === best && "shop-plan--best", chosen && "shop-plan--picked"]
            return <button key={price.plan}
                           type="button"
                           role="radio"
                           aria-checked={chosen}
                           disabled={disabled}
                           className={classes.filter(Boolean).join(" ")}
                           onClick={() => onPick(price.plan)}>
                {price.plan === best && <span className="shop-plan-ribbon">{t("shop.bestValue")}</span>}
                <span className="shop-plan-name">{planName(price.plan)}</span>
                <Money price={price} className="shop-plan-amount"/>
                <span className="shop-plan-cadence">{cadenceOf(price)}</span>
                <span className="shop-plan-deal">
                    {saving > 0 && <span className="shop-plan-saving">{savingLine(saving)}</span>}
                    <span className="shop-plan-daily">{perDayLine(price)}</span>
                </span>
                <span className="shop-plan-radio">{chosen && <CheckIcon size={14}/>}</span>
            </button>
        })}
    </div>
}
