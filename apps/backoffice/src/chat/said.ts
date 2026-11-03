import type {KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"

// What a message said: KeptMessage, KeptQuote and GetMessageResponse share its shape.
export type Body = KeptMessage["body"]

export function textBody(text: string): Body {
    return {case: "text", value: text}
}

// Where only words fit.
export function wordsIn(body: Body): string {
    switch (body.case) {
        case "text":
            return body.value
        case "gif":
            return "a GIF"
        case undefined:
            return ""
    }
}
