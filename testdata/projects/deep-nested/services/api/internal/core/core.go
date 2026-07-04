package core

// Greeting proves code lives several directories deep — the layout that the
// old scanner (which only looked at the root and cmd/*) could not discover.
func Greeting() string {
	return "deep-nested up"
}
