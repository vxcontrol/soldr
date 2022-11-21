export function parseErrorCode(errorResponse: string): string {
    return errorResponse?.split('.').reverse()[0];
}
