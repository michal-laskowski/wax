import * as jr from "./jsx-runtime";
import { Child } from "./jsx-runtime";

interface WaxType {
    Raw(v: string): Child;
    Now(v: Child): string;
}

declare global {
    var wax: WaxType;

    type PropsWithChildren<P = unknown> = P & {
        children?: Child | undefined;
    };

    interface CustomIntrinsicElements {}

    namespace JSX {
        export import IntrinsicElements = jr.JSX.IntrinsicElements;
        export import HTMLAttributes = jr.JSX.HTMLAttributes;
        export import Child = jr.Child;
    }
}

declare module "wax-jsx/jsx-runtime" {
    namespace JSX {
        interface IntrinsicElements extends CustomIntrinsicElements {}
    }
}
