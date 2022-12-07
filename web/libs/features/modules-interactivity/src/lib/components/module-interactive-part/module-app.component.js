import * as luxon from 'luxon';
import * as monaco from 'monaco-editor';
import Vue from 'vue';

import { DraggableSelect as draggableSelect, GridComponent as grid } from '..';

const template = `
    <div class="module-interactivity-container layout-padding-l">
        <component
            v-if="canShow"
            :view-mode="$root.viewMode"
            :is="$root.subview"
            :entity="$root.entity"
            :hash="$root.entity.hash"
            :module="$root.module"
            :protoAPI="$root.protoAPI"
            :components="components"
            :helpers="helpers"
            :api="$root.api">
        </component>
    </div>
`;

export const ModuleAppComponent = Vue.extend({
    name: 'module-app',

    props: [],

    data: () => ({
        components: {
            draggableSelect,
            grid,
            monaco
        },
        helpers: {
            luxon
        },
        subview: undefined
    }),

    mounted() {
        this.helpers.getView = this.$root.api.modulesAPI.getView.bind(this.$root.api.modulesAPI);
    },

    computed: {
        canShow() {
            return this.$root.entity && this.$root.module && this.components && this.helpers && this.$root.subview;
        }
    },

    template
});
