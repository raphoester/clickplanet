export function download(file: File) {
    const url = URL.createObjectURL(file)

    const link = document.createElement("a")
    link.href = url
    link.download = file.name
    link.click()

    // Revoking in the same tick cancels the download in some browsers.
    setTimeout(() => URL.revokeObjectURL(url), 0)
}
