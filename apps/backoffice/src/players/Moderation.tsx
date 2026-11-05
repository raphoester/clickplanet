import {useState} from "react"
import {AntibotAdmin, ChatAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {timeOf} from "../components/time.ts"
import {MuteKind} from "../gen/grpc/chat/v1/chat_pb.ts"
import {
    ban, bannedWords, DEFAULT_MUTE_SECONDS, LADDER, mute, MUTE_KINDS, muteWordsOf, refusalOf, unban, Unbanned, unmute,
} from "./actions.ts"
import {Caller} from "./caller.ts"
import {BAN_CHOICES, Choice, labelOf, MUTE_CHOICES} from "./durations.ts"

type MuteProps = {chat: ChatAdmin, accountId: string, who: string, onDone?: () => void}

// A mute is an account's alone: the chat mutes no address.
export function MuteAction({chat, accountId, who, onDone}: MuteProps) {
    const [kind, setKind] = useState(MuteKind.CHAT)
    const [muteFor, setMuteFor] = useState(DEFAULT_MUTE_SECONDS)
    const words = muteWordsOf(kind)

    return (
        <div className="act">
            <MuteKindPicker value={kind} onChange={setKind}/>
            <DurationPicker label="Mute for" choices={MUTE_CHOICES} value={muteFor} onChange={setMuteFor}/>
            <ConfirmAction
                label="Mute"
                question={`Mute ${words.of(who)} in the chat for ${labelOf(MUTE_CHOICES, muteFor)}?`}
                run={() => mute(chat, accountId, muteFor, kind)}
                said={muted => `${words.muted} until ${timeOf(muted.until)}.`}
                refusal={err => refusalOf("mute", err)}
                onDone={onDone}
            />
        </div>
    )
}

// Lifts the mutes of one kind: a mute of the other kind stays.
export function UnmuteAction({chat, accountId, who, onDone}: MuteProps) {
    const [kind, setKind] = useState(MuteKind.CHAT)
    const words = muteWordsOf(kind)

    return (
        <div className="act">
            <MuteKindPicker value={kind} onChange={setKind}/>
            <ConfirmAction
                label="Unmute"
                question={`Unmute ${words.of(who)} in the chat now?`}
                run={() => unmute(chat, accountId, kind)}
                said={unmuted => `${words.unmuted}. The mute was until ${timeOf(unmuted.until)}.`}
                refusal={err => refusalOf("unmute", err)}
                onDone={onDone}
            />
        </div>
    )
}

export function MuteKindPicker({value, onChange}: {value: MuteKind, onChange: (kind: MuteKind) => void}) {
    return (
        <label className="picker">
            <span className="muted">Mute what</span>
            <select value={value} onChange={event => onChange(Number(event.target.value) as MuteKind)}>
                {MUTE_KINDS.map(each => <option key={each.kind} value={each.kind}>{each.label}</option>)}
            </select>
        </label>
    )
}

type BanProps = {antibot: AntibotAdmin, caller: Caller, who: string, onDone?: () => void}

export function BanAction({antibot, caller, who, onDone}: BanProps) {
    const [banFor, setBanFor] = useState(LADDER)

    return (
        <div className="act">
            <DurationPicker label="Ban for" choices={BAN_CHOICES} value={banFor} onChange={setBanFor}/>
            <ConfirmAction
                label="Ban"
                tone="danger"
                question={`Ban ${who} for ${labelOf(BAN_CHOICES, banFor)}?`}
                run={() => ban(antibot, caller, banFor)}
                said={bannedWords}
                refusal={err => refusalOf("ban", err)}
                onDone={onDone}
            />
        </div>
    )
}

export function UnbanAction({antibot, caller, who, onDone}: BanProps) {
    return (
        <div className="act">
            <p className="hint">For a ban that was a mistake. It forgets the latest offence, ended or running.</p>
            <ConfirmAction
                label="Unban"
                question={`Lift the latest ban of ${who} and forget its offence?`}
                run={() => unban(antibot, caller)}
                said={unbannedWords}
                refusal={err => refusalOf("unban", err)}
                onDone={onDone}
            />
        </div>
    )
}

function unbannedWords(lifted: Unbanned): string {
    const until = lifted.until === undefined ? "" : `: its ban was until ${timeOf(lifted.until)}`
    const pardoned = lifted.pardonedUntil === undefined ? "" : ` Pardoned until ${timeOf(lifted.pardonedUntil)}.`
    return `Lifted offence ${lifted.offence}${until}.${pardoned}`
}

type PickerProps = {label: string, choices: Choice[], value: number, onChange: (seconds: number) => void}

function DurationPicker({label, choices, value, onChange}: PickerProps) {
    return (
        <label className="picker">
            <span className="muted">{label}</span>
            <select value={value} onChange={event => onChange(Number(event.target.value))}>
                {choices.map(choice => <option key={choice.seconds} value={choice.seconds}>{choice.label}</option>)}
            </select>
        </label>
    )
}
