import {createContext, useContext} from "react"
import {NO_FILTER, WordFilter} from "../../domain/wordFilter.ts"

export const WordFilterContext = createContext<WordFilter>(NO_FILTER)

export function useWordFilter(): WordFilter {
    return useContext(WordFilterContext)
}
