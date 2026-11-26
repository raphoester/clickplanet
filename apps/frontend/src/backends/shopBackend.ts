import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {ShopService} from "../gen/grpc/shop/v1/shop_connect.ts"
import {
    Plan as PlanPb,
    PlanPrice as PlanPricePb,
    Renewal as RenewalPb,
    RenewalState as RenewalStatePb,
    Sale as SalePb,
} from "../gen/grpc/shop/v1/shop_pb.ts"
import type {Catalog, CheckoutTarget, Plan, PlanPrice, Renewal, RenewalState, Sale} from "../domain/shop.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {CheckoutRequest, Shop, ShopRefusal, ShopRefusedError} from "./shop.ts"
import {Config, retrying} from "./transport.ts"

export function newShopServiceClient(config: Config): PromiseClient<typeof ShopService> {
    return createPromiseClient(ShopService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

const PLANS: Record<PlanPb, Plan | undefined> = {
    [PlanPb.UNSPECIFIED]: undefined,
    [PlanPb.WEEK]: "week",
    [PlanPb.MONTH]: "month",
    [PlanPb.YEAR]: "year",
}

const PLANS_PB: Record<Plan, PlanPb> = {
    week: PlanPb.WEEK,
    month: PlanPb.MONTH,
    year: PlanPb.YEAR,
}

const STATES: Record<RenewalStatePb, RenewalState | undefined> = {
    [RenewalStatePb.UNSPECIFIED]: undefined,
    [RenewalStatePb.RENEWING]: "renewing",
    [RenewalStatePb.STOPPING]: "stopping",
}

// A plan or a state this build does not know is dropped: a new enum value must not break an old tab.
export function planPriceOf({plan, amount, currency, renews}: PlanPricePb): PlanPrice | undefined {
    const known = PLANS[plan]
    return known === undefined ? undefined : {plan: known, amount, currency, renews}
}

// A sale this build cannot price is no sale: the plans show at their full price.
export function saleOf(sale: SalePb | undefined): Sale | undefined {
    if (sale === undefined || sale.endsAt === undefined || sale.percentOff < 1 || sale.percentOff > 99) return undefined
    return {percentOff: sale.percentOff, endsAt: sale.endsAt.toDate().getTime()}
}

export function renewalOf(renewal: RenewalPb): Renewal | undefined {
    const plan = PLANS[renewal.plan]
    const state = STATES[renewal.state]
    const target: CheckoutTarget | undefined = renewal.target.case === "submissionId"
        ? {kind: "submission", id: renewal.target.value}
        : renewal.target.case === "flag" ? {kind: "flag", flag: renewal.target.value} : undefined
    if (plan === undefined || state === undefined || target === undefined) return undefined
    return {id: renewal.id, plan, target, state, periodEndsAt: renewal.periodEndsAt?.toDate().getTime() ?? 0}
}

const absent = (e: unknown) => e instanceof ConnectError && [Code.Unimplemented, Code.Unauthenticated].includes(e.code)

// The shop module, with the token that names the account, as ListMySubmissions sends it; ListPlans with none.
export class ConnectShop implements Shop {
    constructor(
        private readonly client: PromiseClient<typeof ShopService>,
        private readonly session: SessionProvider,
    ) {
    }

    public async catalog(): Promise<Catalog> {
        try {
            const res = await retrying(() => this.client.listPlans({}), "ListPlans")
            const plans = res.plans.flatMap((plan) => planPriceOf(plan) ?? [])
            const sale = saleOf(res.sale)
            return sale === undefined ? {plans} : {plans, sale}
        } catch (e) {
            if (absent(e)) return {plans: []}
            throw e
        }
    }

    // Not retried: each call is a new page on Stripe.
    public async checkout({plan, target, startsNow}: CheckoutRequest): Promise<string> {
        const headers = await this.headers()
        try {
            const res = await this.client.createCheckout({
                plan: PLANS_PB[plan],
                target: target.kind === "submission" ? {case: "submissionId", value: target.id} : {case: "flag", value: target.flag},
                startsNow,
            }, {headers})
            if (res.url === "") throw new Error("the shop answered no page")
            return res.url
        } catch (e) {
            throw refusalOf(e)
        }
    }

    public async myRenewals(): Promise<Renewal[]> {
        const token = await this.session.identity()
        if (!token) return []
        const headers = new Headers({[SESSION_HEADER]: token})
        try {
            const res = await retrying(() => this.client.listMyRenewals({}, {headers}), "ListMyRenewals")
            return res.renewals.flatMap((renewal) => renewalOf(renewal) ?? [])
        } catch (e) {
            if (absent(e)) return []
            throw e
        }
    }

    public async stopRenewal(id: string): Promise<Renewal> {
        const headers = await this.headers()
        try {
            const res = await this.client.stopRenewal({renewalId: id}, {headers})
            const renewal = res.renewal && renewalOf(res.renewal)
            if (!renewal) throw new Error("the shop answered no renewal it can read")
            return renewal
        } catch (e) {
            throw refusalOf(e)
        }
    }

    private async headers(): Promise<Headers> {
        const token = await this.session.identity()
        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }
}

const REFUSALS: Partial<Record<Code, ShopRefusal>> = {
    [Code.PermissionDenied]: "not-yours",
    [Code.NotFound]: "not-found",
    [Code.FailedPrecondition]: "unavailable",
    [Code.InvalidArgument]: "invalid",
}

function refusalOf(e: unknown): unknown {
    if (!(e instanceof ConnectError)) return e
    const refusal = REFUSALS[e.code]
    return refusal ? new ShopRefusedError(refusal, {cause: e}) : e
}
