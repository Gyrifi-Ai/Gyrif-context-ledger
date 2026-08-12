import { useEffect, useState } from "react";

export type Route = "ledgers" | "changes" | "proposals" | "releases";

function currentRoute(): Route {
  const route = window.location.hash.slice(1);
  return (["ledgers", "changes", "proposals", "releases"] as Route[]).includes(route as Route) ? route as Route : "ledgers";
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
