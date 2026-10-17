import {RefObject, useCallback, useEffect, useRef, useState} from "react";
import {ChatMessage} from "../../backends/chat.ts";
import Bubble, {BUBBLE_MS} from "../components/Bubble.tsx";
import {ReportIcon} from "../components/icons.tsx";
import {useEscape} from "../components/useDialog.ts";
import {SEND_FAILURES, SendFailure, sendFailureOf} from "../support/sendFailure.ts";

export const ARMED_MS = 4000

type Step = "idle" | "armed" | "sending"

export type ReportMessageButtonProps = {
    message: ChatMessage
    reported: boolean
    onReport: (message: ChatMessage) => Promise<void>
}

export default function ReportMessageButton({message, reported, onReport}: ReportMessageButtonProps) {
    const button = useRef<HTMLButtonElement>(null)
    const [step, setStep] = useState<Step>("idle")
    const [failure, setFailure] = useState<SendFailure>()
    const disarm = useCallback(() => setStep("idle"), [])

    useEffect(() => {
        if (!failure) return
        const timer = window.setTimeout(() => setFailure(undefined), BUBBLE_MS)
        return () => window.clearTimeout(timer)
    }, [failure])

    if (reported) {
        return <span className="chat-report-slot">
            <span className="chat-report-add chat-report-done" role="img" aria-label="Reported" title="Reported">
                <ReportIcon/>
            </span>
        </span>
    }

    const press = async () => {
        if (step === "idle") {
            setFailure(undefined)
            setStep("armed")
            return
        }
        setStep("sending")
        try {
            await onReport(message)
        } catch (e) {
            setFailure(sendFailureOf(e))
        }
        setStep("idle")
    }

    const armed = step !== "idle"

    return <span className="chat-report-slot">
        <button type="button"
                ref={button}
                className={armed ? "chat-report-add chat-report-armed" : "chat-report-add"}
                aria-label={armed
                    ? `Confirm the report of this message from ${message.authorName}`
                    : `Report this message from ${message.authorName}`}
                title="Report"
                aria-busy={step === "sending"}
                disabled={step === "sending"}
                onClick={press}>
            {armed ? "Report" : <ReportIcon/>}
        </button>
        {step === "armed" && <Disarm button={button} onDisarm={disarm}/>}
        {failure && <Bubble anchor={button}>{SEND_FAILURES[failure]}</Bubble>}
    </span>
}

function Disarm({button, onDisarm}: {button: RefObject<HTMLButtonElement>, onDisarm: () => void}) {
    useEscape(onDisarm)

    useEffect(() => {
        const elsewhere = (event: PointerEvent) => {
            if (!button.current?.contains(event.target as Node)) onDisarm()
        }
        const timer = window.setTimeout(onDisarm, ARMED_MS)
        document.addEventListener("pointerdown", elsewhere)
        return () => {
            window.clearTimeout(timer)
            document.removeEventListener("pointerdown", elsewhere)
        }
    }, [button, onDisarm])

    return null
}
