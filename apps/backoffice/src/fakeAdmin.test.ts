import {Code, ConnectError} from "@connectrpc/connect"
import {describe, expect, it} from "vitest"
import {FakeAdmin, nameRefusalOf} from "./fakeAdmin.ts"

describe("the fake's flag names", () => {
    it("refuses the name of a flag the game ships with, whatever its case, as the server does", () => {
        expect(nameRefusalOf("germany", [])).toBe("a flag the game ships with is named Germany")
        expect(nameRefusalOf("FRANCE", [])).toBe("a flag the game ships with is named France")
        expect(nameRefusalOf("Bayern", [])).toBeUndefined()
    })
})

describe("the fake's approval", () => {
    it("refuses a capital off the globe as the server does", async () => {
        const {flags} = new FakeAdmin()
        const refused = await flags.approveSubmission({submissionId: "sub-catalonia", pin: {latitude: 91, longitude: 2}}).catch(err => err)
        expect(refused).toBeInstanceOf(ConnectError)
        expect((refused as ConnectError).code).toBe(Code.InvalidArgument)
    })
})
