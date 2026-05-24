import { NOT_FOUND_HTML } from "./not-found.js";

const ERRORS = {
  "001": { status: 400 },
  "002": { status: 404 },
  "003": { status: 401 },
  "004": { status: 500 },
};

export function ok(data) {
  return json({ success: true, data }, 200);
}

export function error(descKey) {
  return json({ success: false, desc_key: descKey }, ERRORS[descKey]?.status || 500);
}

export function notFoundHTML() {
  return new Response(NOT_FOUND_HTML, {
    status: 404,
    headers: headers({ "Content-Type": "text/html; charset=utf-8" }),
  });
}

export function preflight() {
  return new Response(null, { status: 204, headers: headers() });
}

export function appError(descKey) {
  const err = new Error(descKey);
  err.descKey = descKey;
  return err;
}

export function providerError(message) {
  return error(String(message || "").includes("密码错误") ? "003" : "004");
}

function json(body, status) {
  return new Response(JSON.stringify(body), {
    status,
    headers: headers({ "Content-Type": "application/json; charset=utf-8" }),
  });
}

function headers(extra = {}) {
  return {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Methods": "GET, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type",
    ...extra,
  };
}
