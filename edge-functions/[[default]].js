import { courseData, grades, guidanceTeaching } from "./_shared/hnit-a.js";
import { error, notFoundHTML, ok } from "./_shared/http.js";
import { getSupportSchools, SCHOOLS } from "./_shared/schools.js";

const routes = new Map([
  ["/get-support-school", () => ok(getSupportSchools())],
  ["/get-support-function", supportFunctions],
  ["/get-course-data", courseData],
  ["/get-course-grades", grades],
  ["/get-guidance-teaching", guidanceTeaching],
]);

export default async function onRequest(context) {
  const request = context.request;
  if (request.method !== "GET") return error("001");

  const url = new URL(request.url);
  const handler = routes.get(url.pathname);
  if (!handler) return notFoundHTML();

  try {
    return await handler(url);
  } catch (err) {
    return error(err?.descKey || "004");
  }
}

function supportFunctions(url) {
  const school = url.searchParams.get("school");
  if (!school) return error("001");
  if (!SCHOOLS[school]) return error("002");
  return ok(SCHOOLS[school].functions || []);
}
