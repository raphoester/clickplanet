import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    AnswerQuizResponse,
    Heartbeat,
    OpenQuizResponse,
    QuizEvent,
    QuizOffered,
    QuizWon,
} from "../gen/grpc/quizzes/v1/quizzes_pb.ts"
import {BonusReward} from "../domain/bonus.ts"
import {QuizOffer} from "../domain/quiz.ts"
import {BonusLostError, GrantSource, SessionSource} from "./backend.ts"
import {quizOf, QuizzesBackend} from "./quizzesBackend.ts"
import {NoSession, SESSION_HEADER, type SessionProvider} from "./session.ts"

const heartbeat = () => new QuizEvent({event: {case: "heartbeat", value: new Heartbeat()}})

type QuizzesListen = (request: object, options: {signal: AbortSignal, headers: Headers}) => AsyncIterable<QuizEvent>

// A quiz stream that opens and says nothing, until it is aborted.
function quietQuizzes() {
    return vi.fn<QuizzesListen>(() => (async function* () {
        await new Promise(() => {})
        yield* [] as QuizEvent[]
    })())
}

function quizzesWith(fields: Record<string, unknown> = {}) {
    return {listenForEvents: quietQuizzes(), ...fields} as never
}

class Grants implements GrantSource {
    private listeners: ((rewards: BonusReward[]) => void)[] = []

    listenForGrants(onGranted: (rewards: BonusReward[]) => void): () => void {
        this.listeners.push(onGranted)
        return () => {
            this.listeners = this.listeners.filter((listener) => listener !== onGranted)
        }
    }

    grant(...rewards: BonusReward[]) {
        this.listeners.forEach((listener) => listener(rewards))
    }
}

class Sessions implements SessionSource {
    private listeners: ((token: string) => void)[] = []

    listenForSessions(onSession: (token: string) => void): () => void {
        this.listeners.push(onSession)
        return () => {
            this.listeners = this.listeners.filter((listener) => listener !== onSession)
        }
    }

    carried(token: string) {
        this.listeners.forEach((listener) => listener(token))
    }
}

function fixedSession(token: string): SessionProvider {
    return {
        token: async () => token, held: () => token, identity: async () => token, heldIdentity: () => token,
        invalidate: () => {},
    }
}

const tokensOpenedWith = (listen: ReturnType<typeof quietQuizzes>) =>
    listen.mock.calls.map(call => call[1].headers.get(SESSION_HEADER))

describe("quizOf", () => {
    const asked = (fields: {token?: string, expiresAtUnixMs?: bigint} = {}) =>
        new QuizEvent({
            event: {
                case: "quizOffered",
                value: new QuizOffered({token: "a-token", expiresAtUnixMs: 1_000_000n, ...fields}),
            },
        })

    it("reads the banner the server addressed to this client", () => {
        expect(quizOf(asked())).toEqual({token: "a-token", expiresAt: expect.any(Number)})
    })

    it("is a token and a deadline, and nothing about the question at all", () => {
        expect(Object.keys(quizOf(asked())!)).toEqual(["token", "expiresAt"])
    })

    it("builds the deadline from how long is left, not from the server's clock", () => {
        expect(quizOf(asked({expiresAtUnixMs: 1_025_000n}), 5_000, 1_000_000)?.expiresAt).toBe(5_000 + 25_000)
    })

    it("drops everything that is not a banner", () => {
        expect(quizOf(heartbeat())).toBeUndefined()
        expect(quizOf(new QuizEvent({event: {case: "quizWon", value: new QuizWon({countryId: "fr"})}}))).toBeUndefined()
    })
})

describe("QuizzesBackend", () => {
    it("hands each quiz offered on its stream to the quiz listeners, and nothing else", async () => {
        const listenForEvents = vi.fn(async function* () {
            yield heartbeat()
            yield new QuizEvent({event: {case: "quizOffered", value: new QuizOffered({token: "quiz-1", expiresAtUnixMs: BigInt(Date.now() + 25_000)})}})
            yield new QuizEvent({event: {case: "quizWon", value: new QuizWon({countryId: "fr", subjectCountryId: "ee"})}})
            yield new QuizEvent()
            await new Promise(() => {})
        })
        const backend = new QuizzesBackend(quizzesWith({listenForEvents}), new Grants(), new Sessions(), new NoSession())
        const seen: QuizOffer[] = []
        backend.listenForQuizzes((offer) => seen.push(offer))

        await vi.waitFor(() => expect(seen.map((offer) => offer.token)).toEqual(["quiz-1"]))
        backend.close()
    })

    it("opens its stream with the identity in hand", async () => {
        const listenForEvents = quietQuizzes()
        const backend = new QuizzesBackend(quizzesWith({listenForEvents}), new Grants(), new Sessions(), fixedSession("identity-1"))

        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual(["identity-1"]))
        backend.close()
    })

    it("opens it again as the account a later call carried, as a guest signs in", async () => {
        const listenForEvents = quietQuizzes()
        const sessions = new Sessions()
        const backend = new QuizzesBackend(quizzesWith({listenForEvents}), new Grants(), sessions, fixedSession("guest"))
        await vi.waitFor(() => expect(tokensOpenedWith(listenForEvents)).toEqual(["guest"]))

        sessions.carried("guest")
        sessions.carried("signed-in")

        await vi.waitFor(() => expect(listenForEvents).toHaveBeenCalledTimes(2))
        expect(listenForEvents.mock.calls[0][1].signal.aborted).toBe(true)
        backend.close()
    })

    it("opens a quiz with a session token, its clock read from how long is left", async () => {
        const openQuiz = vi.fn().mockResolvedValue(new OpenQuizResponse({
            question: "What is the capital of Estonia?", choices: ["Riga", "Tallinn", "Vilnius"],
            deadlineUnixMs: BigInt(Date.now() + 5_000), answerSeconds: 5,
        }))
        const backend = new QuizzesBackend(quizzesWith({openQuiz}), new Grants(), new Sessions(), fixedSession("session-1"))

        const question = await backend.openQuiz("quiz-1")

        expect(question).toMatchObject({text: "What is the capital of Estonia?", choices: ["Riga", "Tallinn", "Vilnius"], window: 5_000})
        expect(openQuiz).toHaveBeenCalledWith({token: "quiz-1"}, expect.anything())
        expect((openQuiz.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        backend.close()
    })

    it("answers a quiz right or wrong, with no reward in the answer", async () => {
        const answerQuiz = vi.fn().mockResolvedValue(new AnswerQuizResponse({correct: true, correctChoice: 1}))
        const backend = new QuizzesBackend(quizzesWith({answerQuiz}), new Grants(), new Sessions(), fixedSession("session-1"))

        await expect(backend.answerQuiz("quiz-1", 1, "fr")).resolves.toEqual({correct: true, correctChoice: 1})
        expect(answerQuiz).toHaveBeenCalledWith({token: "quiz-1", choice: 1, countryId: "fr"}, expect.anything())
        expect((answerQuiz.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("session-1")
        backend.close()
    })

    it("reports a quiz that is gone as lost", async () => {
        const gone = vi.fn().mockRejectedValue(new ConnectError("gone", Code.NotFound))
        const backend = new QuizzesBackend(quizzesWith({openQuiz: gone, answerQuiz: gone}), new Grants(), new Sessions(),
            fixedSession("session-1"))

        await expect(backend.openQuiz("quiz-1")).rejects.toBeInstanceOf(BonusLostError)
        await expect(backend.answerQuiz("quiz-1", 0, "fr")).rejects.toBeInstanceOf(BonusLostError)
        backend.close()
    })

    it("hands each charge planet granted to the reward listeners", () => {
        const grants = new Grants()
        const backend = new QuizzesBackend(quizzesWith(), grants, new Sessions(), new NoSession())
        const rewards: BonusReward[] = []
        const stop = backend.listenForRewards((reward) => rewards.push(reward))

        grants.grant({kind: "shields", shields: 12}, {kind: "refill"})
        stop()
        grants.grant({kind: "bomb", radius: 4})

        expect(rewards).toEqual([{kind: "shields", shields: 12}, {kind: "refill"}])
        backend.close()
    })
})
