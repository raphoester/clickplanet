import {useState} from "react"
import {timeOf} from "../components/time.ts"
import {ban, ChatAdmin, DEFAULT_MUTE_SECONDS, LADDER, mute, PlanetAdmin, refusalOf} from "./actions.ts"
import {ConfirmAction} from "./ConfirmAction.tsx"
import {BAN_CHOICES, Choice, labelOf, MUTE_CHOICES} from "./durations.ts"
import {InspectPanel} from "./InspectPanel.tsx"

type Props = {
    chat: ChatAdmin
    planet: PlanetAdmin
    accountId: string
    name: string
}

export function ActPanel({chat, planet, accountId, name}: Props) {
    const [muteFor, setMuteFor] = useState(DEFAULT_MUTE_SECONDS)
    const [banFor, setBanFor] = useState(LADDER)
    const who = name === "" ? "this account" : name

    return (
        <section className="panel" aria-label="Act on the reported account">
            <h2>Act on {who}</h2>

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

            <div className="act">
                <DurationPicker label="Ban for" choices={BAN_CHOICES} value={banFor} onChange={setBanFor}/>
                <ConfirmAction
                    label="Ban"
                    tone="danger"
                    question={`Ban ${who} for ${labelOf(BAN_CHOICES, banFor)}?`}
                    run={() => ban(planet, accountId, banFor)}
                    said={banned => `Banned until ${timeOf(banned.until)}, offence ${banned.offence}.`
                        + (banned.enforced ? "" : " Not enforced: the antibot only watches on this server.")}
                    refusal={err => refusalOf("ban", err)}
                />
            </div>

            <div className="act">
                <InspectPanel planet={planet} accountId={accountId}/>
            </div>
        </section>
    )
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
