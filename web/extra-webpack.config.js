module.exports = (config) => {
    config.module.rules.unshift({
        test: /\.css$/i,
        use: ['style-loader', 'css-loader']
    });

    return config;
};
