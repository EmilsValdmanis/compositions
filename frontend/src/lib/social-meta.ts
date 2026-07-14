export const defaultSocialDescription =
  "A tactical rummy-style card game about building runs, closing rounds, and staying under 100 points.";

type SocialMetaDescriptor =
  | { name: string; content: string }
  | { property: string; content: string };

export function createSocialMeta({
  title,
  description,
  origin,
  type = "website",
  url,
}: {
  title: string;
  description: string;
  origin: string;
  type?: "profile" | "website";
  url?: string;
}): SocialMetaDescriptor[] {
  const image = `${origin}/social-card.png`;

  return [
    { name: "description", content: description },
    { property: "og:type", content: type },
    { property: "og:site_name", content: "Compositions" },
    { property: "og:title", content: title },
    { property: "og:description", content: description },
    ...(url ? [{ property: "og:url", content: url }] : []),
    { property: "og:image", content: image },
    { property: "og:image:width", content: "1200" },
    { property: "og:image:height", content: "630" },
    {
      property: "og:image:alt",
      content: "Compositions — build runs, close rounds, play to 100",
    },
    { name: "twitter:card", content: "summary_large_image" },
    { name: "twitter:title", content: title },
    { name: "twitter:description", content: description },
    { name: "twitter:image", content: image },
  ];
}
