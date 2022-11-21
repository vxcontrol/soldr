export interface NcformRule {
    value?: any; // The value passed to the validation rule
    errMsg?: string; // Error message
    options?: {
        // Rule options
        delay?: number; // Delayed verification time (ms)
        delayMsg?: string; // Prompt for delayed verification
    };
}

export enum PropertyType {
    STRING = 'string',
    NUMBER = 'number',
    INTEGER = 'integer',
    BOOLEAN = 'boolean',
    OBJECT = 'object',
    ARRAY = 'array',
    HTML = 'HTML',
    COMP = 'COMP',
    NONE = 'none'
}

export interface NcFormReference extends NcFormProperty {
    $ref?: string;
}

export interface NcFormProperty {
    allOf?: Partial<NcFormReference>[];
    /* Data */
    type?: PropertyType;
    // Note: The type of uppercase is a special read-only type, and the common use case is to display a separator bar. The data will be auto filtered out when the form is submitted.
    // HTML: set "value", the value is a piece of HTML [support dx expression];
    // COMP: set ui.widget and ui.widgetConfig

    value?: any; // Value of the field
    default?: any; // The default value of the field. Take this one when the "value" is empty.
    valueTemplate?: string; // Value template. Dynamically calculate the "value" based on the supplied dx expression [support dx expression]
    items?: any;
    enum?: any[];
    minItems?: number;
    maxItems?: number;

    properties?: {
        [fieldName: string]: NcFormProperty;
    };

    /* UI */
    ui?: {
        columns?: number; // Total are 12 columns. [support dx expression]
        label?: string; // Label display [support dx expression]
        showLabel?: boolean; // Whether to show the label (when it is false, it still takes up space)
        noLabelSpace?: boolean; // Whether the label does not occupy space, the priority is higher than showLabel
        legend?: string; // Legend content, valid when the type is object or array [support dx expression]
        showLegend?: boolean; // Whether to display the legend.
        description?: string; // Description information [support dx expression]
        placeholder?: string; // Placeholder content [support dx expression]
        disabled?: boolean; // Whether to disable [support dx expression]
        readonly?: boolean; // Whether read-only [support dx expression]
        hidden?: boolean; // Whether to hide [support dx expression]
        help?: {
            // Help information
            show?: boolean; // Whether to display, default is false
            content?: string; // Help detail information
            iconCls?: string; // Help icon class name
            text?: string; // Help text
        };
        itemClass?: string; // The form item class name
        preview?: {
            // Preview
            type: 'video' | 'audio' | 'image' | 'link'; // Preview type. Options: video / audio / image / link
            value?: string; // Default: 'dx: {{$self}}' [supports dx expressions]
            clearable?: boolean; // Whether to display the clear button
            outward?: {
                // outward appearance. Valid only if type=image
                width?: number; // Width, 0 means unlimited
                height?: number; // Height, 0 means unlimited
                shape?: string; // Appearance shape. Options: '' / rounded / circle. default is ''
            };
        };
        // Associated fields. when the value changes, it will trigger some actions of the associated field, such as rules check
        linkFields?: {
            fieldPath?: string; // The associated item field path. such as 'user.name'，'user[i].name'
            rules?: string[]; // The rules, such as ['required']
        }[];

        /* Rendering Widget */
        widget?: string; // Widget component name
        widgetConfig?: any; // widget component config
    };

    required?: string[];

    /* Verification rules */
    rules?: {
        // All validation rules have two forms of assignment:
        // Simple version: <rule name>: <rule value>. Such as required: true, minimum: 10
        // Detailed version: <rule name>: { value: <rule value>, errMsg: '', options: { deplay: xxx, delayMsg: '' } }. Such as the following required example

        // for Any Instance Type
        required?: NcformRule | boolean;
        // eslint-disable-next-line id-blacklist
        number?: NcformRule | boolean; // value:boolean
        ajax?:
            | NcformRule
            | {
                  remoteUrl: string;
                  method: 'get' | 'post';
                  paramName: string;
                  otherParams: Record<string, any>;
              }; // Value: { remoteUrl: 'remote api url', method: 'get or post', paramName: 'request parameter name, the value is the control\'s value', otherParams: {} }

        // for Numeric Instances
        minimum?: NcformRule | number; // value: number
        maximum?: NcformRule | number; // value: number
        multipleOf?: NcformRule | number; // value: number
        exclusiveMaximum?: NcformRule | number; // value: number
        exclusiveMinimum?: NcformRule | number; // value: number

        // for Strings
        url?: NcformRule | boolean; // value: boolean
        tel?: NcformRule | boolean; // value: boolean
        ipv4?: NcformRule | boolean; // value: boolean
        ipv6?: NcformRule | boolean; // value: boolean
        email?: NcformRule | boolean; // value: boolean
        pattern?: NcformRule | string; // value: string。 such as "\\d+"
        hostname?: NcformRule | boolean; // value: boolean
        dateTime?: NcformRule | boolean; // value: boolean
        maxLength?: NcformRule | number; // value: number
        minLength?: NcformRule | number; // value: number

        // for Arrays
        contains?: NcformRule | any; // value: any
        maxItems?: NcformRule | number; // value: number
        minItems?: NcformRule | number; // value: number
        uniqueItems?: NcformRule | boolean; // value: boolean

        // for Objects
        maxProperties?: NcformRule | number; // value: number
        minProperties?: NcformRule | number; // value: number

        /* Custom Validation Rules */
        customRule?: {
            script?: string; // [Support dx expression]
            errMsg?: string; // Error message
            // When the check is triggered, the customRule rule validation of these associated items is also triggered (recommended using ui.linkFields instead)
            linkItems?: {
                fieldPath?: string; // The associated item field path. such as 'user.name'，'user[i].name'
                customRuleIdx: number; // The index of the customRule of the link item
            }[];
        }[];
    };
}

export interface NcformSchema {
    type: 'object'; // Root node. object type only
    definitions?: Record<string, any>;
    additionalProperties?: boolean;
    properties: {
        [fieldName: string]: NcFormProperty;
    };
    globalConfig?: {
        // Global configuration
        ignoreRulesWhenHidden?: boolean; // When the controls are hidden, its validation rules are automatically ignored. Default is true
        style?: {
            // Global style configuration
            formCls?: string; // Form class
            invalidFeedbackCls?: string; // Invalid feedback class
        };
        constants?: Record<string, any>;
        scrollToFailField?: {
            // Automatically scroll to fields that failed validation
            enabled?: boolean; // Enable this feature or not
            container?: string; // The container that has to be scrolled.
            duration?: number; // The duration (in milliseconds) of the scrolling animation
            offset?: number; // The offset that should be applied when scrolling.
        };
    };
    required?: string[];
    ui?: Record<string, any>;
}

export const usedPropertyTypes = ['string', 'number', 'integer', 'boolean', 'object', 'array'];
