import type {Submission} from "../../domain/flagSubmissions.ts"

// A person looks at a rejection again: the first review may be Claude's.
export function appealOf({name}: Submission): string {
    return `mailto:support@clickplanet.lol?subject=${encodeURIComponent(`Look again at my flag ${name}`)}`
}
