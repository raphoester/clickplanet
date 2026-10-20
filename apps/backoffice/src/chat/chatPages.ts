import {ChatAdmin} from "../clients.ts"
import {KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"

// An account a report names, so the conversation marks what it said.
export type Party = {accountId: string, name: string}
export type Parties = {reporter: Party, reported: Party}

// A page is full when it holds as many as asked: there may be more past it.
export type Page = {messages: KeptMessage[], more: boolean}
type Ask = {beforeMessageId?: string, afterMessageId?: string, accountId?: string}

export async function pageOf(chat: ChatAdmin, ask: Ask, limit: number, signal?: AbortSignal): Promise<Page> {
    const answer = await chat.listMessages({...ask, limit}, {signal})
    return {messages: answer.messages, more: answer.messages.length === limit}
}
