import React from "react";
import * as Sentry from "@sentry/tanstackstart-react";
import { Link, type ErrorComponentProps, type NotFoundRouteProps } from "@tanstack/react-router";
import { Button } from "#/components/ui/button";
import { Card, CardDescription, CardFooter, CardHeader } from "#/components/ui/card";
import { Caption, H2 } from "#/components/typography";
import { m } from "#/paraglide/messages.js";

function errorMessage(error: unknown) {
  void error;
  return m.unexpected_page_error();
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
          <Caption className="font-medium tracking-[0.24em] uppercase">{eyebrow}</Caption>
          <div className="space-y-1">
            <H2>{title}</H2>
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
        eyebrow={m.application_error()}
        title={m.something_went_wrong()}
        description={errorMessage(this.props.error)}
      >
        <Button type="button" onClick={this.props.reset}>
          {m.try_again()}
        </Button>
        <Button render={<Link to="/" />} nativeButton={false} variant="outline">
          {m.go_home()}
        </Button>
      </RouteStatusFrame>
    );
  }
}

const SentryWrappedGlobalRouteErrorBoundary = Sentry.withErrorBoundary(GlobalRouteErrorBoundary, {
  fallback: (
    <RouteStatusFrame
      eyebrow={m.application_error()}
      title={m.something_went_wrong()}
      description={m.error_view_failed()}
    >
      <Button type="button" onClick={() => window.location.assign("/")}>
        {m.reload_app()}
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
      title={m.page_not_found()}
      description={m.page_not_found_description()}
    >
      <Button render={<Link to="/" />} nativeButton={false}>
        {m.back_home()}
      </Button>
    </RouteStatusFrame>
  );
}
