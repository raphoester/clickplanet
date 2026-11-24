import {FormEvent, useEffect, useId, useRef, useState} from "react"
import {FlagSubmitter, SubmissionRefusal, SubmissionRefusedError} from "../../backends/flags.ts"
import {
    flagNameLength,
    flagNameOf,
    isValidFlagName,
    MAX_FLAG_NAME_LENGTH,
    MAX_PICTURE_BYTES,
    PICTURE_TYPES,
    PictureRefusal,
    pictureFileRefusal,
    pictureShapeRefusal,
} from "../../domain/flagSubmissions.ts"
import {t, translator} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"
import Modal from "../components/Modal.tsx"
import {PictureIcon, PinIcon} from "../components/icons.tsx"
import {useScreen} from "../hud/screenStore.ts"
import {sizeLine} from "../support/attachmentLines.ts"
import {useFlags} from "./flagsStore.ts"
import {Draft, useFlagForm} from "./flagFormStore.ts"
import {useFlagSubmissions} from "./submissionsStore.ts"
import {coordinatesOf} from "./coordinates.ts"
import "./FlagSubmission.css"

export type FlagSubmissionFormProps = {
    submitter: FlagSubmitter
}

type Failure = SubmissionRefusal | "upload-failed" | "failed"

type State =
    | {kind: "writing", failure?: Failure}
    | {kind: "uploading", fraction: number}
    | {kind: "sending"}

const FAILURES: Record<Failure, Key> = {
    "no-session": "support.failure.noSession",
    "guest": "flagSubmission.signIn",
    "not-yours": "flagSubmission.refusal.notYours",
    "waiting": "flagSubmission.refusal.waiting",
    "invalid": "flagSubmission.refusal.invalid",
    "off": "flagSubmission.refusal.off",
    "too-many": "flagSubmission.refusal.tooMany",
    "upload-failed": "flagSubmission.refusal.uploadFailed",
    "failed": "support.failure.failed",
}

type PictureProblem = {refusal: PictureRefusal, file: string}

function pictureLine({refusal, file}: PictureProblem): string {
    switch (refusal) {
        case "type":
            return t("flagSubmission.picture.type", {file})
        case "size":
            return t("flagSubmission.picture.size", {file, size: sizeLine(MAX_PICTURE_BYTES)})
        case "narrow":
            return t("flagSubmission.picture.narrow")
        case "wide":
            return t("flagSubmission.picture.wide")
        case "small":
            return t("flagSubmission.picture.small")
    }
}

// A change sends a picture too: the server reviews every flag as the players will see it.
function readyToSend(draft: Draft, problem: PictureProblem | undefined): boolean {
    return isValidFlagName(draft.name) && draft.pinned !== undefined && problem === undefined && draft.file !== undefined
}

export default function FlagSubmissionForm({submitter}: FlagSubmissionFormProps) {
    const draft = useFlagForm((form) => form.draft)
    const edit = useFlagForm((form) => form.edit)
    const aim = useFlagForm((form) => form.aim)
    const close = useFlagForm((form) => form.close)
    const sent = useFlagForm((form) => form.sent)
    const catalog = useFlags((store) => store.catalog)
    const [state, setState] = useState<State>({kind: "writing"})
    const [problem, setProblem] = useState<PictureProblem>()
    const closing = useRef<AbortController>()
    const input = useRef<HTMLInputElement>(null)
    const formId = useId()
    const nameId = useId()
    const counterId = useId()
    const preview = usePreview(draft.file)

    // Made here and not once per component: StrictMode runs the cleanup once before the form is ever used.
    useEffect(() => {
        const controller = new AbortController()
        closing.current = controller
        return () => controller.abort()
    }, [])

    const change = draft.flag !== undefined
    const currentPicture = draft.flag === undefined ? undefined : catalog.pictureOf(draft.flag)
    const busy = state.kind !== "writing"
    const canSend = !busy && readyToSend(draft, problem)
    const length = flagNameLength(draft.name)
    const failure = state.kind === "writing" ? state.failure : undefined

    const pick = (file: File | undefined) => {
        if (!file) return
        const refusal = pictureFileRefusal(file)
        if (refusal) {
            setProblem({refusal, file: file.name})
            return
        }
        setProblem(undefined)
        edit({file})
    }

    const aimPin = () => {
        useScreen.getState().closeSheet()
        aim()
    }

    const submit = async (event: FormEvent) => {
        event.preventDefault()
        const pinned = draft.pinned
        if (!canSend || pinned === undefined) return

        const signal = closing.current?.signal
        let uploadId = ""
        try {
            if (draft.file) {
                setState({kind: "uploading", fraction: 0})
                try {
                    uploadId = await submitter.uploadPicture(draft.file, {
                        signal,
                        onProgress: (fraction) => setState({kind: "uploading", fraction}),
                    })
                } catch (e) {
                    if (signal?.aborted) return
                    if (e instanceof SubmissionRefusedError) throw e
                    console.error("could not upload the flag's picture", e)
                    setState({kind: "writing", failure: "upload-failed"})
                    return
                }
            }
            if (signal?.aborted) return
            setState({kind: "sending"})
            const submission = await submitter.submitFlag({name: flagNameOf(draft.name), uploadId, pin: pinned.pin, flag: draft.flag})
            if (signal?.aborted) return
            useFlagSubmissions.getState().submitted(submission)
            sent()
        } catch (e) {
            if (signal?.aborted) return
            if (!(e instanceof SubmissionRefusedError)) console.error("could not send the flag", e)
            setState({kind: "writing", failure: e instanceof SubmissionRefusedError ? e.refusal : "failed"})
        }
    }

    const shown = preview ?? currentPicture?.url
    const sendLabel = state.kind === "uploading"
        ? t("flagSubmission.uploading", {percent: translator().number(state.fraction, {style: "percent"})})
        : state.kind === "sending" ? t("flagSubmission.sending") : t("common.send")

    return <Modal title={change ? t("flagSubmission.changeTitle") : t("flagSubmission.add")}
                  stayOnBackdropClick={busy || draft.name !== "" || draft.file !== undefined || draft.pinned !== undefined}
                  footer={<button type="submit" form={formId} className="button button-action support-wide" disabled={!canSend}>
                      {sendLabel}
                  </button>}
                  onClose={close}>
        <form id={formId} className="support-form flag-form" onSubmit={submit} aria-busy={busy}>
            <div className="flag-form-label-row">
                <label className="menu-label" htmlFor={nameId}>{t("flagSubmission.name")}</label>
                <span id={counterId}
                      className={length > MAX_FLAG_NAME_LENGTH ? "menu-label flag-form-counter flag-form-counter--over" : "menu-label flag-form-counter"}>
                    {`${translator().number(length)}/${translator().number(MAX_FLAG_NAME_LENGTH)}`}
                </span>
            </div>
            <input id={nameId}
                   className="field support-wide"
                   type="text"
                   autoComplete="off"
                   maxLength={MAX_FLAG_NAME_LENGTH * 2}
                   value={draft.name}
                   readOnly={busy}
                   aria-describedby={counterId}
                   onChange={(e) => edit({name: e.target.value})}/>

            <span className="menu-label">{t("flagSubmission.picture")}</span>
            <div className="flag-form-picture">
                {shown && <span className="flag-form-preview">
                    <img src={shown}
                         alt=""
                         onLoad={(e) => {
                             if (!preview || !draft.file) return
                             const refusal = pictureShapeRefusal(e.currentTarget.naturalWidth, e.currentTarget.naturalHeight)
                             setProblem(refusal && {refusal, file: draft.file.name})
                         }}/>
                    {state.kind === "uploading" && <span className="flag-form-track"
                                                         role="progressbar"
                                                         aria-label={t("flagSubmission.uploadingPicture")}
                                                         aria-valuemin={0}
                                                         aria-valuemax={100}
                                                         aria-valuenow={Math.round(state.fraction * 100)}>
                        <span style={{width: `${Math.round(state.fraction * 100)}%`}}/>
                    </span>}
                </span>}
                <input ref={input}
                       type="file"
                       hidden
                       accept={PICTURE_TYPES.join(",")}
                       onChange={(e) => {
                           pick(e.target.files?.[0])
                           e.target.value = ""
                       }}/>
                <button type="button"
                        className="button button-mini button-secondary"
                        disabled={busy}
                        onClick={() => input.current?.click()}>
                    <PictureIcon/>
                    {shown ? t("flagSubmission.otherPicture") : t("flagSubmission.pickPicture")}
                </button>
            </div>
            {problem && <p className="support-notice" role="alert">{pictureLine(problem)}</p>}

            <span className="menu-label">{t("flagSubmission.place")}</span>
            {draft.pinned
                ? <div className="panel-box flag-form-pin">
                    <PinIcon/>
                    <span className="flag-form-pin-where">
                        {draft.pinned.place && <span className="flag-form-pin-place">{draft.pinned.place}</span>}
                        <span className="flag-form-pin-coordinates">{coordinatesOf(draft.pinned.pin)}</span>
                    </span>
                    <button type="button" className="button button-mini button-secondary" disabled={busy} onClick={aimPin}>
                        {t("flagSubmission.movePin")}
                    </button>
                </div>
                : <div>
                    <button type="button" className="button button-mini button-secondary" disabled={busy} onClick={aimPin}>
                        <PinIcon/>
                        {t("flagSubmission.pinIt")}
                    </button>
                </div>}

            {failure && <p className="support-notice" role="alert">{t(FAILURES[failure])}</p>}
        </form>
    </Modal>
}

function usePreview(file: File | undefined): string | undefined {
    const [url, setUrl] = useState<string>()

    useEffect(() => {
        if (!file) {
            setUrl(undefined)
            return
        }
        const made = URL.createObjectURL(file)
        setUrl(made)
        return () => URL.revokeObjectURL(made)
    }, [file])

    return url
}
