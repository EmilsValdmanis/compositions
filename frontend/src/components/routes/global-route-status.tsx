import React from "react";
import * as Sentry from "@sentry/tanstackstart-react";
import { Link, type ErrorComponentProps, type NotFoundRouteProps } from "@tanstack/react-router";
import { Button } from "#/components/ui/button";
import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "#/components/ui/card";

function errorMessage(error: unknown) {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  if (typeof error === "string" && error.length > 0) {
    return error;
  }

  return "An unexpected error interrupted this page.";
}

function RouteStatusFrame({
  eyebrow,
  title,
  description,
  children,
}: {
  eyebrow: string;
  title: string;
  description: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="flex min-h-full flex-1 items-center justify-center p-6 sm:p-8">
      <Card className="w-full max-w-lg border border-border/70 bg-card/95 shadow-xl backdrop-blur-sm">
        <CardHeader className="gap-3">
          <div className="text-xs font-medium tracking-[0.24em] text-muted-foreground uppercase">
            {eyebrow}
          </div>
          <div className="space-y-1">
            <CardTitle className="text-2xl">{title}</CardTitle>
            <CardDescription>{description}</CardDescription>
          </div>
        </CardHeader>
        {children ? <CardFooter className="gap-3">{children}</CardFooter> : null}
      </Card>
    </div>
  );
}

class GlobalRouteErrorBoundary extends React.Component<ErrorComponentProps> {
  render() {
    return (
      <RouteStatusFrame
        eyebrow="Application Error"
        title="Something went wrong"
        description={errorMessage(this.props.error)}
      >
        <Button type="button" onClick={this.props.reset}>
          Try again
        </Button>
        <Button render={<Link to="/" />} nativeButton={false} variant="outline">
          Go home
        </Button>
      </RouteStatusFrame>
    );
  }
}

const SentryWrappedGlobalRouteErrorBoundary = Sentry.withErrorBoundary(GlobalRouteErrorBoundary, {
  fallback: (
    <RouteStatusFrame
      eyebrow="Application Error"
      title="Something went wrong"
      description="The error view failed to render, but the problem has been captured."
    >
      <Button type="button" onClick={() => window.location.assign("/")}>
        Reload app
      </Button>
    </RouteStatusFrame>
  ),
});

export function GlobalErrorComponent(props: ErrorComponentProps) {
  React.useEffect(() => {
    Sentry.captureException(props.error);
  }, [props.error]);

  return <SentryWrappedGlobalRouteErrorBoundary {...props} />;
}

export function GlobalNotFoundComponent(_: NotFoundRouteProps) {
  return (
    <RouteStatusFrame
      eyebrow="404"
      title="Page not found"
      description="The page you requested does not exist or may have moved."
    >
      <Button render={<Link to="/" />} nativeButton={false}>
        Back to home
      </Button>
    </RouteStatusFrame>
  );
}
