import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

import { FilesTreeNode, FileTreeNodeType, ModuleFolderSection } from '../types';

@Injectable({
    providedIn: 'root'
})
export class FilesTreeService {
    private clibsFolders = ['windows/amd64', 'darwin/amd64', 'linux/amd64', 'windows/386', 'linux/386'];
    private clibsFoldersWithSys = this.clibsFolders.map((v) => `${v}/sys/`);
    private allowedClibsPattern = new RegExp(
        `^${this.clibsFolders.map((v) => `(${v.replace('/', '\\/')})`).join('|')}`
    );

    constructor(private transloco: TranslocoService) {}

    buildTree(files: string[], hideClibs: boolean, section?: ModuleFolderSection): FilesTreeNode[] {
        if (!section) {
            const agentSection: FilesTreeNode = {
                type: FileTreeNodeType.Section,
                data: {
                    name: this.transloco.translate('modules.Modules.ModuleEdit.Label.AgentModule'),
                    path: 'cmodule'
                },
                level: 0
            };
            agentSection.children = this.buildSectionTree(files, ModuleFolderSection.Agent, hideClibs, agentSection);

            const serverSection: FilesTreeNode = {
                type: FileTreeNodeType.Section,
                data: {
                    name: this.transloco.translate('modules.Modules.ModuleEdit.Label.ServerModule'),
                    path: 'smodule'
                },
                level: 0
            };
            serverSection.children = this.buildSectionTree(files, ModuleFolderSection.Server, hideClibs, serverSection);

            const browserSection: FilesTreeNode = {
                type: FileTreeNodeType.Section,
                data: {
                    name: this.transloco.translate('modules.Modules.ModuleEdit.Label.BrowserModule'),
                    path: 'bmodule'
                },
                level: 0
            };
            browserSection.children = this.buildSectionTree(
                files,
                ModuleFolderSection.Browser,
                hideClibs,
                browserSection
            );

            return [agentSection, serverSection, browserSection];
        }

        return this.buildSectionTree(files, section, hideClibs);
    }

    isValidClibsFilePath(value: string) {
        return this.allowedClibsPattern.test(value);
    }

    private buildSectionTree(
        files: string[],
        section: ModuleFolderSection,
        hideClibs: boolean,
        parent?: FilesTreeNode
    ) {
        const sectionFiles = this.getFilesBySection(files, section);
        const partLevel = parent ? parent.level + 1 : 0;

        const codePart: FilesTreeNode = {
            type: FileTreeNodeType.Part,
            data: {
                name: 'code',
                path: `${section}`
            },
            level: partLevel,
            parent
        };
        codePart.children = this.getTree(this.getCodeFiles(sectionFiles), codePart);

        const dataPart: FilesTreeNode = {
            type: FileTreeNodeType.Part,
            data: {
                name: 'data',
                path: `${section}/data`
            },
            level: partLevel,
            parent
        };
        dataPart.children = this.getTree(this.getDataFiles(sectionFiles), dataPart);

        const clibsPart: FilesTreeNode = {
            type: FileTreeNodeType.Part,
            data: {
                name: 'clibs',
                path: `${section}/clibs`
            },
            level: partLevel,
            parent
        };
        clibsPart.children = this.getTree(this.geCLibsFiles(sectionFiles), clibsPart);

        return [codePart, dataPart, ...(hideClibs ? [] : [clibsPart])];
    }

    private getFilesBySection(files: string[], section: string) {
        return files
            .filter((path) => new RegExp(`^${section}`).test(path))
            .map((path) => path.split('/').slice(1).join('/'));
    }

    private getTree(files: string[], parent: FilesTreeNode): FilesTreeNode[] {
        const data = [...files].sort();
        const groupedPathByFirstSegment: Record<string, string[]> = {};
        const rest = [];

        for (const path of data) {
            const segments = path.split('/');

            if (segments.length > 1) {
                const folderName = segments[0];

                if (!groupedPathByFirstSegment[folderName]) {
                    groupedPathByFirstSegment[folderName] = [];
                }

                groupedPathByFirstSegment[folderName].push(segments.splice(1).join('/'));
            } else if (segments[0]) {
                rest.push(segments[0]);
            }
        }

        return [
            ...Object.keys(groupedPathByFirstSegment).map((folderName) => {
                const folder: FilesTreeNode = {
                    type: FileTreeNodeType.Folder,
                    level: parent.level + 1,
                    data: {
                        name: folderName,
                        path: `${parent.data.path}/${folderName}`
                    },
                    parent
                };

                folder.children = this.getTree(groupedPathByFirstSegment[folderName], folder);

                return folder;
            }),
            ...rest.map((fileName) => ({
                type: FileTreeNodeType.File,
                level: parent.level + 1,
                data: {
                    name: fileName,
                    path: `${parent.data.path}/${fileName}`
                },
                parent
            }))
        ];
    }

    private getCodeFiles(files: string[]) {
        const excludedPaths = ['data/', 'clibs/'];

        return files.filter((path) => !excludedPaths.some((excludedPath) => path.startsWith(excludedPath)));
    }

    private getDataFiles(files: string[]) {
        return files.filter((path) => path.startsWith('data/')).map((path) => path.replace(/^data\//, ''));
    }

    private geCLibsFiles(files: string[]) {
        return [...this.clibsFoldersWithSys, ...files.filter((path) => path.startsWith('clibs/'))].map((path) =>
            path.replace(/^clibs\//, '')
        );
    }
}
