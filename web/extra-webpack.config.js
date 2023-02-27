const MonacoWebpackPlugin = require('monaco-editor-webpack-plugin');

module.exports = (config) => {
    config.module.rules.unshift(
        {
            test: /\.css$/i,
            use: ['style-loader', 'css-loader']
        },
        {
            test: /\.ttf$/,
            use: ['file-loader']
        }
    );

    config.plugins.push(new MonacoWebpackPlugin());

    config.optimization.minimize = true;

    config.optimization.minimizer = [
      new TerserPlugin({
        minify: TerserPlugin.terserMinify,
        test: /\.js$/i
      }),
      ...(config.optimization.minimizer || [])
    ];

    return config;
};
