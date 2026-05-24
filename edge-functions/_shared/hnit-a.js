import { appError, error, ok, providerError } from "./http.js";
import { SCHOOLS } from "./schools.js";
import { convertCourse, generateCourseTSV, safeString, semesterSortKey, stableGroupByRawID } from "./tsv.js";

const BASE_URL = "https://jw.hnit.edu.cn/njwhd";
const HTTP_TIMEOUT_MS = 10 * 1000;
const AES_KEY = hexToBytes("717a6b6a316b6a6768643d383736262a");
const AES_SBOX = [
  99, 124, 119, 123, 242, 107, 111, 197, 48, 1, 103, 43, 254, 215, 171, 118, 202, 130, 201, 125, 250, 89, 71, 240, 173, 212, 162, 175, 156, 164, 114, 192,
  183, 253, 147, 38, 54, 63, 247, 204, 52, 165, 229, 241, 113, 216, 49, 21, 4, 199, 35, 195, 24, 150, 5, 154, 7, 18, 128, 226, 235, 39, 178, 117,
  9, 131, 44, 26, 27, 110, 90, 160, 82, 59, 214, 179, 41, 227, 47, 132, 83, 209, 0, 237, 32, 252, 177, 91, 106, 203, 190, 57, 74, 76, 88, 207,
  208, 239, 170, 251, 67, 77, 51, 133, 69, 249, 2, 127, 80, 60, 159, 168, 81, 163, 64, 143, 146, 157, 56, 245, 188, 182, 218, 33, 16, 255, 243, 210,
  205, 12, 19, 236, 95, 151, 68, 23, 196, 167, 126, 61, 100, 93, 25, 115, 96, 129, 79, 220, 34, 42, 144, 136, 70, 238, 184, 20, 222, 94, 11, 219,
  224, 50, 58, 10, 73, 6, 36, 92, 194, 211, 172, 98, 145, 149, 228, 121, 231, 200, 55, 109, 141, 213, 78, 169, 108, 86, 244, 234, 101, 122, 174, 8,
  186, 120, 37, 46, 28, 166, 180, 198, 232, 221, 116, 31, 75, 189, 139, 138, 112, 62, 181, 102, 72, 3, 246, 14, 97, 53, 87, 185, 134, 193, 29, 158,
  225, 248, 152, 17, 105, 217, 142, 148, 155, 30, 135, 233, 206, 85, 40, 223, 140, 161, 137, 13, 191, 230, 66, 104, 65, 153, 45, 15, 176, 84, 187, 22,
];
const AES_RCON = [1, 2, 4, 8, 16, 32, 64, 128, 27, 54];

export async function courseData(url) {
  const input = readInput(url);
  const rejected = rejectInput(input);
  if (rejected) return rejected;

  const client = await createClient(input.account, input.password);
  const semesters = await getSemesterConfigs(client, SCHOOLS.hnit_a);
  if (Object.keys(semesters).length === 0) return error("004");

  const collected = {};
  await Promise.all(
    Object.entries(semesters).map(async ([semesterID, cfg]) => {
      const payload = await client.get(
        `/student/curriculum?token=${encodeURIComponent(client.token)}&xnxq01id=${encodeURIComponent(semesterID)}&week=all`,
      );
      collected[semesterID] = { cfg, courses: parseCourses(payload) };
    }),
  );

  return ok(generateCourseTSV("hnit_a", collected));
}

export async function grades(url) {
  const input = readInput(url);
  const rejected = rejectInput(input);
  if (rejected) return rejected;

  const client = await createClient(input.account, input.password);
  const semesterList = await client.get(`/semesterList?token=${encodeURIComponent(client.token)}`);
  if (safeString(semesterList?.code) !== "1") return providerError(semesterList?.Msg);

  const semester = url.searchParams.get("semester") || "";
  const gradesData = await client.get(
    `/student/termGPA?token=${encodeURIComponent(client.token)}&semester=${encodeURIComponent(semester)}`,
  );
  if (safeString(gradesData?.code) !== "1") return providerError(gradesData?.Msg);

  return ok({ "all-semester": semesterList.data, "all-grades": gradesData.data });
}

export async function guidanceTeaching(url) {
  const input = readInput(url);
  const rejected = rejectInput(input);
  if (rejected) return rejected;

  const client = await createClient(input.account, input.password);
  const payload = await client.get(`/student/guidanceTeaching?token=${encodeURIComponent(client.token)}`);
  if (safeString(payload?.code) !== "1") return providerError(payload?.Msg);
  return ok(payload.data);
}

function readInput(url) {
  return {
    school: url.searchParams.get("school") || "",
    account: url.searchParams.get("account") || "",
    password: url.searchParams.get("password") || "",
  };
}

function rejectInput({ school, account, password }) {
  if (!school || !account || !password) return error("001");
  if (school === "hnit_a") return null;
  return SCHOOLS[school] ? error("004") : error("002");
}

async function createClient(account, password) {
  const client = {
    account,
    password,
    token: "",
    async get(path) {
      let payload = await getJSON(path);
      const code = safeString(payload?.code);
      if (code && code !== "1") {
        this.token = await login(this.account, this.password);
        payload = await getJSON(replaceToken(path, this.token));
      }
      return payload;
    },
  };
  client.token = await login(account, password);
  return client;
}

async function login(account, password) {
  const encrypted = await encryptPassword(password);
  const payload = await postJSON(`/login?userNo=${encodeURIComponent(account)}&pwd=${encodeURIComponent(encrypted)}`);
  if (safeString(payload?.code) !== "1") {
    throw appError(String(payload?.Msg || "login failed").includes("密码错误") ? "003" : "004");
  }

  const token = safeString(payload?.data?.token);
  if (!token) throw appError("004");
  return token;
}

async function getSemesterConfigs(client, cfg) {
  if (!cfg.semesterConfigFrom?.length) return cfg.semesters || {};

  const semesterIDs = await getSemesterIDs(client);
  const semesters = {};
  await Promise.all(
    semesterIDs.map(async (semesterID) => {
      const rule = selectSemesterConfigFrom(semesterID, cfg.semesterConfigFrom);
      if (!rule) return;
      semesters[semesterID] = {
        semesterStart: await getSemesterStart(client, semesterID),
        totalWeeks: rule.totalWeeks,
        timeSlots: rule.timeSlots,
        mergeableSections: rule.mergeableSections,
      };
    }),
  );
  return semesters;
}

async function getSemesterIDs(client) {
  try {
    return await fetchSemesterIDs(client.token);
  } catch {
    client.token = await login(client.account, client.password);
    return fetchSemesterIDs(client.token);
  }
}

async function fetchSemesterIDs(token) {
  const resp = await fetchWithTimeout(`${BASE_URL}/getXnxqList?${new URLSearchParams({ token }).toString()}`);
  if (resp.status >= 500) throw appError("004");
  const payload = await resp.json();
  if (!Array.isArray(payload)) throw appError("004");
  return payload.map((item) => safeString(item?.xnxq01id)).filter(Boolean);
}

async function getSemesterStart(client, semesterID) {
  const payload = await client.get(
    `/student/curriculum?token=${encodeURIComponent(client.token)}&xnxq01id=${encodeURIComponent(semesterID)}&week=2`,
  );
  if (safeString(payload?.code) !== "1") throw appError("004");

  const monday = payload?.data?.[0]?.date?.find((item) => Number(item?.xqid) === 1)?.mxrq;
  if (!monday) throw appError("004");

  const date = new Date(`${monday}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() - 7);
  return date.toISOString().slice(0, 10);
}

function parseCourses(payload) {
  const courses = [];
  for (const day of Array.isArray(payload?.data) ? payload.data : []) {
    for (const course of Array.isArray(day?.item) ? day.item : []) {
      courses.push(convertCourse(course));
    }
  }
  return stableGroupByRawID(courses);
}

function replaceToken(path, token) {
  const [pathname, query = ""] = path.split("?");
  const params = new URLSearchParams(query);
  params.set("token", token);
  return `${pathname}?${params.toString()}`;
}

function selectSemesterConfigFrom(semesterID, rules) {
  const key = semesterSortKey(semesterID);
  return [...rules]
    .sort((a, b) => semesterSortKey(a.from) - semesterSortKey(b.from))
    .filter((rule) => semesterSortKey(rule.from) <= key)
    .pop();
}

async function getJSON(path) {
  const resp = await fetchWithTimeout(BASE_URL + path, { headers: { Accept: "application/json" } });
  if (resp.status >= 500) throw appError("004");
  return resp.json();
}

async function postJSON(path) {
  const resp = await fetchWithTimeout(BASE_URL + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: "",
  });
  if (resp.status >= 500) throw appError("004");
  return resp.json();
}

async function encryptPassword(password) {
  const plain = pkcs7Pad(new TextEncoder().encode(`"${password}"`), 16);
  const roundKeys = expandAESKey(AES_KEY);
  const encrypted = new Uint8Array(plain.length);

  for (let offset = 0; offset < plain.length; offset += 16) {
    encrypted.set(encryptAESBlock(plain.slice(offset, offset + 16), roundKeys), offset);
  }

  return btoa(btoa(String.fromCharCode(...encrypted)));
}

function hexToBytes(hex) {
  return Uint8Array.from(hex.match(/.{2}/g).map((part) => parseInt(part, 16)));
}

function pkcs7Pad(bytes, blockSize) {
  const padding = blockSize - (bytes.length % blockSize);
  const out = new Uint8Array(bytes.length + padding);
  out.set(bytes);
  out.fill(padding, bytes.length);
  return out;
}

function expandAESKey(key) {
  const out = new Uint8Array(176);
  out.set(key);

  for (let pos = 16, rcon = 0; pos < out.length; pos += 4) {
    const word = [out[pos - 4], out[pos - 3], out[pos - 2], out[pos - 1]];
    if (pos % 16 === 0) {
      word.push(word.shift());
      for (let i = 0; i < 4; i += 1) word[i] = AES_SBOX[word[i]];
      word[0] ^= AES_RCON[rcon];
      rcon += 1;
    }
    for (let i = 0; i < 4; i += 1) out[pos + i] = out[pos + i - 16] ^ word[i];
  }

  return out;
}

function encryptAESBlock(block, roundKeys) {
  const state = Uint8Array.from(block);
  addRoundKey(state, roundKeys, 0);
  for (let round = 1; round < 10; round += 1) {
    subBytes(state);
    shiftRows(state);
    mixColumns(state);
    addRoundKey(state, roundKeys, round * 16);
  }
  subBytes(state);
  shiftRows(state);
  addRoundKey(state, roundKeys, 160);
  return state;
}

function addRoundKey(state, roundKeys, offset) {
  for (let i = 0; i < 16; i += 1) state[i] ^= roundKeys[offset + i];
}

function subBytes(state) {
  for (let i = 0; i < 16; i += 1) state[i] = AES_SBOX[state[i]];
}

function shiftRows(state) {
  const s = Uint8Array.from(state);
  state[1] = s[5]; state[5] = s[9]; state[9] = s[13]; state[13] = s[1];
  state[2] = s[10]; state[6] = s[14]; state[10] = s[2]; state[14] = s[6];
  state[3] = s[15]; state[7] = s[3]; state[11] = s[7]; state[15] = s[11];
}

function mixColumns(state) {
  for (let col = 0; col < 16; col += 4) {
    const a = state.slice(col, col + 4);
    state[col] = gmul2(a[0]) ^ gmul3(a[1]) ^ a[2] ^ a[3];
    state[col + 1] = a[0] ^ gmul2(a[1]) ^ gmul3(a[2]) ^ a[3];
    state[col + 2] = a[0] ^ a[1] ^ gmul2(a[2]) ^ gmul3(a[3]);
    state[col + 3] = gmul3(a[0]) ^ a[1] ^ a[2] ^ gmul2(a[3]);
  }
}

function gmul2(value) {
  return ((value << 1) ^ (value & 128 ? 27 : 0)) & 255;
}

function gmul3(value) {
  return gmul2(value) ^ value;
}

function fetchWithTimeout(url, init = {}) {
  if (typeof AbortSignal !== "undefined" && typeof AbortSignal.timeout === "function") {
    return fetch(url, { ...init, signal: AbortSignal.timeout(HTTP_TIMEOUT_MS) });
  }

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), HTTP_TIMEOUT_MS);
  return fetch(url, { ...init, signal: controller.signal }).finally(() => clearTimeout(timer));
}
