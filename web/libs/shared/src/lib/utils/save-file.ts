export interface FileInfo {
    url: string;
    name: string;
}

export function saveFile(file: FileInfo) {
    const anchor = document.createElement('a');
    anchor.download = file.name;
    anchor.href = file.url;
    anchor.click();
}
