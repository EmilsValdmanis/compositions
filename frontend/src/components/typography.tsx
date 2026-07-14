import { cva, type VariantProps } from "class-variance-authority";
import {
  type ComponentProps,
  type ComponentPropsWithoutRef,
  type ElementType,
  type ReactNode,
} from "react";
import { cn } from "#/lib/utils";

const typographyVariants = cva("font-mono", {
  variants: {
    variant: {
      display: "font-heading text-4xl/none font-bold tracking-tight text-balance sm:text-7xl/none",
      h1: "font-heading text-3xl/9 font-semibold tracking-tight text-balance md:text-4xl/10",
      h2: "font-heading text-2xl/8 font-semibold tracking-tight text-balance",
      h3: "font-heading text-xl/7 font-semibold tracking-tight text-balance",
      h4: "font-heading text-lg/6 font-semibold tracking-tight text-balance",
      h5: "font-heading text-base/6 font-semibold text-balance",
      h6: "font-heading text-sm/5 font-semibold text-balance",
      body: "text-sm/5 font-normal tracking-normal",
      "body-strong": "text-sm/5 font-medium tracking-normal",
      caption: "text-xs/5 font-normal tracking-normal text-muted-foreground",
      "caption-strong": "text-xs/5 font-medium tracking-normal",
      label: "text-xs/4 font-medium tracking-normal",
      status: "text-xs/4 font-medium tracking-normal capitalize",
      eyebrow: "text-xs/4 font-medium tracking-[0.18em] uppercase text-muted-foreground",
      "eyebrow-wide": "text-xs/4 font-medium tracking-[0.24em] uppercase text-muted-foreground",
      "eyebrow-compact":
        "text-[0.7rem]/4 font-normal tracking-normal uppercase text-muted-foreground",
      brand:
        "text-sm/5 font-semibold tracking-[0.16em] uppercase text-foreground/90 md:text-base/6",
      metric: "font-heading text-4xl/10 font-semibold tracking-tight tabular-nums",
      stat: "font-heading text-3xl/9 font-medium tracking-tight tabular-nums",
      numeric: "text-sm/5 font-medium tracking-normal tabular-nums",
      activity: "text-[0.65rem]/4 font-medium tracking-wide uppercase text-muted-foreground",
      "turn-eyebrow":
        "text-[0.55rem]/none font-medium tracking-[0.2em] uppercase text-muted-foreground",
      "turn-title": "text-[1.05rem]/none font-bold tracking-[-0.04em] uppercase",
      "turn-meta":
        "text-[0.5rem]/[1.45] font-normal tracking-[0.16em] uppercase text-muted-foreground tabular-nums",
      "card-rank-hand": "text-lg/none font-semibold",
      "card-rank-default": "text-sm/none font-semibold",
      "card-rank-compact": "text-[0.65rem]/none font-semibold",
      symbol: "text-[0.7rem]/none font-normal",
      "emoji-sm": "text-sm/none font-normal",
      "emoji-base": "text-base/none font-normal",
      "emoji-lg": "text-lg/none font-normal",
      "emoji-xl": "text-xl/none font-normal",
    },
  },
  defaultVariants: {
    variant: "body",
  },
});

type TypographyVariant = NonNullable<VariantProps<typeof typographyVariants>["variant"]>;

type TextProps<T extends ElementType> = {
  as?: T;
  children?: ReactNode;
  variant?: TypographyVariant;
} & Omit<ComponentPropsWithoutRef<T>, "as" | "children">;

function Text<T extends ElementType = "span">({ as, className, variant, ...props }: TextProps<T>) {
  const Component = as ?? "span";

  return <Component className={cn(typographyVariants({ variant }), className)} {...props} />;
}

type HeadingProps<T extends "h1" | "h2" | "h3" | "h4" | "h5" | "h6"> = Omit<
  ComponentProps<T>,
  "children"
> & {
  children: ReactNode;
};

function H1(props: HeadingProps<"h1">) {
  return <Text as="h1" variant="h1" {...props} />;
}

function H2(props: HeadingProps<"h2">) {
  return <Text as="h2" variant="h2" {...props} />;
}

function H3(props: HeadingProps<"h3">) {
  return <Text as="h3" variant="h3" {...props} />;
}

function H4(props: HeadingProps<"h4">) {
  return <Text as="h4" variant="h4" {...props} />;
}

function H5(props: HeadingProps<"h5">) {
  return <Text as="h5" variant="h5" {...props} />;
}

function H6(props: HeadingProps<"h6">) {
  return <Text as="h6" variant="h6" {...props} />;
}

const paragraphVariants = cva("", {
  variants: {
    size: {
      xs: "text-xs/5",
      sm: "text-sm/5",
      default: "text-sm/6",
      lg: "text-base/7",
    },
  },
  defaultVariants: {
    size: "default",
  },
});

type PProps = ComponentProps<"p"> & VariantProps<typeof paragraphVariants>;

function P({ className, size, ...props }: PProps) {
  return (
    <Text as="p" variant="body" className={cn(paragraphVariants({ size }), className)} {...props} />
  );
}

function Caption(props: ComponentProps<"p">) {
  return <Text as="p" variant="caption" {...props} />;
}

function Eyebrow(props: ComponentProps<"p">) {
  return <Text as="p" variant="eyebrow" {...props} />;
}

function Strong(props: ComponentProps<"strong">) {
  return <Text as="strong" variant="body-strong" {...props} />;
}

export { Caption, Eyebrow, H1, H2, H3, H4, H5, H6, P, Strong, Text, typographyVariants };
export type { TypographyVariant };
