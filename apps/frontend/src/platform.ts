import {onCrazyGames} from "./domain/crazyGames.ts"

export const ON_CRAZYGAMES = typeof window !== "undefined" && onCrazyGames(window.location.hostname, import.meta.env.VITE_CRAZYGAMES)
