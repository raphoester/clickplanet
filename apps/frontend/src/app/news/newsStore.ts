import {create} from "zustand"
import {BombDrop} from "../../backends/backend.ts"
import {Fortified} from "../../domain/fortify.ts"
import {Decided} from "../../domain/flagSubmissions.ts"
import type {ShopReturn} from "../../domain/shopReturn.ts"

export type News =
    | {kind: "bomb", drop: BombDrop, land: string | undefined, id: number}
    | {kind: "fortify", fortified: Fortified, id: number}
    | {kind: "attack", id: number}
    | {kind: "welcome", recruiter: string, flagName: string, id: number}
    // A decision on a flag the player submitted.
    | {kind: "submission", submission: Decided, id: number}
    // Back from Stripe's page.
    | {kind: "shop", back: ShopReturn, id: number}

type NewsStore = {
    // The last news of each kind, the newest last.
    told: readonly News[]
    tellBomb: (drop: BombDrop, land: string | undefined) => void
    tellFortified: (fortified: Fortified) => void
    tellAttack: () => void
    tellWelcome: (recruiter: string, flagName: string) => void
    tellSubmission: (submission: Decided) => void
    tellShop: (back: ShopReturn) => void
    dismiss: (news: News) => void
}

export const newestNews = (news: NewsStore): News | undefined => news.told.at(-1)

export const lastFortified = (news: NewsStore) => news.told.find((told) => told.kind === "fortify")

export const useNews = create<NewsStore>()((set) => {
    let tellings = 0
    const tell = (news: News) => set(({told}) => ({told: [...told.filter((was) => was.kind !== news.kind), news]}))

    return {
        told: [],
        tellBomb: (drop, land) => tell({kind: "bomb", drop, land, id: ++tellings}),
        tellFortified: (fortified) => {
            if (fortified.news) tell({kind: "fortify", fortified, id: ++tellings})
        },
        tellAttack: () => tell({kind: "attack", id: ++tellings}),
        tellWelcome: (recruiter, flagName) => tell({kind: "welcome", recruiter, flagName, id: ++tellings}),
        tellSubmission: (submission) => tell({kind: "submission", submission, id: ++tellings}),
        tellShop: (back) => tell({kind: "shop", back, id: ++tellings}),
        dismiss: (news) => set(({told}) => ({told: told.filter((was) => was !== news)})),
    }
})
