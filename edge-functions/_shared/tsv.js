export function convertCourse(course) {
  return {
    rawID: safeString(course?.kch),
    name: safeString(course?.courseName),
    location: safeString(course?.location).replace(/\(.*?\)|（.*?）/g, ""),
    teacher: safeString(course?.teacherName),
    weeks: parseWeeks(safeString(course?.classWeekDetails)),
    schedule: parseClassTime(safeString(course?.classTime)),
  };
}

export function stableGroupByRawID(courses) {
  const groups = new Map();
  for (const course of courses) {
    if (!groups.has(course.rawID)) groups.set(course.rawID, []);
    groups.get(course.rawID).push(course);
  }
  return [...groups.values()].flat();
}

export function generateCourseTSV(schoolID, data) {
  const semesterIDs = Object.keys(data).sort((a, b) => semesterSortKey(a) - semesterSortKey(b));
  return `${buildTermsTSV(schoolID, semesterIDs, data)}\n${buildCoursesTSV(semesterIDs, data)}`;
}

export function semesterSortKey(semesterID) {
  return Number(String(semesterID || "").replaceAll("-", "")) || 0;
}

export function safeString(value) {
  return typeof value === "string" ? value : "";
}

function parseClassTime(value) {
  if (value.length < 3) return {};
  const weekday = value[0];
  const sections = [];
  for (let i = 1; i + 1 < value.length; i += 2) sections.push(Number(value.slice(i, i + 2)) || 0);
  return { [weekday]: sections };
}

function parseWeeks(value) {
  return value.split(",").map((item) => Number(item)).filter(Boolean);
}

function buildTermsTSV(schoolID, semesterIDs, data) {
  const lines = ["@terms", "school_id\tterm_id\ttotal_weeks\tstart_date\tperiod_group\tsection_no\tsection_start_time\tsection_end_time"];
  let prev = null;
  for (const semesterID of semesterIDs) {
    const cfg = data[semesterID].cfg;
    const groups = buildPeriodGroups(cfg.mergeableSections || cfg.MergeableSections || []);
    for (const [i, ts] of (cfg.timeSlots || cfg.TimeSlots || []).entries()) {
      const row = {
        schoolID,
        termID: semesterID,
        totalWeeks: String(cfg.totalWeeks || cfg.TotalWeeks || ""),
        startDate: cfg.semesterStart || cfg.SemesterStart || "",
        periodGroup: groups[i] || "",
        sectionNo: String(ts.section || ts.Section || ""),
        sectionStartTime: ts.start || ts.Start || "",
        sectionEndTime: ts.end || ts.End || "",
      };
      const fields = ["schoolID", "termID", "totalWeeks", "startDate", "periodGroup", "sectionNo", "sectionStartTime", "sectionEndTime"];
      const out = prev
        ? fields.map((key) => (["sectionNo", "sectionStartTime", "sectionEndTime"].includes(key) || row[key] !== prev[key] ? nn(row[key]) : ""))
        : [row.schoolID, row.termID, row.totalWeeks, nn(row.startDate), row.periodGroup, row.sectionNo, row.sectionStartTime, row.sectionEndTime];
      lines.push(out.join("\t"));
      prev = row;
    }
  }
  return lines.join("\n") + "\n";
}

function buildCoursesTSV(semesterIDs, data) {
  const lines = ["@courses", "c_hash\tterm_id\traw_id\tcourse_name\tlocation\tteacher\tweeks\tweekday\tsections"];
  let prev = null;
  for (const semesterID of semesterIDs) {
    for (const course of data[semesterID].courses) {
      const weeks = joinSortedInts(course.weeks);
      const weekdays = Object.keys(course.schedule).sort();
      const entries = weekdays.length ? weekdays : [""];
      for (const weekday of entries) {
        const sections = weekday ? joinSortedInts(course.schedule[weekday]) : "";
        const row = makeCourseRow(semesterID, course.rawID, course.name, course.location, course.teacher, weeks, weekday, sections);
        const out = prev
          ? [row.cHash, row.termID !== prev.termID ? row.termID : "", row.rawID !== prev.rawID ? row.rawID : "", row.name !== prev.name ? row.name : "", row.location !== prev.location ? row.location : "", row.teacher !== prev.teacher ? row.teacher : "", row.weeks !== prev.weeks ? row.weeks : "", row.weekday, row.sections]
          : [row.cHash, row.termID, row.rawID, row.name, row.location, row.teacher, row.weeks, row.weekday, row.sections];
        lines.push(out.join("\t"));
        prev = row;
      }
    }
  }
  return lines.join("\n");
}

function makeCourseRow(termID, rawID, name, location, teacher, weeks, weekday, sections) {
  const row = {
    termID,
    rawID: nn(rawID),
    name: nn(name),
    location: nn(location),
    teacher: nn(teacher),
    weeks: nn(weeks),
    weekday: nn(weekday),
    sections: nn(sections),
  };
  row.cHash = murmurHashBase36([termID, rawID, name, location, teacher, weeks, weekday, sections].join("\t"));
  return row;
}

function buildPeriodGroups(mergeableSections) {
  return mergeableSections.flatMap((item, idx) => String(item).split("-").map(() => String(idx + 1)));
}

function murmurHashBase36(input) {
  const data = new TextEncoder().encode(input);
  let h = 0;
  const c1 = 0xcc9e2d51;
  const c2 = 0x1b873593;
  let i = 0;
  for (; i + 4 <= data.length; i += 4) {
    let k = (data[i] | (data[i + 1] << 8) | (data[i + 2] << 16) | (data[i + 3] << 24)) >>> 0;
    k = Math.imul(k, c1);
    k = rotl32(k, 15);
    k = Math.imul(k, c2);
    h ^= k;
    h = rotl32(h, 13);
    h = (Math.imul(h, 5) + 0xe6546b64) >>> 0;
  }
  let k = 0;
  if (i < data.length) {
    if (i + 2 < data.length) k ^= data[i + 2] << 16;
    if (i + 1 < data.length) k ^= data[i + 1] << 8;
    k ^= data[i];
    k = Math.imul(k, c1);
    k = rotl32(k, 15);
    k = Math.imul(k, c2);
    h ^= k;
  }
  h ^= data.length;
  h ^= h >>> 16;
  h = Math.imul(h, 0x85ebca6b);
  h ^= h >>> 13;
  h = Math.imul(h, 0xc2b2ae35);
  h ^= h >>> 16;
  return (h >>> 0).toString(36).padStart(8, "0");
}

function rotl32(value, bits) {
  return ((value << bits) | (value >>> (32 - bits))) >>> 0;
}

function joinSortedInts(values) {
  return [...values].sort((a, b) => a - b).join(",");
}

function nn(value) {
  return value === "" ? "\\N" : String(value);
}
