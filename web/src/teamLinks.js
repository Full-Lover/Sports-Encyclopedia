export function officialSite(url) {
  try {
    const parsed = new URL(url);
    return parsed.protocol === "https:" && parsed.hostname && !parsed.username && !parsed.password
      ? parsed.href : null;
  } catch {
    return null;
  }
}

export function teamPath(path) {
  return typeof path === "string" && /^\/teams\/[a-z0-9]+(?:-[a-z0-9]+)*$/.test(path) ? path : null;
}
