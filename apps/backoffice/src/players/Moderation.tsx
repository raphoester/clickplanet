import {useState} from "react"
import {AntibotAdmin, ChatAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {timeOf} from "../components/time.ts"
import {ban, Banned, DEFAULT_MUTE_SECONDS, LADDER, mute, refusalOf, unban, unbannedWords} from "./actions.ts"
import {Caller} from "./caller.ts"
import {BAN_CHOICES, Choice, labelOf, MUTE_CHOICES} from "./durations.ts"

type MuteProps = {chat: ChatAdmin, accountId: string, who: string}

// A mute is an account's alone: the chat mutes no address.
export function MuteAction({chat, accountId, who}: MuteProps) {
    const [muteFor, setMuteFor] = useState(DEFAULT_MUTE_SECONDS)

    return (
        <div className="act">
            <DurationPicker label="Mute for" choices={MUTE_CHOICES} value={muteFor} onChange={setMuteFor}/>
            <ConfirmAction
                label="Mute"
                question={`Mute ${who} in the chat for ${labelOf(MUTE_CHOICES, muteFor)}?`}
                run={() => mute(chat, accountId, muteFor)}
                said={muted => `Muted until ${timeOf(muted.until)}.`}
                refusal={err => refusalOf("mute", err)}
            />
        </div>
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

function bannedWords(banned: Banned): string {
    return `Banned until ${timeOf(banned.until)}, offence ${banned.offence}.`
        + (banned.enforced ? "" : " Not enforced: the antibot only watches on this server.")
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
