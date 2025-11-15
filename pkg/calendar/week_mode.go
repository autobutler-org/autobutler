package calendar

type WeekMode int

const (
	WeekModeStandard WeekMode = iota // Week starts on Sunday
	WeekModeISO                      // Week starts on Monday
)
