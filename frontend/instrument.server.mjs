import * as Sentry from "@sentry/tanstackstart-react";

Sentry.init({
  dsn: "https://1b571087a2847838e64e3a7856ee9533@o4511438083653632.ingest.de.sentry.io/4511438085161040",
  enabled: process.env.NODE_ENV === "production",

  // Setting this option to true will send default PII data to Sentry.
  // For example, automatic IP address collection on events
  sendDefaultPii: true,
});
