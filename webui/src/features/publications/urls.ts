export function resolvePublicURL(path: string): string {
  const base = import.meta.env.VITE_API_BASE_URL?.trim();
  if (base && (base.startsWith("http://") || base.startsWith("https://"))) {
    return new URL(path, base).toString();
  }
  return `${window.location.origin}${path}`;
}
