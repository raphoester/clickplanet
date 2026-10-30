import {ChatQuote} from "../../backends/chat.ts";
import {ReplyIcon} from "../components/icons.tsx";
import {truncate} from "../truncate.ts";
import {useWordFilter} from "../crazygames/wordFilter.ts";

export const QUOTE_MAX_LENGTH = 100

export const REMOVED_TEXT = "This message has been deleted"

const QUOTE_AUTHOR_MAX_LENGTH = 16

export default function QuoteLine({quote, lead}: {quote: ChatQuote, lead: string}) {
    const words = useWordFilter()

    return <>
        <ReplyIcon size={12}/>
        <span className="sr-only">{lead}</span>
        {" "}
        <span className="chat-quote-author">{truncate(words.masked(quote.authorName), QUOTE_AUTHOR_MAX_LENGTH)}</span>
        {" "}
        {quote.removed
            ? <span className="chat-quote-text chat-removed">{REMOVED_TEXT}</span>
            : <span className="chat-quote-text">{truncate(words.masked(quote.text), QUOTE_MAX_LENGTH)}</span>}
    </>
}
