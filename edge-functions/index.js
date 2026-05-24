import { notFoundHTML, preflight } from "./_shared/http.js";

export default function onRequest(context) {
  if (context.request.method === "OPTIONS") return preflight();
  return notFoundHTML();
}
