import { cva, type VariantProps } from "class-variance-authority";
import { type ComponentProps, type ReactNode } from "react";
import { cn } from "#/lib/utils";

type HeadingProps<T extends "h1" | "h2" | "h3" | "h4" | "h5" | "h6"> = Omit<
  ComponentProps<T>,
  "children"
> & {
  children: ReactNode;
};

function H1({ className, ...props }: HeadingProps<"h1">) {
  return (
    <h1
      className={cn(
        "font-heading text-3xl/9 font-semibold tracking-tight text-balance md:text-4xl/10",
        className,
      )}
      {...props}
    />
  );
}

function H2({ className, ...props }: HeadingProps<"h2">) {
  return (
    <h2
      className={cn("font-heading text-2xl/8 font-semibold tracking-tight text-balance", className)}
      {...props}
    />
  );
}

function H3({ className, ...props }: HeadingProps<"h3">) {
  return (
    <h3
      className={cn("font-heading text-xl/7 font-semibold tracking-tight text-balance", className)}
      {...props}
    />
  );
}

function H4({ className, ...props }: HeadingProps<"h4">) {
  return (
    <h4
      className={cn("font-heading text-lg/6 font-semibold tracking-tight text-balance", className)}
      {...props}
    />
  );
}

function H5({ className, ...props }: HeadingProps<"h5">) {
  return (
    <h5
      className={cn("font-heading text-base/6 font-semibold text-balance", className)}
      {...props}
    />
  );
}

function H6({ className, ...props }: HeadingProps<"h6">) {
  return (
    <h6 className={cn("font-heading text-sm/5 font-semibold text-balance", className)} {...props} />
  );
}

const paragraphVariants = cva("text-sm font-normal tracking-normal", {
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
  return <p className={cn(paragraphVariants({ size }), className)} {...props} />;
}

function Caption({ className, ...props }: ComponentProps<"p">) {
  return (
    <p
      className={cn("text-xs/5 font-normal tracking-normal text-muted-foreground", className)}
      {...props}
    />
  );
}

export { Caption, H1, H2, H3, H4, H5, H6, P };
