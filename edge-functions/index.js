import { notFoundHTML } from "./_shared/http.js";

export default function onRequest() {
  return notFoundHTML();
}
