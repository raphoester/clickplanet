import {perDayOf, Plan, PlanPrice} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"

const NAMES: Record<Plan, Key> = {
    week: "shop.plan.week",
    month: "shop.plan.month",
    year: "shop.plan.year",
}

const EVERY: Record<Plan, Key> = {
    week: "shop.cadence.week",
    month: "shop.cadence.month",
    year: "shop.cadence.year",
}

export function planName(plan: Plan): string {
    return t(NAMES[plan])
}

// In the currency's smallest unit; whole amounts with no cents.
function moneyFormat(amount: number, currency: string): Intl.NumberFormatOptions {
    const digits = amount % 100 === 0 ? 0 : 2
    return {style: "currency", currency: currency.toUpperCase(), minimumFractionDigits: digits, maximumFractionDigits: digits}
}

function moneyOf(amount: number, currency: string): string {
    return translator().number(amount / 100, moneyFormat(amount, currency))
}

export type MoneyPart = {kind: "symbol" | "number", text: string}

// "€60" as its symbol and its number, in the language's order ("60 €"), with no space between: the symbol is set apart.
export function pricePartsOf({amount, currency}: PlanPrice): MoneyPart[] {
    const parts: MoneyPart[] = []
    for (const {type, value} of translator().numberParts(amount / 100, moneyFormat(amount, currency))) {
        if (type === "literal" && value.trim() === "") continue
        const kind = type === "currency" ? "symbol" : "number"
        const last = parts.at(-1)
        if (last?.kind === "number" && kind === "number") last.text += value
        else parts.push({kind, text: value})
    }
    return parts
}

// "Paid once", or for a plan that renews "Every month".
export function cadenceOf({plan, renews}: PlanPrice): string {
    return renews ? t(EVERY[plan]) : t("shop.cadence.once")
}

// "€0.17 / day".
export function perDayLine(price: PlanPrice): string {
    return t("shop.price.day", {price: moneyOf(perDayOf(price), price.currency)})
}

// "-42%".
export function savingLine(percent: number): string {
    return translator().number(-percent / 100, {style: "percent"})
}
