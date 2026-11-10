import {expect, it} from "vitest"
import * as served from "../api.ts"
import * as demo from "./api.ts"

it("answers every name the page imports from the real api", () => {
    expect(Object.keys(demo).sort()).toEqual(Object.keys(served).sort())
})
