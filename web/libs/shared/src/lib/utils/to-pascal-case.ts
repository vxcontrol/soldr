import { capitalize } from './capitalize';

export function toPascalCase(inputString: string): string {
    return capitalize(
        inputString
            // заменяем букву после не-буквы на заглавную
            .replace(/[-_.](\w)/gi, (match, p1: string) => capitalize(p1))
            // удаляем не-буквы и не-цифры
            .replace(/[^a-z0-9\d]/gi, '')
    );
}
