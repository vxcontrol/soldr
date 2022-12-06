import { AfterViewInit, Component, ElementRef, Inject, Input, OnChanges, SimpleChanges } from '@angular/core';
import * as monaco from 'monaco-editor';

import { MosaicTokens, THEME_TOKENS } from '@soldr/core';

import IStandaloneEditorConstructionOptions = monaco.editor.IStandaloneEditorConstructionOptions;

@Component({
    selector: 'soldr-event-data-block',
    templateUrl: './event-data-block.component.html',
    styleUrls: ['./event-data-block.component.scss']
})
export class EventDataBlockComponent implements OnChanges, AfterViewInit {
    @Input() data = '';

    isLoading = true;
    options: IStandaloneEditorConstructionOptions = {
        automaticLayout: true,
        folding: false,
        foldingHighlight: false,
        fontSize: parseInt(this.tokens.TypographySubheadingFontSize),
        glyphMargin: false,
        hideCursorInOverviewRuler: true,
        language: 'json',
        lineDecorationsWidth: 0,
        lineNumbers: 'off',
        lineNumbersMinChars: 0,
        minimap: { enabled: false },
        overviewRulerBorder: false,
        overviewRulerLanes: 0,
        padding: { top: 12 },
        readOnly: true,
        renderLineHighlight: 'none',
        selectionHighlight: false,
        theme: 'soldrJsonTheme'
    };
    editor: any;

    constructor(private element: ElementRef, @Inject(THEME_TOKENS) private tokens: MosaicTokens) {}

    ngAfterViewInit(): void {
        const editorContainer: HTMLElement = this.element.nativeElement.querySelector('.event-data-block__editor');
        this.editor = monaco.editor.create(editorContainer, {
            ...this.options,
            value: this.data
        });

        this.editor.onDidLayoutChange(() => {
            this.isLoading = false;
        });
    }

    ngOnChanges({ data }: SimpleChanges): void {
        if (data?.currentValue) {
            this.editor?.setValue(this.data);
        }
    }

    copy() {
        window.navigator.clipboard?.writeText(this.data);
    }
}
