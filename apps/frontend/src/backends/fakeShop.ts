import {Catalog, CheckoutTarget, flagRenewal, payable, Plan, PlanPrice, Renewal, Sale, saleRuns} from "../domain/shop.ts"
import {FakeFlagsBackend} from "./fakeFlagsBackend.ts"
import {FakeFlagSubmissions} from "./fakeFlagSubmissions.ts"
import {CheckoutRequest, Shop, ShopRefusal, ShopRefusedError} from "./shop.ts"

const ANSWER_DELAY_MS = 400
// As Stripe's webhook, a moment after the buyer is back in the game.
const PAID_LANDS_MS = 2_000
const DAY_MS = 24 * 60 * 60 * 1000
const MINUTE_MS = 60_000

const PLANS: readonly PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6000, currency: "eur", renews: true},
]

const DAYS: Record<Plan, number> = {week: 7, month: 30, year: 365}

// The shop module in memory, on by default, with no sale. A checkout is paid at once: the page comes back from "Stripe"
// with ?shop=paid, and the flag opens or gets its time a moment later, as the webhook would. shopOff() and shopOn() on
// the console, refuseCheckouts(refusal), renewingElsewhere(code), another player's renewal that still runs on a flag,
// startSale(minutes) and endSale().
export class FakeShop implements Shop {
    private on = true
    private refusal?: ShopRefusal
    private sale?: Sale
    private readonly renewals: Renewal[] = []
    // Flags another player's renewal still pays for, as while Stripe tries a card again: nobody else may claim them.
    private readonly renewedElsewhere = new Set<string>()
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

    public renewElsewhere(flag: string, on: boolean): string {
        if (on) this.renewedElsewhere.add(flag)
        else this.renewedElsewhere.delete(flag)
        return on ? `another player's renewal still runs on ${flag}: nobody else may pay for it` : `no other renewal runs on ${flag}`
    }

    // As the server's: percentOff off every plan's first payment, until it ends.
    public sell(minutes: number, percentOff = 50): string {
        this.sale = {percentOff, endsAt: this.now() + minutes * MINUTE_MS}
        return `${percentOff}% off every plan for ${minutes} minutes: the plans are read again`
    }

    public endSale(): string {
        this.sale = undefined
        return "no sale: the plans are read again"
    }

    public async catalog(): Promise<Catalog> {
        await this.wait(this.delayMs / 4)
        if (!this.on) return {plans: []}
        const plans = PLANS.map((plan) => ({...plan}))
        return saleRuns(this.sale, this.now()) ? {plans, sale: {...this.sale}} : {plans}
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
        const submission = (await this.submissions.myFlags()).submissions.find((mine) => mine.id === id)
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

    // Its holder's, or anyone's once its time ended, never a removed one, nor while a renewal of it runs.
    private async forFlag(flag: string, days: number): Promise<() => Promise<void>> {
        const {submissions} = await this.submissions.myFlags()
        const listed = (await this.flags.listFlags()).find(({code}) => code === flag)
        if (listed?.sold === undefined) throw new ShopRefusedError("not-found")
        if (listed.sold.removed) throw new ShopRefusedError("unavailable")
        if (!listed.closed && !this.flags.holds(flag)) throw new ShopRefusedError("not-yours")
        const renewing = flagRenewal(this.renewals.filter(({state}) => state === "renewing"), flag, submissions)
        if (renewing || this.renewedElsewhere.has(flag)) throw new ShopRefusedError("unavailable")
        return async () => {
            const paid = this.flags.pay(flag, days)
            if (paid === "removed" || paid === "held" || paid === "unknown") console.error(`the fake shop could not pay for ${flag}: ${paid}`)
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
