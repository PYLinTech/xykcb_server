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

type courseRecord struct {
	RawID    string
	Name     string
	Location string
	Teacher  string
	Weeks    []int
	Schedule map[string][]int
}

func convertCourse(c map[string]interface{}) courseRecord {
	return courseRecord{
		RawID:    safeString(c["kch"], ""),
		Name:     safeString(c["courseName"], ""),
		Location: cleanLocation(safeString(c["location"], "")),
		Teacher:  safeString(c["teacherName"], ""),
		Weeks:    parseWeeks(safeString(c["classWeekDetails"], "")),
		Schedule: parseClassTime(safeString(c["classTime"], "")),
	}
}

func stableGroupByRawID(courses []courseRecord) []courseRecord {
	idMap := make(map[string][]courseRecord)
	idOrder := make([]string, 0, len(courses))
	for _, course := range courses {
		if _, ok := idMap[course.RawID]; !ok {
			idOrder = append(idOrder, course.RawID)
		}
		idMap[course.RawID] = append(idMap[course.RawID], course)
	}
	result := make([]courseRecord, 0, len(courses))
	for _, id := range idOrder {
		result = append(result, idMap[id]...)
	}
	return result
}
