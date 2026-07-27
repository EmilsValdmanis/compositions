import { createFileRoute, redirect } from "@tanstack/react-router";
import { AdminBugReportsPage } from "#/components/routes/admin-bug-reports-page";
import { adminBugReportPageOptions } from "#/lib/admin-bug-reports";
import { pageTitle } from "#/lib/page-title";
import { m } from "#/paraglide/messages.js";

export const Route = createFileRoute("/_protected/admin")({
  beforeLoad: ({ context }) => {
    if (!context.session?.user.isAdmin) {
      throw redirect({ to: "/" });
    }
  },
  loader: async ({ context }) => {
    return context.queryClient.ensureQueryData(adminBugReportPageOptions(1));
  },
  head: () => ({
    meta: [{ title: pageTitle(m.admin()) }],
  }),
  component: AdminRoute,
});

function AdminRoute() {
  const initialPage = Route.useLoaderData();
  return <AdminBugReportsPage initialPage={initialPage} />;
}
