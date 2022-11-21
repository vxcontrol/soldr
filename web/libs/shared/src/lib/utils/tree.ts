export interface ITreeNode {
    children?: ITreeNode[];
}

export type TreeNodeCallbackNode<T extends ITreeNode> = (node: T, level: number, parent?: T) => any;

export function traverseTree<T extends ITreeNode>(tree: T[], callback: TreeNodeCallbackNode<T>, level = 0, parent?: T) {
    if (tree && Array.isArray(tree)) {
        for (const item of tree) {
            callback(item, level, parent);

            if (item && item.children && Array.isArray(item.children)) {
                traverseTree(item.children as T[], callback, level + 1, item);
            }
        }
    }
}
