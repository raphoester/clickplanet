import {create} from "zustand"
import {ClickBudget} from "../../backends/clickBudget.ts"

type ClickBudgetStore = {
    budget?: ClickBudget
    hear: (budget: ClickBudget) => void
}

export const useClickBudget = create<ClickBudgetStore>()((set) => ({
    hear: (budget) => set({budget}),
}))
