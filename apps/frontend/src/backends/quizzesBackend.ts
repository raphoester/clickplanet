import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {v4 as generateUUID} from "uuid"
import {QuizService} from "../gen/grpc/quizzes/v1/quizzes_connect.ts"
import {QuizEvent} from "../gen/grpc/quizzes/v1/quizzes_pb.ts"
import {BonusReward} from "../domain/bonus.ts"
import {QuizOffer, QuizOutcome, QuizQuestion} from "../domain/quiz.ts"
import {GrantSource, QuizMaster, SessionSource} from "./backend.ts"
import {now as budgetNow} from "./clickBudget.ts"
import {asBonusError} from "./planetBackend.ts"
import {NoSession, SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, NO_TIMEOUT, openStream} from "./transport.ts"

export function newQuizServiceClient(config: Config): PromiseClient<typeof QuizService> {
    return createPromiseClient(QuizService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export function quizOf(event: QuizEvent, at = budgetNow(), wallClock = Date.now()): QuizOffer | undefined {
    if (event.event.case !== "quizOffered") return undefined

    const offered = event.event.value

    return {
        token: offered.token,
        expiresAt: at + (Number(offered.expiresAtUnixMs) - wallClock),
    }
}

// The quizzes asked of the page, on a stream of their own. The quizzes pay nothing: a right answer's reward is a grant
// on planet's stream, which the grants say.
export class QuizzesBackend implements QuizMaster {
    private readonly quizCallbacks = new Map<string, (offer: QuizOffer) => void>()
    private stopQuizzes: () => void
    private readonly stopFollowing: () => void
    private streamToken: string | undefined

    constructor(
        private client: PromiseClient<typeof QuizService>,
        private grants: GrantSource,
        sessions: SessionSource,
        private session: SessionProvider = new NoSession(),
    ) {
        this.stopQuizzes = this.openQuizStream()
        this.stopFollowing = sessions.listenForSessions((token) => this.followSession(token))
        void this.followIdentity()
    }

    public close() {
        this.stopFollowing()
        this.stopQuizzes()
        this.quizCallbacks.clear()
    }

    // Opened as the planet's is, and beside it: a quiz goes only to a caller with it open.
    private openQuizStream(): () => void {
        return openStream(
            (signal) => {
                const token = this.session.heldIdentity()
                this.streamToken = token

                const headers = new Headers()
                if (token) headers.set(SESSION_HEADER, token)

                return this.client.listenForEvents({}, {signal, headers, timeoutMs: NO_TIMEOUT})
            },
            (event) => {
                const quiz = quizOf(event)
                if (quiz) this.quizCallbacks.forEach(callback => callback(quiz))
            },
            "quiz events",
        )
    }

    private async followIdentity(): Promise<void> {
        this.followSession(await this.session.identity())
    }

    // The stream reads the token only when it opens: a new one, as a guest signs in, opens it again as that account.
    private followSession(token: string | undefined): void {
        if (!token || token === this.streamToken) return

        this.streamToken = token
        this.stopQuizzes()
        this.stopQuizzes = this.openQuizStream()
    }

    public listenForQuizzes(onOffered: (offer: QuizOffer) => void): () => void {
        const id = generateUUID()
        this.quizCallbacks.set(id, onOffered)

        return () => this.quizCallbacks.delete(id)
    }

    public listenForRewards(onRewarded: (reward: BonusReward) => void): () => void {
        return this.grants.listenForGrants((rewards) => rewards.forEach(onRewarded))
    }

    public async openQuiz(token: string): Promise<QuizQuestion> {
        try {
            return await this.open(token)
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw asBonusError(e)

            this.session.invalidate()

            try {
                return await this.open(token)
            } catch (retried) {
                throw asBonusError(retried)
            }
        }
    }

    private async open(token: string): Promise<QuizQuestion> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        const at = budgetNow()
        const wallClock = Date.now()
        const res = await this.client.openQuiz({token}, {headers})
        this.followSession(sessionToken)

        return {
            text: res.question,
            choices: [...res.choices],
            deadline: at + (Number(res.deadlineUnixMs) - wallClock),
            window: res.answerSeconds * 1000,
        }
    }

    public async answerQuiz(token: string, choice: number, countryId: string): Promise<QuizOutcome> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        let res
        try {
            res = await this.client.answerQuiz({token, choice, countryId}, {headers})
        } catch (e) {
            throw asBonusError(e)
        }
        this.followSession(sessionToken)

        return {correct: res.correct, correctChoice: res.correctChoice}
    }
}
