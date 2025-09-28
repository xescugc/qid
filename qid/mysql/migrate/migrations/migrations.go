package migrations

// Migration defines a basic struct to write
// the migraions
type Migration struct {
	Name string
	SQL  string
}

// Migrations is a list of all the Migrations
// we have, is defined as an array so it fails
// in compilation time if some order is wrong
// if it where to have more than one person working
// on it
var Migrations = [1]Migration{
	V0Initial,
}
