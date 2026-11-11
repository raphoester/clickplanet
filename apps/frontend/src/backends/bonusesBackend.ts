import {createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {BonusService} from "../gen/grpc/bonuses/v1/bonuses_connect.ts"
import {BonusEvent} from "../gen/grpc/bonuses/v1/bonuses_pb.ts"
import {QuizOffer} from "../domain/quiz.ts"
import {BonusCatch, BonusOffer} from "./backend.ts"
import {rewardOf} from "./bonusKind.ts"
import {now as budgetNow} from "./clickBudget.ts"
import {Config} from "./transport.ts"

// The boxes and quizzes the game offers: what they grant is planet's charges, which PlanetBackend holds.
export function newBonusServiceClient(config: Config): PromiseClient<typeof BonusService> {
    return createPromiseClient(BonusService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export function offerOf(event: BonusEvent, at = budgetNow(), wallClock = Date.now()): BonusOffer | undefined {
    if (event.event.case !== "bonusOffered") return undefined

    const offered = event.event.value

    const reward = rewardOf(offered.kind)
    if (!reward) return undefined

    return {
        token: offered.token,
        seed: offered.seed,
        reward,
        expiresAt: at + (Number(offered.expiresAtUnixMs) - wallClock),
    }
}

export function quizOf(event: BonusEvent, at = budgetNow(), wallClock = Date.now()): QuizOffer | undefined {
    if (event.event.case !== "quizOffered") return undefined

    const offered = event.event.value

    return {
        token: offered.token,
        expiresAt: at + (Number(offered.expiresAtUnixMs) - wallClock),
    }
}

export function catchOf(event: BonusEvent): BonusCatch | undefined {
    if (event.event.case !== "bonusTaken") return undefined

    const taken = event.event.value
    return {countryId: taken.countryId, quizSubject: taken.quizSubjectCountryId || undefined}
}

// The offers took another set: whether a first sign-in earns a refill may have moved.
export function grantRulesChanged(event: BonusEvent): boolean {
    return event.event.case === "rulesChanged"
}
