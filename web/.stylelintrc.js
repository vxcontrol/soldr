module.exports = {
    ignoreFiles: ['libs/styles/**/design-tokens/**/*.scss', 'libs/styles/**/element-ui/**/*.scss'],
    plugins: 'stylelint-scss',
    rules: {
        'at-rule-no-unknown': null,
        'scss/at-rule-no-unknown': true,
        'value-keyword-case': ['lower', { ignoreKeywords: ['/[A-Z]\\d+/'] }]
    }
};
