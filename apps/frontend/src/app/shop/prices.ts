import type {Plan, PlanPrice} from "../../domain/shop.ts"
import {t, translator} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"

const NAMES: Record<Plan, Key> = {
    week: "shop.plan.week",
    month: "shop.plan.month",
    year: "shop.plan.year",
}

export function planName(plan: Plan): string {
    return t(NAMES[plan])
}

// In the currency's smallest unit; whole amounts with no cents.
function priceOf(amount: number, currency: string): string {
    const digits = amount % 100 === 0 ? 0 : 2
    return translator().number(amount / 100, {
        style: "currency",
        currency: currency.toUpperCase(),
        minimumFractionDigits: digits,
        maximumFractionDigits: digits,
    })
}

// "€4", or for a plan that renews "€10 / month".
export function priceLine({plan, amount, currency, renews}: PlanPrice): string {
    const price = priceOf(amount, currency)
    if (!renews) return price
    switch (plan) {
        case "week":
            return t("shop.price.week", {price})
        case "month":
            return t("shop.price.month", {price})
        case "year":
            return t("shop.price.year", {price})
    }
}
