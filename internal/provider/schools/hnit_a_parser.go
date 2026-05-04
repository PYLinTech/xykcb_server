package schools

import (
	"regexp"
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

func orderByRawID(courses []map[string]interface{}) []map[string]interface{} {
	idMap := make(map[string][]map[string]interface{})
	idOrder := make([]string, 0)
	for _, course := range courses {
		id := getStringField(course, "rawId")
		if _, ok := idMap[id]; !ok {
			idOrder = append(idOrder, id)
		}
		idMap[id] = append(idMap[id], course)
	}
	result := make([]map[string]interface{}, 0, len(courses))
	for _, id := range idOrder {
		result = append(result, idMap[id]...)
	}
	return result
}

func convertCourse(c map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"rawId":    safeString(c["kch"], ""),
		"name":     safeString(c["courseName"], ""),
		"location": cleanLocation(safeString(c["location"], "")),
		"teacher":  safeString(c["teacherName"], ""),
		"weeks":    parseWeeks(safeString(c["classWeekDetails"], "")),
		"schedule": parseClassTime(safeString(c["classTime"], "")),
	}
}
