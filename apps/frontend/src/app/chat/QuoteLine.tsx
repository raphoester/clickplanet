import {ChatQuote} from "../../backends/chat.ts";
import {ReplyIcon} from "../components/icons.tsx";
import {truncate} from "../truncate.ts";

export const QUOTE_MAX_LENGTH = 100

const QUOTE_AUTHOR_MAX_LENGTH = 16

export default function QuoteLine({quote, lead}: {quote: ChatQuote, lead: string}) {
    return <>
        <ReplyIcon size={12}/>
        <span className="sr-only">{lead}</span>
        {" "}
        <span className="chat-quote-author">{truncate(quote.authorName, QUOTE_AUTHOR_MAX_LENGTH)}</span>
        {" "}
        <span className="chat-quote-text">{truncate(quote.text, QUOTE_MAX_LENGTH)}</span>
    </>
}
