const BASE_PATH = import.meta.env.BASE_URL.replace(/\/$/, "");

export function withBasePath(href: string): string {
  if (!href || href.startsWith("#") || /^[a-z][a-z\d+.-]*:/i.test(href)) {
    return href;
  }

  const normalized = href.startsWith("/") ? href : `/${href}`;
  if (!BASE_PATH || normalized === BASE_PATH || normalized.startsWith(`${BASE_PATH}/`)) {
    return normalized;
  }

  return `${BASE_PATH}${normalized}`;
}

export function stripBasePath(pathname: string): string {
  const normalized = pathname.replace(/\/$/, "") || "/";
  if (!BASE_PATH || normalized === BASE_PATH) return "/";
  if (normalized.startsWith(`${BASE_PATH}/`)) {
    return normalized.slice(BASE_PATH.length) || "/";
  }
  return normalized;
}

export function getBasePath(): string {
  return BASE_PATH;
}
