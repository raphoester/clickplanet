import {UploadOptions} from "./support.ts"

export type PutFile = (url: string, file: Blob, options?: UploadOptions) => Promise<void>

// XMLHttpRequest and not fetch: fetch says nothing while a body goes up.
export const putFile: PutFile = (url, file, {onProgress, signal} = {}) => new Promise<void>((resolve, reject) => {
    if (signal?.aborted) {
        reject(new Error("the upload was called off"))
        return
    }

    const request = new XMLHttpRequest()
    request.open("PUT", url)
    // The link is signed for this type: with another one, or none, the bucket refuses the file.
    request.setRequestHeader("Content-Type", file.type)
    request.upload.onprogress = (event) => {
        if (event.lengthComputable && event.total > 0) onProgress?.(event.loaded / event.total)
    }
    request.onload = () => {
        if (request.status >= 200 && request.status < 300) resolve()
        else reject(new Error(`the upload was answered ${request.status}`))
    }
    request.onerror = () => reject(new Error("the upload could not be sent"))
    request.onabort = () => reject(new Error("the upload was called off"))
    signal?.addEventListener("abort", () => request.abort(), {once: true})
    request.send(file)
})
