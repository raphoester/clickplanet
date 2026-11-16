import {create} from "zustand"
import {Gates, NO_GATES} from "../../domain/bonus.ts"

export const useGates = create<{gates: Gates}>()(() => ({gates: NO_GATES}))
