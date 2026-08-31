import { useEffect, useState } from "react";

export type RouteArea = "ledgers" | "changes" | "proposals" | "releases";
export type Route = { area: RouteArea; id?: string };

export function parseRoute(hash: string): Route {
  const [area, id, ...rest] = hash.replace(/^#/, "").split("/");
  if (area === "proposals" && id && rest.length === 0) return { area, id };
  if ((["ledgers", "changes", "proposals", "releases"] as string[]).includes(area) && !id) return { area: area as RouteArea };
  return { area: "ledgers" };
}

function currentRoute(): Route {
  return parseRoute(window.location.hash);
}

export function useRoute() {
  const [route, setRoute] = useState<Route>(currentRoute);
  useEffect(() => {
    const update = () => setRoute(currentRoute());
    window.addEventListener("hashchange", update);
    return () => window.removeEventListener("hashchange", update);
  }, []);
  return route;
}
