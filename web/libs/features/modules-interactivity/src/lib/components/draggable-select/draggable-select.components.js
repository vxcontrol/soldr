import Sortable from 'sortablejs';

const template = `
    <el-select ref="dragSelect" v-model="selectVal" v-bind="$attrs" class="drag-select" multiple v-on="$listeners">
        <slot/>
    </el-select>
`;

export const DraggableSelect = {
    name: 'draggable-select',

    props: {
        value: {
            type: Array,
            required: true
        }
    },

    computed: {
        selectVal: {
            get() {
                return [...this.value];
            },
            set(val) {
                this.$emit('input', [...val]);
            }
        }
    },

    mounted() {
        this.setSort();
    },

    methods: {
        setSort() {
            const el = this.$refs.dragSelect.$el.querySelectorAll('.el-select__tags > span')[0];

            this.sortable = Sortable.create(el, {
                ghostClass: 'sortable-ghost',
                setData(dataTransfer) {
                    dataTransfer.setData('Text', '');
                },
                onEnd: (event) => {
                    const target = this.value.splice(event.oldIndex, 1)[0];

                    this.value.splice(event.newIndex, 0, target);
                    this.$set(this, 'value', this.value);
                }
            });
        }
    },

    template
};
