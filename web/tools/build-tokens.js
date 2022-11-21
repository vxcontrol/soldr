const path = require('path');

const buildTokens = require('../node_modules/@ptsecurity/mosaic/design-tokens/style-dictionary/build');

const skins = ['pt-2022', 'legacy-2017'];

buildTokens(
    skins.map((skin) => {
        const overriddenMosaicTokensPath = path.resolve(
            __dirname,
            `../libs/styles/src/lib/design-tokens/${skin}/mosaic`
        );
        const appTokensPath = path.resolve(__dirname, `../libs/styles/src/lib/design-tokens/${skin}/token`);

        return {
            name: 'default-theme',
            buildPath: [
                // originally mosaic tokens
                `node_modules/@ptsecurity/mosaic/design-tokens/${skin}/tokens/properties/**/*.json5`,
                `node_modules/@ptsecurity/mosaic/design-tokens/${skin}/tokens/components/**/*.json5`,

                // overridden mosaic tokens
                `${overriddenMosaicTokensPath}/properties/**/*.json5`,
                `${overriddenMosaicTokensPath}/components/**/*.json5`,

                // app tokens
                `${appTokensPath}/**/*.json5`
            ],
            outputPath: `libs/styles/src/lib/design-tokens/${skin}/default-theme/`
        };
    })
);
