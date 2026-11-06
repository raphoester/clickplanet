// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {SaveRulesRequest} from "../gen/grpc/missions/v1/admin_pb.ts"
import {MissionRulesPanel} from "./MissionRulesPanel.tsx"

afterEach(cleanup)

const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), {target: {value}})

const saveButton = () => screen.getByRole("button", {name: "Save"}) as HTMLButtonElement

describe("the mission rules page", () => {
    it("shows the numbers missions draws with", async () => {
        render(<MissionRulesPanel missions={new FakeAdmin().missions}/>)

        expect(await screen.findByLabelText("Goal, % of all the flag did the day before")).toHaveProperty("value", "30")
        expect(screen.getByLabelText("Least goal, tiles or shields")).toHaveProperty("value", "250")
        expect(screen.getByLabelText("Points for a flag holding no tile")).toHaveProperty("value", "10")
        expect(screen.getByLabelText("Points for the biggest flag")).toHaveProperty("value", "3")
        expect(saveButton().disabled).toBe(true)
    })

    it("asks first, says what changes, then saves every number", async () => {
        const admin = new FakeAdmin()
        render(<MissionRulesPanel missions={admin.missions}/>)
        await screen.findByLabelText("Least goal, tiles or shields")

        type("Least goal, tiles or shields", "300")
        fireEvent.click(saveButton())

        expect(screen.getByRole("list", {name: "What changes"}).textContent).toContain("Least goal, tiles or shields 250 → 300")
        expect(admin.askedFor("saveMissionRules")).toEqual([])
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        expect(await screen.findByRole("status")).toHaveProperty("textContent", "Saved.")
        const [saved] = admin.askedFor("saveMissionRules") as SaveRulesRequest[]
        expect(saved.rules?.toJson()).toEqual({
            flagMissions: {goalPercent: 30, leastGoal: 300, fewestPoints: 3, mostPoints: 10},
        })
        await waitFor(() => expect(screen.getByText("No change yet")).toBeTruthy())
        expect(screen.getByRole("status").textContent).toBe("Saved.")
        expect(admin.askedFor("getMissionRules")).toHaveLength(1)
    })

    it("keeps a number that is not whole from the server", async () => {
        render(<MissionRulesPanel missions={new FakeAdmin().missions}/>)
        await screen.findByLabelText("Points for the biggest flag")

        type("Points for the biggest flag", "")

        expect(screen.getByRole("alert").textContent).toBe("Points for the biggest flag: a whole number.")
        expect(saveButton().disabled).toBe(true)
    })

    it("quotes the server when it refuses the numbers", async () => {
        const admin = new FakeAdmin()
        admin.refuse("saveMissionRules", Code.InvalidArgument)
        render(<MissionRulesPanel missions={admin.missions}/>)
        await screen.findByLabelText("Points for the biggest flag")

        type("Points for the biggest flag", "11")
        fireEvent.click(saveButton())
        fireEvent.click(screen.getByRole("button", {name: "Yes, save"}))

        expect((await screen.findByRole("alert")).textContent).toContain("The server refused it")
    })
})
