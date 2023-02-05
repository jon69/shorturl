package dbh

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func Ping(conn string) bool {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Println("Error connect to db: " + err.Error())
		return false
	}
	defer db.Close()
	return true
}

func CreateIfNotExist(conn string) bool {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Println("Error connect to db: " + err.Error())
		return false
	}
	defer db.Close()

	// проверяем есть ли таблица в БД
	var exist bool
	queryCheck := "SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND  tablename  = 'shorturls')"
	row := db.QueryRow(queryCheck)
	err = row.Scan(&exist)
	if err != nil {
		log.Println("Error exec query [" + queryCheck + "]: " + err.Error())
		return false
	}
	// если нет, создаем
	if !exist {
		queryCreate := "CREATE TABLE public.shorturls (uid bigserial, url bytea)"
		_, err = db.Exec(queryCreate)
		if err != nil {
			log.Println("Error exec query [" + queryCreate + "]: " + err.Error())
			return false
		}
	}
	return true
}

func InsertURL(conn string, data []byte) bool {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Println("Error connect to db: " + err.Error())
		return false
	}
	defer db.Close()

	_, err = db.Exec("INSERT INFO public.shorturls (url) vlaues ($1)", data)
	if err != nil {
		log.Println("Error insert value to db: " + err.Error())
		return false
	}
	return true
}

func ReadURLS(conn string) ([][]byte, bool) {
	var ret [][]byte
	db, errOpen := sql.Open("postgres", conn)
	if errOpen != nil {
		log.Println("Error connect to db: " + errOpen.Error())
		return ret, false
	}
	defer db.Close()

	rows, err := db.Query("SELECT url from public.shorturls")
	if err != nil {
		log.Println("Error select url: " + err.Error())
		return ret, false
	}

	// обязательно закрываем перед возвратом функции
	defer rows.Close()

	// пробегаем по всем записям
	for rows.Next() {
		var shortURL []byte
		err = rows.Scan(&shortURL)
		if err != nil {
			log.Println("Error rows.Scan: " + err.Error())
			return ret, false
		}
		ret = append(ret, shortURL)
	}
	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		log.Println("Error rows.Err: " + err.Error())
		return ret, false
	}

	return ret, true
}
