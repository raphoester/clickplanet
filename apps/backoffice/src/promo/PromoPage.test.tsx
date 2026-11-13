// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {GetRulesResponse as BonusesRules, RuleSet as BonusesRuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {Rules as OfferRules} from "../gen/grpc/bonuses/v1/rules_pb.ts"
import {RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/planet/v1/rules_pb.ts"
import {CardDraft, SaveCardsRequest} from "../gen/grpc/promo/v1/admin_pb.ts"
import {RuleSet as QuizzesRuleSet} from "../gen/grpc/quizzes/v1/admin_pb.ts"
import {Rules as QuizRules} from "../gen/grpc/quizzes/v1/rules_pb.ts"
import {Put} from "./actions.ts"
import {PromoPage} from "./PromoPage.tsx"

afterEach(cleanup)

function putting() {
    return vi.fn<Put>((_url, file, _type, onSent) => {
        onSent(file.size)
        return Promise.resolve()
    })
}

// Planet keeps urf, which refills three times as fast; the offers keep it, a box every minute at the soonest; the
// quizzes keep it, five seconds to answer; and urf has two cards.
function withUrf(): FakeAdmin {
    const admin = new FakeAdmin()
    admin.rules.sets.push(new RuleSet({name: "urf", rules: Rules.fromJson({clicks: {tempo: 3}})}))
    admin.bonusRules.sets.push(new BonusesRuleSet({name: "urf", rules: OfferRules.fromJson({boxes: {minInterval: "60s"}})}))
    admin.quizRules.sets.push(new QuizzesRuleSet({name: "urf", rules: QuizRules.fromJson({answerWindow: "5s"})}))
    admin.promo.decks.set("urf", [
        new CardDraft({backgroundId: "background-war", headline: "Clicks refill {clicks.tempo}× faster", line: "Every country at once."}),
        new CardDraft({backgroundId: "background-points", headline: "A box every {bonuses.boxes.minInterval}"}),
    ])
    return admin
}

function show(admin: FakeAdmin, put: Put = putting()) {
    return render(<PromoPage promo={admin.promo} planet={admin} bonuses={admin.bonuses} quizzes={admin.quizzes} put={put}/>)
}

async function card(n: number): Promise<HTMLElement> {
    return screen.findByRole("listitem", {name: `Card ${n}`})
}

describe("PromoPage", () => {
    it("lists the background library with each image", async () => {
        show(new FakeAdmin())

        const library = screen.getByRole("region", {name: "Backgrounds"})
        const war = await within(library).findByRole("img", {name: "The war"})
        expect(war.getAttribute("src")).toBe("https://media.test/promo/backgrounds/background-war")
        expect(within(library).getByText("2 backgrounds")).toBeTruthy()
    })

    it("uploads a picked image to the bucket, then keeps it under its name", async () => {
        const admin = new FakeAdmin()
        const put = putting()
        show(admin, put)

        fireEvent.change(screen.getByLabelText("Pick images"), {target: {files: [new File(["png"], "tank.png", {type: "image/png"})]}})
        fireEvent.change(screen.getByLabelText("Name"), {target: {value: "The tank"}})
        fireEvent.click(screen.getByRole("button", {name: "Upload"}))

        expect(await screen.findByText("Uploaded as The tank.")).toBeTruthy()
        expect(admin.askedFor("createBackgroundUpload")).toEqual([{contentType: "image/png", sizeBytes: 3n}])
        expect(put).toHaveBeenCalledWith("https://bucket.test/promo/backgrounds/background-1", expect.any(File), "image/png",
            expect.any(Function))
        expect(admin.askedFor("keepBackground")).toEqual([{backgroundId: "background-1", name: "The tank"}])
        expect(await screen.findByRole("img", {name: "The tank"})).toBeTruthy()
    })

    it("says which cards show a background it may not remove", async () => {
        const admin = new FakeAdmin()
        admin.refuse("removeBackground", Code.FailedPrecondition)
        show(admin)

        const war = (await screen.findByRole("img", {name: "The war"})).closest("li") as HTMLElement
        fireEvent.click(within(war).getByRole("button", {name: "Remove"}))
        fireEvent.click(within(war).getByRole("button", {name: "Yes, remove"}))

        expect((await within(war).findByRole("alert")).textContent).toMatch(/^Change the cards that show it first/)
    })

    it("shows a set's cards as players read them, filled with what the set plays", async () => {
        show(withUrf())

        const first = await card(1)
        expect(within(first).getByLabelText("Card 1 as players read it").textContent)
            .toBe("Clicks refill 3× faster" + "Every country at once.")
        expect(within(await card(2)).getByLabelText("Card 2 as players read it").textContent).toBe("A box every 1 min")
    })

    it("fills a card as it is written, and inserts a value where the cursor was", async () => {
        show(withUrf())
        const first = await card(1)
        const headline = within(first).getByLabelText(/^Headline/) as HTMLInputElement

        fireEvent.change(headline, {target: {value: "Bombs blast "}})
        fireEvent.change(within(first).getByLabelText("Insert a value in Card 1"), {target: {value: "charges.bomb.rings"}})

        expect(headline.value).toBe("Bombs blast {charges.bomb.rings}")
        expect(within(first).getByLabelText("Card 1 as players read it").textContent)
            .toBe("Bombs blast 4" + "Every country at once.")
    })

    it("fills a value of the offers from their set of the same name, default's where it holds none", async () => {
        show(withUrf())
        const first = await card(1)

        fireEvent.change(within(first).getByLabelText(/^Headline/), {target: {value: "Boxes up to "}})
        fireEvent.change(within(first).getByLabelText("Insert a value in Card 1"), {target: {value: "bonuses.boxes.maxInterval"}})

        expect(within(first).getByLabelText("Card 1 as players read it").textContent)
            .toBe("Boxes up to 8 min" + "Every country at once.")
        expect(within(await card(2)).getByLabelText("Card 2 as players read it").textContent).toBe("A box every 1 min")
    })

    it("fills a value of the quizzes from their set of the same name, default's where it holds none", async () => {
        show(withUrf())
        const first = await card(1)

        fireEvent.change(within(first).getByLabelText(/^Headline/), {target: {value: "Banner "}})
        fireEvent.change(within(first).getByLabelText("Insert a value in Card 1"), {target: {value: "quizzes.offerTtl"}})
        fireEvent.change(within(first).getByLabelText(/^Line/), {target: {value: "Answer within {quizzes.answerWindow}"}})

        expect(within(first).getByLabelText("Card 1 as players read it").textContent)
            .toBe("Banner 25 s" + "Answer within 5 s")
    })

    it("says a value of the offers a set plays nothing of, when the offers do not keep it", async () => {
        const admin = withUrf()
        admin.bonusRules = new BonusesRules({sets: [admin.bonusRules.sets[0]], playing: "default"})
        show(admin)
        const first = await card(1)

        fireEvent.change(within(first).getByLabelText(/^Headline/), {target: {value: "A box every {bonuses.boxes.maxInterval}"}})

        expect(within(first).getByRole("alert").textContent).toContain("The set plays no value at {bonuses.boxes.maxInterval}.")
        expect(within(await card(2)).getByRole("alert").textContent).toContain("The set plays no value at {bonuses.boxes.minInterval}.")
    })

    it("fills a value of the battle as in the example battle, and says so", async () => {
        show(withUrf())
        const first = await card(1)
        expect(within(first).queryByText("Example: a ×3 battle")).toBeNull()

        fireEvent.change(within(first).getByLabelText(/^Line/), {target: {value: "1st place wins "}})
        fireEvent.change(within(first).getByLabelText("Insert a value in Card 1"), {target: {value: "battle.points.first"}})

        expect((within(first).getByLabelText(/^Line/) as HTMLInputElement).value).toBe("1st place wins {battle.points.first}")
        expect(within(first).getByLabelText("Card 1 as players read it").textContent)
            .toBe("Clicks refill 3× faster" + "1st place wins 75")
        expect(within(first).getByText("Example: a ×3 battle")).toBeTruthy()
        expect(within(first).getByRole("option", {name: "battle.points.first (×3: 75)"})).toBeTruthy()
    })

    it("saves the set's cards, in order, once asked", async () => {
        const admin = withUrf()
        show(admin)
        const second = await card(2)

        fireEvent.click(within(second).getByRole("button", {name: "Move Card 2 up"}))
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        expect(screen.getByText("Save the 2 cards of urf?")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        expect(await screen.findByText("Saved. Players read them within a minute.")).toBeTruthy()
        const [saved] = admin.askedFor("saveCards") as SaveCardsRequest[]
        expect(saved.ruleSet).toBe("urf")
        expect(saved.cards.map(each => each.headline)).toEqual(["A box every {bonuses.boxes.minInterval}", "Clicks refill {clicks.tempo}× faster"])
    })

    it("will not save a card the server would refuse, and says why", async () => {
        show(withUrf())
        const first = await card(1)

        fireEvent.change(within(first).getByLabelText(/^Headline/), {target: {value: "Clicks refill {clicks.speed}× faster"}})

        expect(within(first).getByRole("alert").textContent).toContain("{clicks.speed} names no number or duration of the rules or the battle.")
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("removes every card of a set, which then is not teased", async () => {
        const admin = withUrf()
        show(admin)

        for (const n of [2, 1]) fireEvent.click(within(await card(n)).getByRole("button", {name: "Remove"}))
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        expect(screen.getByText("Remove every card of urf? Its battles are not teased then.")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        await waitFor(() => expect(admin.askedFor("saveCards")).toHaveLength(1))
        expect((admin.askedFor("saveCards")[0] as SaveCardsRequest).cards).toEqual([])
    })

    it("quotes the server when it refuses the cards", async () => {
        const admin = withUrf()
        admin.refuse("saveCards", Code.FailedPrecondition)
        show(admin)
        const first = await card(1)

        fireEvent.change(within(first).getByLabelText(/^Line/), {target: {value: "Every flag at once."}})
        fireEvent.click(screen.getByRole("button", {name: "Save"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        expect((await screen.findByText(/^The server refused it/)).textContent).toMatch(/^The server refused it: /)
    })
})
