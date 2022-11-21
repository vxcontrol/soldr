class NotificationsService {
    constructor(app) {
        this.app = app;
    }

    error(message, title) {
        this.app.$notify({
            title,
            message,
            duration: 5000,
            type: 'error'
        });
    }

    warning(message, title) {
        this.app.$notify({
            title,
            message,
            duration: 5000,
            type: 'warning'
        });
    }

    success(message, title) {
        this.app.$notify({
            title,
            message,
            duration: 5000,
            type: 'success'
        });
    }
}

export { NotificationsService };
