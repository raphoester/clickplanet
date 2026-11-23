import {SupportAdmin} from "../clients.ts"
import {Note, NoteAnswer} from "../gen/grpc/support/v1/admin_pb.ts"
import {Meanings, wordsOf} from "../refusal.ts"

export const MAX_ANSWER_LENGTH = 5000

// The note as the operator left it, and what the page says of it.
export type Acted = {note: Note, said: string}

export async function answerNote(support: SupportAdmin, note: Note, text: string): Promise<Acted> {
    const answer = text.trim()
    const {sentAt} = await support.answerNote({noteId: note.id, text: answer})
    return {
        note: new Note({...note, answers: [...note.answers, new NoteAnswer({text: answer, sentAt})]}),
        said: `Sent to ${note.replyAddress}.`,
    }
}

export async function closeNote(support: SupportAdmin, note: Note): Promise<Acted> {
    const {closedAt} = await support.closeNote({noteId: note.id})
    return {note: new Note({...note, closedAt}), said: "Closed with no answer."}
}

export type Action = "answer" | "close"

const NOTE_GONE = "This note is no longer kept."

const MEANINGS: Record<Action, Meanings> = {
    answer: {
        notFound: NOTE_GONE,
        failedPrecondition: "This note has no address to answer at.",
        unavailable: "The mail did not go. Send it again.",
    },
    close: {notFound: NOTE_GONE},
}

// What a refusal means for the action it refused, in the operator's words.
export function refusalOf(action: Action, err: unknown): string {
    return wordsOf(err, MEANINGS[action])
}
