import { isDarkTheme } from './app/utils';

(window as any).MonacoEnvironment = {
    getWorkerUrl: () => './assets/monaco-editor/min/vs/base/worker/workerMain.js'
};

const foregroundTextColor = getComputedStyle(document.documentElement)
    .getPropertyValue('--foreground-text')
    .trim()
    .replace('#', '');
const foregroundTextLessContrastColor = getComputedStyle(document.documentElement)
    .getPropertyValue('--foreground-text-less-contrast')
    .trim()
    .replace('#', '');
const background = getComputedStyle(document.documentElement).getPropertyValue('--monaco-background').trim();

(window as any).monaco.editor.defineTheme('soldrJsonTheme', {
    base: isDarkTheme ? 'vs-dark' : 'vs',
    inherit: true,
    rules: [
        {
            token: 'string.key.json',
            foreground: foregroundTextLessContrastColor
        },
        {
            token: 'string.value.json',
            foreground: foregroundTextColor
        },
        { token: 'number', foreground: foregroundTextColor },
        { token: 'keyword.json', foreground: foregroundTextColor }
    ],
    colors: {
        'editor.foreground': foregroundTextColor,
        'editor.background': background
    }
});

(window as any).monaco.editor.defineTheme('soldrFileTheme', {
    base: isDarkTheme ? 'vs-dark' : 'vs',
    inherit: true,
    rules: [{ background }],
    colors: {
        'editor.background': background
    }
});
