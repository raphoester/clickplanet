import {foundedFlags} from "../domain/flagSubmissions.ts"
import {canAddTime, CheckoutTarget, flagRenewal, payable, Plan, PlanPrice, Renewal} from "../domain/shop.ts"
import {FakeFlagsBackend} from "./fakeFlagsBackend.ts"
import {FakeFlagSubmissions} from "./fakeFlagSubmissions.ts"
import {CheckoutRequest, Shop, ShopRefusal, ShopRefusedError} from "./shop.ts"

const ANSWER_DELAY_MS = 400
// As Stripe's webhook, a moment after the buyer is back in the game.
const PAID_LANDS_MS = 2_000
const DAY_MS = 24 * 60 * 60 * 1000

const PLANS: readonly PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6000, currency: "eur", renews: true},
]

const DAYS: Record<Plan, number> = {week: 7, month: 30, year: 365}

// The shop module in memory, on by default. A checkout is paid at once: the page comes back from "Stripe" with
// ?shop=paid, and the flag opens or gets its time a moment later, as the webhook would. shopOff() and shopOn() on the
// console, and refuseCheckouts(refusal).
export class FakeShop implements Shop {
    private on = true
    private refusal?: ShopRefusal
    private readonly renewals: Renewal[] = []
    // Paid, not landed yet: a second checkout of the same target is refused until then.
    private readonly landing = new Set<string>()

    constructor(
        private readonly flags: FakeFlagsBackend,
        private readonly submissions: FakeFlagSubmissions,
        private readonly linked: () => Promise<boolean>,
        private readonly origin: string,
        private readonly paidLandsMs = PAID_LANDS_MS,
        private readonly delayMs = ANSWER_DELAY_MS,
        private readonly now: () => number = Date.now,
    ) {
    }

    public turn(on: boolean): string {
        this.on = on
        return on ? "the shop is on: a flag sent offers its plans" : "the shop is off: no plan, no pay button"
    }

    public refuse(refusal?: ShopRefusal): string {
        this.refusal = refusal
        return refusal ? `the next checkouts are refused: ${refusal}` : "the next checkouts go through"
    }

    public async plans(): Promise<PlanPrice[]> {
        await this.wait(this.delayMs / 4)
        return this.on ? PLANS.map((plan) => ({...plan})) : []
    }

    public async checkout({plan, target, startsNow}: CheckoutRequest): Promise<string> {
        await this.wait(this.delayMs)
        if (!await this.linked()) throw new ShopRefusedError("not-yours")
        if (!startsNow) throw new ShopRefusedError("invalid")
        if (!this.on) throw new ShopRefusedError("unavailable")
        if (this.refusal) throw new ShopRefusedError(this.refusal)
        if (this.landing.has(keyOf(target))) throw new ShopRefusedError("unavailable")

        const days = DAYS[plan]
        const land = target.kind === "submission" ? await this.forSubmission(target.id, days) : await this.forFlag(target.flag, days)
        if (plan !== "week") {
            this.renewals.push({id: crypto.randomUUID(), plan, target, state: "renewing", periodEndsAt: this.now() + days * DAY_MS})
        }
        this.landing.add(keyOf(target))
        setTimeout(() => void land().finally(() => this.landing.delete(keyOf(target))), this.paidLandsMs)
        return `${this.origin}/play?shop=paid`
    }

    public async myRenewals(): Promise<Renewal[]> {
        await this.wait(this.delayMs / 4)
        if (!await this.linked()) return []
        return this.renewals.map(copyOf)
    }

    public async stopRenewal(id: string): Promise<Renewal> {
        await this.wait(this.delayMs)
        if (!await this.linked()) throw new ShopRefusedError("not-yours")
        if (!this.on) throw new ShopRefusedError("unavailable")
        const renewal = this.renewals.find((kept) => kept.id === id)
        if (!renewal) throw new ShopRefusedError("not-found")
        renewal.state = "stopping"
        return copyOf(renewal)
    }

    private async forSubmission(id: string, days: number): Promise<() => Promise<void>> {
        const submission = (await this.submissions.mySubmissions()).find((mine) => mine.id === id)
        if (!submission) throw new ShopRefusedError("not-found")
        if (!payable(submission)) throw new ShopRefusedError("unavailable")
        if (submission.state === "waiting") {
            return async () => {
                if (!this.submissions.hold(id, days)) console.error(`the fake shop could not hold the time of ${submission.name}`)
            }
        }
        return async () => {
            const opened = await this.submissions.openApproved(id, days)
            if ("refused" in opened) console.error(`the fake shop could not open ${submission.name}: ${opened.refused}`)
        }
    }

    private async forFlag(flag: string, days: number): Promise<() => Promise<void>> {
        const submissions = await this.submissions.mySubmissions()
        if (!foundedFlags(submissions).includes(flag)) throw new ShopRefusedError("not-yours")
        const listed = (await this.flags.listFlags()).find(({code}) => code === flag)
        if (listed?.sold === undefined) throw new ShopRefusedError("not-found")
        if (!canAddTime(listed.sold, listed.closed, this.now())) throw new ShopRefusedError("unavailable")
        if (flagRenewal(this.renewals, flag, submissions)) throw new ShopRefusedError("unavailable")
        return async () => {
            this.flags.addTime(flag, days)
        }
    }

    private wait(ms: number): Promise<void> {
        return new Promise((resolve) => setTimeout(resolve, ms))
    }
}

function keyOf(target: CheckoutTarget): string {
    return target.kind === "submission" ? `submission:${target.id}` : `flag:${target.flag}`
}

function copyOf(renewal: Renewal): Renewal {
    return {...renewal, target: {...renewal.target}}
}
