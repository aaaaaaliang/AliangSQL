package main

import (
	"awesomeProject4/storgeengine"
	"awesomeProject4/user"
)

func main() {
	user.Welcome()
	db := storgeengine.NewDB()
	for {
		sql := user.UserInput()

		if !storgeengine.ParseSQL(sql, db) {
			break
		}
	}
}
