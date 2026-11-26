import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {Timestamp} from "@bufbuild/protobuf"
import {ShopService} from "../gen/grpc/shop/v1/shop_connect.ts"
import {ListPlansResponse, Plan, PlanPrice, Renewal, RenewalState, Sale} from "../gen/grpc/shop/v1/shop_pb.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {ShopRefusedError} from "./shop.ts"
import {ConnectShop, planPriceOf, renewalOf, saleOf} from "./shopBackend.ts"

const session = (): SessionProvider => ({
    token: vi.fn(), held: vi.fn(), identity: vi.fn(async () => "identity-1"), heldIdentity: vi.fn(), invalidate: vi.fn(),
})

const guestPage = (): SessionProvider => ({
    token: vi.fn(), held: vi.fn(), identity: vi.fn(async () => undefined), heldIdentity: vi.fn(), invalidate: vi.fn(),
})

const clientOf = (calls: Record<string, unknown>) => calls as unknown as PromiseClient<typeof ShopService>

const headersOf = (call: ReturnType<typeof vi.fn>) => (call.mock.calls[0][1] as {headers: Headers}).headers

const refusalOf = (sent: Promise<unknown>) =>
    sent.then(() => "sent", (e: unknown) => e instanceof ShopRefusedError ? e.refusal : e)

const renewing = () => new Renewal({
    id: "r-1", plan: Plan.MONTH, target: {case: "submissionId", value: "s-1"}, state: RenewalState.RENEWING,
    periodEndsAt: Timestamp.fromDate(new Date(5_000)),
})

describe("the shop's messages", () => {
    it("read a sale, and drop one they cannot price", () => {
        const endsAt = Timestamp.fromDate(new Date(5_000))
        expect(saleOf(new Sale({percentOff: 50, endsAt}))).toEqual({percentOff: 50, endsAt: 5_000})
        expect(saleOf(undefined)).toBeUndefined()
        expect(saleOf(new Sale({percentOff: 50}))).toBeUndefined()
        expect(saleOf(new Sale({percentOff: 0, endsAt}))).toBeUndefined()
        expect(saleOf(new Sale({percentOff: 100, endsAt}))).toBeUndefined()
    })

    it("read a plan's price, and drop a plan this build does not know", () => {
        expect(planPriceOf(new PlanPrice({plan: Plan.YEAR, amount: 6000, currency: "eur", renews: true})))
            .toEqual({plan: "year", amount: 6000, currency: "eur", renews: true})
        expect(planPriceOf(new PlanPrice({plan: 9 as Plan, amount: 100, currency: "eur"}))).toBeUndefined()
    })

    it("read a renewal with its target and its end in milliseconds, and drop one they cannot read", () => {
        expect(renewalOf(renewing())).toEqual({
            id: "r-1", plan: "month", target: {kind: "submission", id: "s-1"}, state: "renewing", periodEndsAt: 5_000,
        })
        expect(renewalOf(new Renewal({...renewing(), target: {case: "flag", value: "r-k3p9x2ma0b"}, state: RenewalState.STOPPING})))
            .toMatchObject({target: {kind: "flag", flag: "r-k3p9x2ma0b"}, state: "stopping"})
        expect(renewalOf(new Renewal({...renewing(), state: RenewalState.UNSPECIFIED}))).toBeUndefined()
        expect(renewalOf(new Renewal({...renewing(), target: {case: undefined}}))).toBeUndefined()
    })
})

describe("ConnectShop", () => {
    it("reads the plans with no token, and none from a server without the shop", async () => {
        const listPlans = vi.fn().mockResolvedValue({plans: [new PlanPrice({plan: Plan.WEEK, amount: 400, currency: "eur"})]})
        const catalog = await new ConnectShop(clientOf({listPlans}), session()).catalog()

        expect(catalog).toEqual({plans: [{plan: "week", amount: 400, currency: "eur", renews: false}]})
        expect(listPlans.mock.calls[0]).toEqual([{}])

        const absent = vi.fn().mockRejectedValue(new ConnectError("no", Code.Unimplemented))
        expect(await new ConnectShop(clientOf({listPlans: absent}), session()).catalog()).toEqual({plans: []})
    })

    it("reads the sale while one runs, its end in milliseconds", async () => {
        const listPlans = vi.fn().mockResolvedValue(new ListPlansResponse({
            plans: [new PlanPrice({plan: Plan.WEEK, amount: 400, currency: "eur"})],
            sale: new Sale({percentOff: 50, endsAt: Timestamp.fromDate(new Date("2026-11-30T22:59:00Z"))}),
        }))

        const {sale} = await new ConnectShop(clientOf({listPlans}), session()).catalog()

        expect(sale).toEqual({percentOff: 50, endsAt: Date.parse("2026-11-30T22:59:00Z")})
    })

    it("asks a checkout for the plan and the target with the token that names the account", async () => {
        const createCheckout = vi.fn().mockResolvedValue({url: "https://checkout.stripe.com/c/pay/cs_1"})
        const shop = new ConnectShop(clientOf({createCheckout}), session())

        expect(await shop.checkout({plan: "year", target: {kind: "flag", flag: "r-k3p9x2ma0b"}, startsNow: true}))
            .toBe("https://checkout.stripe.com/c/pay/cs_1")
        expect(createCheckout.mock.calls[0][0]).toEqual({plan: Plan.YEAR, target: {case: "flag", value: "r-k3p9x2ma0b"}, startsNow: true})
        expect(headersOf(createCheckout).get(SESSION_HEADER)).toBe("identity-1")
    })

    it.each([
        [Code.PermissionDenied, "not-yours"],
        [Code.NotFound, "not-found"],
        [Code.FailedPrecondition, "unavailable"],
        [Code.InvalidArgument, "invalid"],
    ])("says the server's %s as a refusal", async (code, refusal) => {
        const createCheckout = vi.fn().mockRejectedValue(new ConnectError("no", code))
        const sent = new ConnectShop(clientOf({createCheckout}), session())
            .checkout({plan: "week", target: {kind: "submission", id: "s-1"}, startsNow: true})

        expect(await refusalOf(sent)).toBe(refusal)
        expect(createCheckout).toHaveBeenCalledTimes(1)
    })

    it("reads the renewals with the token, and none for a guest", async () => {
        const listMyRenewals = vi.fn().mockResolvedValue({renewals: [renewing()]})
        expect(await new ConnectShop(clientOf({listMyRenewals}), session()).myRenewals()).toMatchObject([{id: "r-1"}])
        expect(headersOf(listMyRenewals).get(SESSION_HEADER)).toBe("identity-1")

        const guest = vi.fn()
        expect(await new ConnectShop(clientOf({listMyRenewals: guest}), guestPage()).myRenewals()).toEqual([])
        expect(guest).not.toHaveBeenCalled()
    })

    it("stops a renewal and reads it back, or says the refusal", async () => {
        const stopRenewal = vi.fn().mockResolvedValue({renewal: new Renewal({...renewing(), state: RenewalState.STOPPING})})
        const stopped = await new ConnectShop(clientOf({stopRenewal}), session()).stopRenewal("r-1")

        expect(stopRenewal.mock.calls[0][0]).toEqual({renewalId: "r-1"})
        expect(stopped.state).toBe("stopping")

        const missing = vi.fn().mockRejectedValue(new ConnectError("no", Code.NotFound))
        expect(await refusalOf(new ConnectShop(clientOf({stopRenewal: missing}), session()).stopRenewal("r-2"))).toBe("not-found")
    })
})
