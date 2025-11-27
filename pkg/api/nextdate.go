package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// NextDate
func  NextDate(now time.Time, dstart string, repeat string) (string, error) {
	date, err := time.Parse(dateLayout, dstart)
	if err != nil {
		return "", err
	}

	interval := strings.Split(repeat, " ")
	rule := interval[0]

	var nextDate string

	switch rule {
	case "d":
		if len(interval) < 2 {
			return "", fmt.Errorf("invalid d rule")
		}
		s := interval[1]
		num, err := strconv.Atoi(s)
		if err != nil {
			return "", err
		}
		if num > 400 || num < 1 {
			return "", fmt.Errorf("interval days out of range")
		}

		current := date
		if afterNow(current, now) {
			current = current.AddDate(0, 0, num)
		} else {
			for !afterNow(current, now) {
				current = current.AddDate(0, 0, num)
			}
		}

		nextDate = current.Format(dateLayout)
	case "y":
		current := date
		if afterNow(current, now) {
			current = current.AddDate(1, 0, 0)
		} else {
			for !afterNow(current, now) {
				current = current.AddDate(1, 0, 0)
			}
		}

		nextDate = current.Format(dateLayout)
	case "w":
		if len(interval) < 2 {
			return "", fmt.Errorf("invalid w rule")
		}
		weekdays := strings.Split(interval[1], ",")
		if len(weekdays) == 0 {
			return "", fmt.Errorf("empty weekdays")
		}

		targetWeekdays := make(map[time.Weekday]bool)
		for _, w := range weekdays {
			num, err := strconv.Atoi(w)
			if err != nil {
				return "", err
			}
			if num < 1 || num > 7 {
				return "", fmt.Errorf("invalid weekday: %d", num)
			}

			if num == 7 {
				targetWeekdays[time.Sunday] = true
			} else {
				targetWeekdays[time.Weekday(num)] = true
			}
		}

		current := now.AddDate(0, 0, 1)
		for !targetWeekdays[current.Weekday()] {
			current = current.AddDate(0, 0, 1)
		}
		nextDate = current.Format(dateLayout)

	case "m":
		if len(interval) < 2 {
			return "", fmt.Errorf("invalid m rule")
		}

		daysPart := interval[1]
		var monthsPart string
		if len(interval) > 2 {
			monthsPart = interval[2]
		}

		days, err := parseDays(daysPart)
		if err != nil {
			return "", err
		}

		months, err := parseMonths(monthsPart)
		if err != nil {
			return "", err
		}

		current := date
		for {
			if afterNow(current, now) && isDayInList(current, days) && isMonthInList(int(current.Month()), months) {
				break
			}
			current = current.AddDate(0, 0, 1)
		}
		nextDate = current.Format(dateLayout)

	default:
		return "", fmt.Errorf("unknown rule: %s", rule)
	}

	return nextDate, nil
}

func afterNow(date, now time.Time) bool {
	return date.After(now)
}

func parseDays(daysStr string) ([]int, error) {
	daysList := strings.Split(daysStr, ",")
	var days []int
	for _, d := range daysList {
		day, err := strconv.Atoi(d)
		if err != nil {
			return nil, err
		}
		if day < -2 || day > 31 || day == 0 {
			return nil, fmt.Errorf("invalid day: %d", day)
		}
		days = append(days, day)
	}
	return days, nil
}

func parseMonths(monthsStr string) ([]int, error) {
	if monthsStr == "" {
		return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, nil
	}

	monthsList := strings.Split(monthsStr, ",")
	var months []int
	for _, m := range monthsList {
		month, err := strconv.Atoi(m)
		if err != nil {
			return nil, err
		}
		if month < 1 || month > 12 {
			return nil, fmt.Errorf("invalid month: %d", month)
		}
		months = append(months, month)
	}
	return months, nil
}

func isMonthInList(month int, months []int) bool {
	for _, m := range months {
		if m == month {
			return true
		}
	}
	return false
}

func isDayInList(date time.Time, days []int) bool {
	day := date.Day()
	for _, d := range days {
		if d == day {
			return true
		}
		if d == -1 && isLastDayOfMonth(date) {
			return true
		}
		if d == -2 && isSecondLastDayOfMonth(date) {
			return true
		}
	}
	return false
}

func isLastDayOfMonth(date time.Time) bool {
	nextDay := date.AddDate(0, 0, 1)
	return nextDay.Month() != date.Month()
}

func isSecondLastDayOfMonth(date time.Time) bool {
	nextDay := date.AddDate(0, 0, 1)
	return isLastDayOfMonth(nextDay)
}

