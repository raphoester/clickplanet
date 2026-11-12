import {ChatQuote} from "../../backends/chat.ts";
import {ReplyIcon} from "../components/icons.tsx";
import {truncate} from "../truncate.ts";
import {useWordFilter} from "../crazygames/wordFilter.ts";
import {nameOf} from "../nameplates/nameplate.ts";
import {useNameplate} from "../nameplates/nameplatesStore.ts";
import {t} from "../../i18n/page.ts";
import {rich} from "../../i18n/rich.tsx";
import {Key} from "../../i18n/translator.ts";
import {wordsOf} from "./wordsOf.ts";

export const QUOTE_MAX_LENGTH = 100

const QUOTE_AUTHOR_MAX_LENGTH = 16

export type QuoteLead = "inReplyTo" | "replyingTo"

const LEADS: Record<QuoteLead, Key> = {
    inReplyTo: "chat.quote.inReplyTo",
    replyingTo: "chat.quote.replyingTo",
}

export default function QuoteLine({quote, lead}: {quote: ChatQuote, lead: QuoteLead}) {
    const words = useWordFilter()
    const author = nameOf(useNameplate(quote.authorAccountId))

    return <>
        <ReplyIcon size={12}/>
        {rich(LEADS[lead], {
            hidden: (children) => <span className="sr-only">{children}</span>,
            author: <span className="chat-quote-author">{truncate(words.masked(author), QUOTE_AUTHOR_MAX_LENGTH)}</span>,
            text: quote.removed
                ? <span className="chat-quote-text chat-removed">{t("chat.removed")}</span>
                : <span className="chat-quote-text">{truncate(words.masked(wordsOf(quote)), QUOTE_MAX_LENGTH)}</span>,
        })}
    </>
}
