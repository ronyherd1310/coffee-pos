package auth

import "time"

const maxSessionDuration = 12 * time.Hour

func SessionExpiry(loginAt time.Time, location *time.Location) time.Time {
	localLoginTime := loginAt.In(location)
	afterTwelveHours := localLoginTime.Add(maxSessionDuration)
	endOfDay := time.Date(
		localLoginTime.Year(),
		localLoginTime.Month(),
		localLoginTime.Day()+1,
		0,
		0,
		0,
		0,
		location,
	)

	if afterTwelveHours.Before(endOfDay) {
		return afterTwelveHours
	}

	return endOfDay
}
