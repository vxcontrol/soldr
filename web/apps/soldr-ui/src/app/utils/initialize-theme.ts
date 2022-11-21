export const isDarkTheme = true;

export const initializeTheme = () =>
    isDarkTheme ? document.body.classList.add('dark-theme') : document.body.classList.add('light-theme');
