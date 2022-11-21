export const CHANGELOG_DATE_FORMAT_RU = 'dd.MM.yyyy';
export const CHANGELOG_DATE_FORMAT_EN = 'MM-dd-yyyy';

export const supportedLanguages: Readonly<Record<string, string>> = {
    bat: 'bat',
    css: 'css',
    css3: 'css',
    html: 'html',
    ini: 'ini',
    js: 'javascript',
    json: 'json',
    less: 'less',
    lua: 'lua',
    php: 'php',
    pl: 'perl',
    ps1: 'powershell',
    psd1: 'powershell',
    psm1: 'powershell',
    py: 'python',
    rb: 'ruby',
    scss: 'scss',
    sh: 'shell',
    sql: 'sql',
    tcl: 'tcl',
    ts: 'typescript',
    txt: 'plain',
    vb: 'vb',
    vba: 'vb',
    vbs: 'vb',
    vue: 'html',
    xml: 'xml',
    yaml: 'yaml',
    yml: 'yaml'
};

export const tagNameMask = /^[\w\d_\-]+$/g;
export const tagNameMaskForReplace = /[^\w\d_\-]+/g;
