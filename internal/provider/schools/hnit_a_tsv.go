package schools

import (
	"encoding/binary"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	"xykcb_server/internal/config"
)

func murmurHash3x86_32(data []byte, seed uint32) uint32 {
	h := seed
	nblocks := len(data) / 4

	const c1 = 0xcc9e2d51
	const c2 = 0x1b873593

	for i := 0; i < nblocks; i++ {
		k := binary.LittleEndian.Uint32(data[i*4:])
		k *= c1
		k = bits.RotateLeft32(k, 15)
		k *= c2
		h ^= k
		h = bits.RotateLeft32(h, 13)
		h = h*5 + 0xe6546b64
	}

	tail := data[nblocks*4:]
	var k1 uint32
	switch len(tail) {
	case 3:
		k1 ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint32(tail[0])
		k1 *= c1
		k1 = bits.RotateLeft32(k1, 15)
		k1 *= c2
		h ^= k1
	}

	h ^= uint32(len(data))
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16

	return h
}

func genCourseTSVHash(termID, rawID, courseName, location, teacher, weeks, weekday, sections string) string {
	input := strings.Join([]string{termID, rawID, courseName, location, teacher, weeks, weekday, sections}, "\t")
	hash := murmurHash3x86_32([]byte(input), 0)
	s := strconv.FormatUint(uint64(hash), 36)
	if len(s) < 8 {
		s = strings.Repeat("0", 8-len(s)) + s
	}
	return s
}

func nn(v string) string {
	if v == "" {
		return `\N`
	}
	return v
}

func joinSortedInts(vals []int) string {
	vals = append([]int(nil), vals...)
	sort.Ints(vals)
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

func buildPeriodGroups(mergeableSections []string) []string {
	groups := make([]string, 0)
	for pgIdx, ms := range mergeableSections {
		for range strings.Split(ms, "-") {
			groups = append(groups, strconv.Itoa(pgIdx+1))
		}
	}
	return groups
}

type tsvTermsRow struct {
	schoolID         string
	termID           string
	totalWeeks       string
	startDate        string
	periodGroup      string
	sectionNo        string
	sectionStartTime string
	sectionEndTime   string
}

func (r tsvTermsRow) toLine() string {
	return strings.Join([]string{
		r.schoolID, r.termID, r.totalWeeks, r.startDate,
		r.periodGroup, r.sectionNo, r.sectionStartTime, r.sectionEndTime,
	}, "\t")
}

func writeTermsRow(sb *strings.Builder, row tsvTermsRow, prev *tsvTermsRow, first bool) {
	if first {
		out := row
		out.startDate = nn(row.startDate)
		sb.WriteString(out.toLine())
		sb.WriteByte('\n')
		*prev = row
		return
	}

	out := tsvTermsRow{
		sectionNo:        row.sectionNo,
		sectionStartTime: row.sectionStartTime,
		sectionEndTime:   row.sectionEndTime,
	}
	if row.schoolID != prev.schoolID {
		out.schoolID = row.schoolID
	}
	if row.termID != prev.termID {
		out.termID = row.termID
	}
	if row.totalWeeks != prev.totalWeeks {
		out.totalWeeks = row.totalWeeks
	}
	if row.startDate != prev.startDate {
		out.startDate = nn(row.startDate)
	}
	if row.periodGroup != prev.periodGroup {
		out.periodGroup = row.periodGroup
	}
	*prev = row
	sb.WriteString(out.toLine())
	sb.WriteByte('\n')
}

func buildTermsTSV(schoolID string, semesterIDs []string, semesters map[string]semData) string {
	var sb strings.Builder
	sb.WriteString("@terms\n")
	sb.WriteString("school_id\tterm_id\ttotal_weeks\tstart_date\tperiod_group\tsection_no\tsection_start_time\tsection_end_time\n")

	var prev tsvTermsRow
	first := true

	for _, semesterID := range semesterIDs {
		d, ok := semesters[semesterID]
		if !ok {
			continue
		}
		cfg := d.cfg
		groups := buildPeriodGroups(cfg.MergeableSections)

		for i, ts := range cfg.TimeSlots {
			pg := ""
			if i < len(groups) {
				pg = groups[i]
			}

			row := tsvTermsRow{
				schoolID:         schoolID,
				termID:           semesterID,
				totalWeeks:       strconv.Itoa(cfg.TotalWeeks),
				startDate:        cfg.SemesterStart,
				periodGroup:      pg,
				sectionNo:        strconv.Itoa(ts.Section),
				sectionStartTime: ts.Start,
				sectionEndTime:   ts.End,
			}

			writeTermsRow(&sb, row, &prev, first)
			first = false
		}
	}

	return sb.String()
}

type tsvCourseRow struct {
	cHash    string
	termID   string
	rawID    string
	name     string
	location string
	teacher  string
	weeks    string
	weekday  string
	sections string
}

func (r tsvCourseRow) toLine() string {
	return strings.Join([]string{
		r.cHash, r.termID, r.rawID, r.name,
		r.location, r.teacher, r.weeks, r.weekday, r.sections,
	}, "\t")
}

func makeCourseRow(semesterID, rawID, courseName, location, teacher, rawWeeksStr, wd, sectionsStr string) tsvCourseRow {
	row := tsvCourseRow{
		termID:   semesterID,
		rawID:    nn(rawID),
		name:     nn(courseName),
		location: nn(location),
		teacher:  nn(teacher),
		weeks:    nn(rawWeeksStr),
		weekday:  nn(wd),
		sections: nn(sectionsStr),
	}
	row.cHash = genCourseTSVHash(semesterID, rawID, courseName, location, teacher, rawWeeksStr, wd, sectionsStr)
	return row
}

func writeCourseRow(sb *strings.Builder, row tsvCourseRow, prev *tsvCourseRow, first bool) {
	if first {
		sb.WriteString(row.toLine())
		sb.WriteByte('\n')
		*prev = row
		return
	}

	inherited := tsvCourseRow{
		cHash:    row.cHash,
		weekday:  row.weekday,
		sections: row.sections,
	}
	if row.termID != prev.termID {
		inherited.termID = row.termID
	}
	if row.rawID != prev.rawID {
		inherited.rawID = row.rawID
	}
	if row.name != prev.name {
		inherited.name = row.name
	}
	if row.location != prev.location {
		inherited.location = row.location
	}
	if row.teacher != prev.teacher {
		inherited.teacher = row.teacher
	}
	if row.weeks != prev.weeks {
		inherited.weeks = row.weeks
	}
	*prev = row
	sb.WriteString(inherited.toLine())
	sb.WriteByte('\n')
}

func buildCoursesTSV(semesterIDs []string, semesters map[string]semData) string {
	var sb strings.Builder
	sb.WriteString("@courses\n")
	sb.WriteString("c_hash\tterm_id\traw_id\tcourse_name\tlocation\tteacher\tweeks\tweekday\tsections\n")

	var prev tsvCourseRow
	first := true

	for _, semesterID := range semesterIDs {
		for _, course := range semesters[semesterID].courses {
			rawWeeksStr := joinSortedInts(course.Weeks)

			if len(course.Schedule) == 0 {
				row := makeCourseRow(semesterID, course.RawID, course.Name, course.Location, course.Teacher, rawWeeksStr, "", "")
				writeCourseRow(&sb, row, &prev, first)
				first = false
				continue
			}

			weekdays := make([]string, 0, len(course.Schedule))
			for wd := range course.Schedule {
				weekdays = append(weekdays, wd)
			}
			sort.Strings(weekdays)

			for _, wd := range weekdays {
				row := makeCourseRow(semesterID, course.RawID, course.Name, course.Location, course.Teacher, rawWeeksStr, wd, joinSortedInts(course.Schedule[wd]))
				writeCourseRow(&sb, row, &prev, first)
				first = false
			}
		}
	}

	return sb.String()
}

type semData struct {
	cfg     config.SemesterConfig
	courses []courseRecord
}

func generateCourseTSV(schoolID string, data map[string]semData) string {
	semesterIDs := make([]string, 0, len(data))
	for semID := range data {
		semesterIDs = append(semesterIDs, semID)
	}
	sort.Slice(semesterIDs, func(i, j int) bool {
		return semesterSortKey(semesterIDs[i]) < semesterSortKey(semesterIDs[j])
	})

	var sb strings.Builder
	sb.WriteString(buildTermsTSV(schoolID, semesterIDs, data))
	sb.WriteString("\n")
	sb.WriteString(buildCoursesTSV(semesterIDs, data))
	return sb.String()
}
