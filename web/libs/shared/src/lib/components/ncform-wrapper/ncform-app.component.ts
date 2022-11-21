/* eslint-disable @typescript-eslint/ban-ts-comment */
// @ts-ignore
import Vue from 'vue';
import { VueConstructor } from 'vue/types/vue';

import { clone, disableNcformWidgets } from '../../utils';

const template = `
<!--suppress AngularInvalidAnimationTriggerAssignment -->
<ncform
    v-if="schema && model && canRenderForm"
    name="ncform"
    :form-schema="schema"
    :is-dirty.sync="isDirty"
    form-name="ncform"
    v-model="model"
    @change="onChange">
</ncform>
`;

// eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
export const NcformAppComponent = (Vue as VueConstructor).extend({
    name: 'ncform-app',
    data: () => ({
        isDirty: false,
        model: undefined,
        schema: undefined,
        canRenderForm: false
    }),
    watch: {
        '$root.model'(v) {
            this.model = clone(v);
        },
        '$root.schema'(v) {
            this.canRenderForm = false;
            const schema = clone(v);
            // @ts-ignore
            if (this.$root.isReadOnly) {
                disableNcformWidgets(schema as Record<string, any>);
            }
            this.schema = schema;
            // без canRenderForm и isDirty не сбрасывается флаг isDirty после сохранения конфига модуля
            setTimeout(() => {
                this.canRenderForm = true;
                this.isDirty = false;
            });
        },
        '$root.isReadOnly'(v) {
            if (v) {
                disableNcformWidgets(this.schema as Record<string, any>);
            }
        },
        isDirty(v) {
            // @ts-ignore
            this.$root.isDirty = v;
        }
    },
    methods: {
        validate(): Promise<any> {
            // @ts-ignore
            return this.$ncformValidate('ncform');
        },
        getValue() {
            // @ts-ignore
            return clone(this.model);
        },
        onChange(v: any) {
            // for dx expressions in valueTemplate (fields in definitions.json), it's logic ncform
            const updateWaitMs = 1000;
            // @ts-ignore
            setTimeout(() => this.$root.onChangeModel(clone(this.model)), updateWaitMs);
        }
    },

    template
});
