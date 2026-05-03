package schools

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var locationRe = regexp.MustCompile(`\(.*?\)|（.*?）`)

func cleanLocation(location string) string {
	return locationRe.ReplaceAllString(location, "")
}

func parseClassTime(ct string) map[string][]int {
	if len(ct) < 3 {
		return nil
	}
	schedule := make(map[string][]int, 1)
	weekday, sections := string(ct[0]), ct[1:]
	schedule[weekday] = make([]int, 0, len(sections)/2)
	for i := 0; i+1 < len(sections); i += 2 {
		section, _ := strconv.Atoi(sections[i : i+2])
		schedule[weekday] = append(schedule[weekday], section)
	}
	return schedule
}

func parseWeeks(wd string) []int {
	fields := strings.Split(wd, ",")
	weeks := make([]int, 0, len(fields))
	for _, p := range fields {
		if p == "" {
			continue
		}
		if w, err := strconv.Atoi(p); err == nil {
			weeks = append(weeks, w)
		}
	}
	return weeks
}

func getStringField(course map[string]interface{}, key string) string {
	return safeString(course[key], "")
}

func getIntSliceField(course map[string]interface{}, key string) []int {
	values, ok := course[key].([]int)
	if !ok {
		return nil
	}
	return values
}

func getScheduleField(course map[string]interface{}) map[string][]int {
	schedule, ok := course["schedule"].(map[string][]int)
	if !ok {
		return nil
	}
	return schedule
}

func genCourseHash(course map[string]interface{}) string {
	key := getStringField(course, "rawId") + "|" + getStringField(course, "name") + "|" + getStringField(course, "location") + "|" + getStringField(course, "teacher") + "|" + weeksKey(getIntSliceField(course, "weeks"))
	hash := int32(0)
	for _, r := range key {
		hash = int32((int64(hash)<<5 - int64(hash)) + int64(r))
	}
	value := int64(hash)
	if value < 0 {
		value = -value
	}
	return strconv.FormatInt(value, 36)
}

func weeksKey(weeks []int) string {
	if len(weeks) == 0 {
		return ""
	}
	sorted := append([]int(nil), weeks...)
	sort.Ints(sorted)
	parts := make([]string, 0, len(sorted))
	prev := 0
	for i, week := range sorted {
		if i > 0 && week == prev {
			continue
		}
		parts = append(parts, strconv.Itoa(week))
		prev = week
	}
	return strings.Join(parts, ",")
}

func mergeCourseSchedule(target, source map[string]interface{}) {
	targetSchedule := getScheduleField(target)
	if targetSchedule == nil {
		targetSchedule = make(map[string][]int)
		target["schedule"] = targetSchedule
	}

	for day, sections := range getScheduleField(source) {
		daySet := make(map[int]struct{}, len(targetSchedule[day])+len(sections))
		for _, section := range targetSchedule[day] {
			daySet[section] = struct{}{}
		}
		for _, section := range sections {
			if _, ok := daySet[section]; !ok {
				targetSchedule[day] = append(targetSchedule[day], section)
				daySet[section] = struct{}{}
			}
		}
	}

	for day := range targetSchedule {
		sort.Ints(targetSchedule[day])
	}
}

func resolveDuplicateCourseIDs(courses []map[string]interface{}) []map[string]interface{} {
	idMap := make(map[string][]map[string]interface{})
	idOrder := make([]string, 0)
	for _, course := range courses {
		id := getStringField(course, "rawId")
		if _, ok := idMap[id]; !ok {
			idOrder = append(idOrder, id)
		}
		idMap[id] = append(idMap[id], course)
	}

	processed := make([]map[string]interface{}, 0, len(courses))
	for _, id := range idOrder {
		group := idMap[id]
		if len(group) == 1 {
			group[0]["id"] = genCourseHash(group[0])
			processed = append(processed, group[0])
			continue
		}

		diffCourses := make([]map[string]interface{}, 0)
		sameBasicCourses := make([]map[string]interface{}, 0)
		base := group[0]
		baseKey := getStringField(base, "name") + "|" + getStringField(base, "location") + "|" + getStringField(base, "teacher")

		for i := 1; i < len(group); i++ {
			course := group[i]
			key := getStringField(course, "name") + "|" + getStringField(course, "location") + "|" + getStringField(course, "teacher")
			if key == baseKey {
				sameBasicCourses = append(sameBasicCourses, course)
			} else {
				diffCourses = append(diffCourses, course)
			}
		}

		for _, course := range diffCourses {
			course["id"] = genCourseHash(course)
			processed = append(processed, course)
		}

		if len(sameBasicCourses) == 0 {
			base["id"] = genCourseHash(base)
			processed = append(processed, base)
			continue
		}

		weeksMap := make(map[string][]map[string]interface{})
		weeksOrder := make([]string, 0)
		baseWeeksKey := weeksKey(getIntSliceField(base, "weeks"))
		weeksMap[baseWeeksKey] = []map[string]interface{}{base}
		weeksOrder = append(weeksOrder, baseWeeksKey)

		for _, course := range sameBasicCourses {
			key := weeksKey(getIntSliceField(course, "weeks"))
			if _, ok := weeksMap[key]; !ok {
				weeksOrder = append(weeksOrder, key)
			}
			weeksMap[key] = append(weeksMap[key], course)
		}

		for _, key := range weeksOrder {
			coursesGroup := weeksMap[key]
			first := coursesGroup[0]
			if len(coursesGroup) == 1 {
				first["id"] = genCourseHash(first)
				processed = append(processed, first)
				continue
			}

			for i := 1; i < len(coursesGroup); i++ {
				mergeCourseSchedule(first, coursesGroup[i])
			}
			first["id"] = genCourseHash(first)
			processed = append(processed, first)
		}
	}

	return processed
}

func convertCourse(c map[string]interface{}) map[string]interface{} {
	rawID := safeString(c["kch"], "")
	course := map[string]interface{}{
		"rawId":    rawID,
		"name":     safeString(c["courseName"], ""),
		"location": cleanLocation(safeString(c["location"], "")),
		"teacher":  safeString(c["teacherName"], ""),
		"weeks":    parseWeeks(safeString(c["classWeekDetails"], "")),
		"schedule": parseClassTime(safeString(c["classTime"], "")),
	}
	course["id"] = genCourseHash(course)
	return course
}
